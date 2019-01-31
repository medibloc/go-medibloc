package core

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/roundrobin"
	corestate "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// TODO move error code
var (
	ErrNonceNotAcceptable = errors.New("nonce not acceptable")
	// ErrFailedToReplacePendingTx
)

const (
	// MaxPendingByAccount is the max number of pending transactions per account
	MaxPendingByAccount = 64
)

// PendingTransactionPool struct manages pending transactions by account.
type PendingTransactionPool struct {
	mu sync.Mutex

	all   map[string]*TxContext
	from  map[common.Address]*AccountFrom
	payer map[common.Address]*AccountPayer

	selector   *roundrobin.RoundRobin
	nonceCache map[common.Address]uint64
}

// NewPendingTransactionPool creates PendingTransactionPool.
func NewPendingTransactionPool() *PendingTransactionPool {
	return &PendingTransactionPool{
		all:        make(map[string]*TxContext),
		from:       make(map[common.Address]*AccountFrom),
		payer:      make(map[common.Address]*AccountPayer),
		selector:   roundrobin.New(),
		nonceCache: make(map[common.Address]uint64),
	}
}

func (pool *PendingTransactionPool) Len() int {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	return len(pool.all)
}

// PushOrReplace pushes or replaces a transaction.
func (pool *PendingTransactionPool) PushOrReplace(tx *TxContext, acc *corestate.Account, price common.Price) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	if _, exist := pool.all[tx.HexHash()]; exist {
		return ErrDuplicatedTransaction
	}

	accFrom, exist := pool.from[tx.From()]
	if exist && accFrom.isDuplicateNonce(tx.Nonce()) {
		return pool.replace(tx, acc, price)
	}
	return pool.push(tx, acc, price)
}

func (pool *PendingTransactionPool) push(tx *TxContext, acc *corestate.Account, price common.Price) error {
	accFrom, exist := pool.from[tx.From()]
	if !exist {
		accFrom = newAccountFrom(tx.From())
	}
	if !accFrom.isAcceptableNonce(tx.Nonce(), acc.Nonce) {
		return ErrNonceNotAcceptable
	}

	accPayer, exist := pool.payer[tx.PayerOrFrom()]
	if !exist {
		accPayer = newAccountPayer(tx.PayerOrFrom())
	}
	if err := accPayer.checkAvailablePoint(tx, acc, price); err != nil {
		return err
	}

	accFrom.set(tx)
	accPayer.set(tx)

	pool.all[tx.HexHash()] = tx
	pool.from[tx.From()] = accFrom
	pool.payer[tx.PayerOrFrom()] = accPayer

	pool.selector.Include(tx.From().Hex())
	return nil
}

func (pool *PendingTransactionPool) replace(tx *TxContext, acc *corestate.Account, price common.Price) error {
	accFrom, exist := pool.from[tx.From()]
	if !exist {
		accFrom = newAccountFrom(tx.From())
	}
	if !accFrom.isDuplicateNonce(tx.Nonce()) {
		return ErrFailedToReplacePendingTx
	}
	if !accFrom.isReplaceDurationElapsed(tx.Nonce()) {
		return ErrFailedToReplacePendingTx
	}

	accPayer, exist := pool.payer[tx.PayerOrFrom()]
	if !exist {
		accPayer = newAccountPayer(tx.PayerOrFrom())
	}
	if err := accPayer.checkAvailablePoint(tx, acc, price); err != nil {
		return err
	}

	oldTx := accFrom.remove(tx.Nonce())
	oldPayer := pool.payer[oldTx.PayerOrFrom()]
	oldPayer.remove(oldTx)
	if oldPayer.size() == 0 {
		delete(pool.payer, oldTx.PayerOrFrom())
	}
	delete(pool.all, oldTx.HexHash())

	accFrom.set(tx)
	accPayer.set(tx)

	pool.all[tx.HexHash()] = tx
	pool.from[tx.From()] = accFrom
	pool.payer[tx.PayerOrFrom()] = accPayer

	return nil
}

// Get gets a transaction.
func (pool *PendingTransactionPool) Get(hash []byte) *Transaction {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	txc, exist := pool.all[byteutils.Bytes2Hex(hash)]
	if !exist {
		return nil
	}
	return txc.Transaction
}

// Prune prunes transactions by account's current nonce.
// TODO double check nonceLowerLimit
func (pool *PendingTransactionPool) Prune(addr common.Address, nonceLowerLimit uint64) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	accFrom, exist := pool.from[addr]
	if !exist {
		return
	}

	for {
		tx := accFrom.peekFirst()
		if tx == nil {
			break
		}
		if tx.Nonce() > nonceLowerLimit {
			break
		}

		accFrom.remove(tx.Nonce())
		accPayer := pool.payer[tx.PayerOrFrom()]
		accPayer.remove(tx)
		if accPayer.size() == 0 {
			delete(pool.payer, tx.PayerOrFrom())
		}
		delete(pool.all, tx.HexHash())
	}

	if accFrom.size() == 0 {
		delete(pool.from, addr)
		pool.selector.Remove(addr.Hex())
	}
}

// NonceUpperLimit returns the maximum nonce of transactions that can be accepted in the pool.
func (pool *PendingTransactionPool) NonceUpperLimit(acc *corestate.Account) uint64 {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	accFrom, exist := pool.from[acc.Address]
	if !exist {
		return acc.Nonce + 1
	}

	upperLimit := acc.Nonce + MaxPendingByAccount
	if accFrom.maxNonce+1 < upperLimit {
		upperLimit = accFrom.maxNonce + 1
	}

	return upperLimit
}

// Next returns a transaction to process in round-robin order
func (pool *PendingTransactionPool) Next() *Transaction {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	for pool.selector.HasNext() {
		addrStr := pool.selector.Next()
		addr, err := common.HexToAddress(addrStr)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
			}).Error("Failed to convert hex to address.")
			return nil
		}

		accFrom := pool.from[addr]

		requiredNonce, exist := pool.nonceCache[addr]
		if !exist {
			tx := accFrom.peekFirst()
			if tx == nil {
				return nil
			}
			return tx.Transaction
		}

		tx := accFrom.get(requiredNonce)
		if tx == nil {
			pool.selector.Exclude(addrStr)
			continue
		}
		return tx.Transaction
	}
	return nil
}

// SetRequiredNonce sets the transaction's nonce for the next execution by address.
func (pool *PendingTransactionPool) SetRequiredNonce(addr common.Address, nonce uint64) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pool.nonceCache[addr] = nonce
}

// ResetSelector resets transaction selector.
func (pool *PendingTransactionPool) ResetSelector() {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pool.nonceCache = make(map[common.Address]uint64)
	pool.selector.Reset()
}

// AccountFrom manages account's pending transactions.
type AccountFrom struct {
	addr      common.Address
	minNonce  uint64
	maxNonce  uint64
	nonceToTx map[uint64]*TxContext
}

func newAccountFrom(addr common.Address) *AccountFrom {
	return &AccountFrom{
		addr:      addr,
		minNonce:  0,
		maxNonce:  0,
		nonceToTx: make(map[uint64]*TxContext),
	}
}

func (af *AccountFrom) size() int {
	return len(af.nonceToTx)
}

func (af *AccountFrom) isDuplicateNonce(nonce uint64) bool {
	_, exist := af.nonceToTx[nonce]
	return exist
}

func (af *AccountFrom) isAcceptableNonce(nonce uint64, accNonce uint64) bool {
	if af.size() == 0 {
		return accNonce+1 == nonce
	}

	lowerLimit := accNonce
	upperLimit := accNonce + MaxPendingByAccount
	if af.maxNonce+1 < upperLimit {
		upperLimit = af.maxNonce + 1
	}
	return lowerLimit < nonce && nonce <= upperLimit
}

func (af *AccountFrom) isReplaceDurationElapsed(nonce uint64) bool {
	old, exist := af.nonceToTx[nonce]
	if !exist {
		return false
	}
	return old.incomeTime.Add(AllowReplacePendingDuration).Before(time.Now())
}

func (af *AccountFrom) get(nonce uint64) *TxContext {
	return af.nonceToTx[nonce]
}

func (af *AccountFrom) set(tx *TxContext) (evicted *TxContext) {
	old := af.nonceToTx[tx.Nonce()]
	if af.size() == 0 {
		af.minNonce = tx.Nonce()
		af.maxNonce = tx.Nonce()
	}
	if af.minNonce > tx.Nonce() {
		af.minNonce = tx.Nonce()
	}
	if af.maxNonce < tx.Nonce() {
		af.maxNonce = tx.Nonce()
	}
	af.nonceToTx[tx.Nonce()] = tx
	return old
}

func (af *AccountFrom) remove(nonce uint64) (removed *TxContext) {
	old, exist := af.nonceToTx[nonce]
	if !exist {
		return nil
	}
	delete(af.nonceToTx, nonce)
	if af.minNonce == nonce {
		af.minNonce++
	}
	if af.maxNonce == nonce {
		af.maxNonce--
	}
	return old
}

func (af *AccountFrom) peekFirst() *TxContext {
	if af.size() == 0 {
		return nil
	}
	for i := af.minNonce; i <= af.maxNonce; i++ {
		tx, exist := af.nonceToTx[i]
		if exist {
			return tx
		}
	}
	return nil
}

// AccountPayer manages payers bandwidth.
type AccountPayer struct {
	addr          common.Address
	bw            *common.Bandwidth
	addrNonceToTx map[string]*TxContext
}

func newAccountPayer(payer common.Address) *AccountPayer {
	return &AccountPayer{
		addr:          payer,
		bw:            common.NewBandwidth(0, 0),
		addrNonceToTx: make(map[string]*TxContext),
	}
}

func (ap *AccountPayer) size() int {
	return len(ap.addrNonceToTx)
}

func (ap *AccountPayer) checkAvailablePoint(tx *TxContext, acc *corestate.Account, price common.Price) error {
	usage, err := ap.requiredPointUsage(tx, price)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to get required point")
		return err
	}

	return checkFromAccountPoint(acc, tx.exec, usage)
}

func (ap *AccountPayer) requiredPointUsage(tx *TxContext, price common.Price) (point *util.Uint128, err error) {
	bw := ap.bw.Clone()
	key := addrNonceKey(tx)
	old, exist := ap.addrNonceToTx[key]
	if exist {
		bw.Sub(old.exec.Bandwidth())
	}
	bw.Add(tx.exec.Bandwidth())
	return bw.CalcPoints(price)
}

func (ap *AccountPayer) set(tx *TxContext) (evicted *TxContext) {
	key := addrNonceKey(tx)
	old, exist := ap.addrNonceToTx[key]
	if exist {
		ap.bw.Sub(old.exec.Bandwidth())
	}
	ap.bw.Add(tx.exec.Bandwidth())
	ap.addrNonceToTx[key] = tx
	return old
}

func (ap *AccountPayer) remove(tx *TxContext) {
	key := addrNonceKey(tx)
	old, exist := ap.addrNonceToTx[key]
	if !exist {
		return
	}
	ap.bw.Sub(old.exec.Bandwidth())
	delete(ap.addrNonceToTx, key)
}

func addrNonceKey(tx *TxContext) string {
	return fmt.Sprintf("%s-%d", tx.From().Hex(), tx.Nonce())
}
