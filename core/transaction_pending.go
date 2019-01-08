package core

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/roundrobin"
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
	MaxPendingByAccount = 64
)

// AccountPendingPool struct manages pending transactions by account.
type AccountPendingPool struct {
	mu sync.Mutex

	all   map[string]*TxContext
	from  map[common.Address]*AccountFrom
	payer map[common.Address]*AccountPayer

	selector   *roundrobin.RoundRobin
	nonceCache map[common.Address]uint64
}

// NewAccountPendingPool creates AccountPendingPool.
func NewAccountPendingPool() *AccountPendingPool {
	return &AccountPendingPool{
		all:        make(map[string]*TxContext),
		from:       make(map[common.Address]*AccountFrom),
		payer:      make(map[common.Address]*AccountPayer),
		selector:   roundrobin.New(),
		nonceCache: make(map[common.Address]uint64),
	}
}

// PushOrReplace pushes or replaces a transaction.
func (pool *AccountPendingPool) PushOrReplace(tx *TxContext, acc *Account, price Price) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	accFrom, exist := pool.from[tx.From()]
	if exist && accFrom.isDuplicateNonce(tx.Nonce()) {
		return pool.replace(tx, acc, price)
	}
	return pool.push(tx, acc, price)
}

func (pool *AccountPendingPool) push(tx *TxContext, acc *Account, price Price) error {
	accFrom, exist := pool.from[tx.From()]
	if !exist {
		accFrom = newAccountFrom(tx.From())
	}
	if !accFrom.isAcceptableNonce(tx.Nonce(), acc.Nonce) {
		return ErrNonceNotAcceptable
	}

	accPayer, exist := pool.payer[tx.Payer()]
	if !exist {
		accPayer = newAccountPayer(tx.Payer())
	}
	if err := accPayer.checkPointAvailable(tx, acc, price); err != nil {
		return err
	}

	accFrom.set(tx)
	accPayer.set(tx)

	pool.all[tx.HexHash()] = tx
	pool.from[tx.From()] = accFrom
	pool.payer[tx.Payer()] = accPayer

	pool.selector.Include(tx.From().Hex())
	return nil
}

func (pool *AccountPendingPool) replace(tx *TxContext, acc *Account, price Price) error {
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

	accPayer, exist := pool.payer[tx.Payer()]
	if !exist {
		accPayer = newAccountPayer(tx.Payer())
	}
	if err := accPayer.checkPointAvailable(tx, acc, price); err != nil {
		return err
	}

	oldTx := accFrom.remove(tx.Nonce())
	oldPayer := pool.payer[oldTx.Payer()]
	oldPayer.remove(oldTx)
	if oldPayer.size() == 0 {
		delete(pool.payer, oldTx.Payer())
	}
	delete(pool.all, oldTx.HexHash())

	accFrom.set(tx)
	accPayer.set(tx)

	pool.all[tx.HexHash()] = tx
	pool.from[tx.From()] = accFrom
	pool.payer[tx.Payer()] = accPayer

	return nil
}

// Get gets a transaction.
func (pool *AccountPendingPool) Get(hash []byte) *Transaction {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	txc, exist := pool.all[byteutils.Bytes2Hex(hash)]
	if !exist {
		return nil
	}
	return txc.Transaction
}

// TODO double check nonceLowerLimit
// Prune prunes transactions by account's current nonce.
func (pool *AccountPendingPool) Prune(addr common.Address, nonceLowerLimit uint64) {
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
		accFrom.remove(tx.Nonce())
		accPayer := pool.payer[tx.Payer()]
		accPayer.remove(tx)
		if accPayer.size() == 0 {
			delete(pool.payer, tx.Payer())
		}
		delete(pool.all, tx.HexHash())
	}

	if accFrom.size() == 0 {
		delete(pool.from, addr)
		pool.selector.Remove(addr.Hex())
	}
}

func (pool *AccountPendingPool) NonceUpperLimit(acc *Account) uint64 {
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

func (pool *AccountPendingPool) Next() *Transaction {
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

func (pool *AccountPendingPool) SetRequiredNonce(addr common.Address, nonce uint64) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pool.nonceCache[addr] = nonce
}

func (pool *AccountPendingPool) ResetSelector() {
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
	bw            *Bandwidth
	addrNonceToTx map[string]*TxContext
}

func newAccountPayer(payer common.Address) *AccountPayer {
	return &AccountPayer{
		addr:          payer,
		bw:            NewBandwidth(0, 0),
		addrNonceToTx: make(map[string]*TxContext),
	}
}

func (ap *AccountPayer) size() int {
	return len(ap.addrNonceToTx)
}

func (ap *AccountPayer) checkPointAvailable(tx *TxContext, acc *Account, price Price) error {
	bw := ap.bw.Clone()

	key := addrNonceKey(tx)
	old, exist := ap.addrNonceToTx[key]
	if exist {
		bw.Sub(NewBandwidth(old.exec.Bandwidth()))
	}
	bw.Add(NewBandwidth(tx.exec.Bandwidth()))
	points, err := bw.CalcPoints(price)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to calculate points")
		return err
	}

	// TODO Refactor when merging
	err = acc.UpdatePoints(time.Now().Unix())
	if err != nil {
		return err
	}
	if err := acc.checkAccountPoints(tx.Transaction, points); err != nil {
		return err
	}

	return nil
}

func (ap *AccountPayer) set(tx *TxContext) (evicted *TxContext) {
	key := addrNonceKey(tx)
	old, exist := ap.addrNonceToTx[key]
	if exist {
		ap.bw.Sub(NewBandwidth(old.exec.Bandwidth()))
	}
	ap.bw.Add(NewBandwidth(tx.exec.Bandwidth()))
	ap.addrNonceToTx[key] = tx
	return old
}

func (ap *AccountPayer) remove(tx *TxContext) {
	key := addrNonceKey(tx)
	old, exist := ap.addrNonceToTx[key]
	if !exist {
		return
	}
	ap.bw.Sub(NewBandwidth(old.exec.Bandwidth()))
	delete(ap.addrNonceToTx, key)
}

func addrNonceKey(tx *TxContext) string {
	return fmt.Sprintf("%s-%d", tx.From().Hex(), tx.Nonce())
}
