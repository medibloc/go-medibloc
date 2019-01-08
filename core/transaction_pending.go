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

var (
	ErrAccountPendingFull = errors.New("account pending full")
	ErrNonceNotAcceptable = errors.New("nonce not acceptable")
	// ErrFailedToReplacePendingTx
)

const (
	MaxPendingByAccount = 64
)

// AccountPendingPool struct manages pending transactions by account.
type AccountPendingPool struct {
	mu sync.Mutex

	all     map[string]*TxContext
	pending map[common.Address]*AccountPending
	payer   map[common.Address]*AccountPayer

	selector   *roundrobin.RoundRobin
	nonceCache map[common.Address]uint64
}

// NewAccountPendingPool creates AccountPendingPool.
func NewAccountPendingPool() *AccountPendingPool {
	return &AccountPendingPool{
		all:        make(map[string]*TxContext),
		pending:    make(map[common.Address]*AccountPending),
		payer:      make(map[common.Address]*AccountPayer),
		selector:   roundrobin.New(),
		nonceCache: make(map[common.Address]uint64),
	}
}

// PushOrReplace pushes or replaces a transaction.
func (pool *AccountPendingPool) PushOrReplace(tx *TxContext, acc *Account, price Price) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pending, exist := pool.pending[tx.From()]
	if exist && pending.isDuplicateNonce(tx.Nonce()) {
		return pool.replace(tx, acc, price)
	}
	return pool.push(tx, acc, price)
}

func (pool *AccountPendingPool) push(tx *TxContext, acc *Account, price Price) error {
	pool.initAccountIfNotExist(tx)
	defer pool.deleteAccountIfEmpty(tx)

	// TODO delete
	// if pending.size() >= MaxPendingByAccount {
	// 	return ErrAccountPendingFull
	// }

	pending := pool.pending[tx.From()]
	if !pending.isAcceptableNonce(tx.Nonce(), acc.Nonce) {
		return ErrNonceNotAcceptable
	}

	payer := pool.payer[tx.Payer()]
	if err := payer.checkPointAvailable(tx, acc, price); err != nil {
		return err
	}

	pending.add(tx)
	payer.addOrReplace(tx)
	pool.all[tx.HexHash()] = tx

	pool.selector.Include(tx.From().Hex())
	return nil
}

func (pool *AccountPendingPool) replace(tx *TxContext, acc *Account, price Price) error {
	pool.initAccountIfNotExist(tx)
	defer pool.deleteAccountIfEmpty(tx)

	pending := pool.pending[tx.From()]
	if !pending.isDuplicateNonce(tx.Nonce()) {
		return ErrFailedToReplacePendingTx
	}
	if !pending.isReplaceDurationElapsed(tx.Nonce()) {
		return ErrFailedToReplacePendingTx
	}

	payer := pool.payer[tx.Payer()]
	if err := payer.checkPointAvailable(tx, acc, price); err != nil {
		return err
	}

	oldTx := pending.replace(tx)
	payer.addOrReplace(tx)

	// TODO comment
	if oldTx.Payer().Equals(tx.Payer()) {
		return nil
	}

	oldPayer := pool.payer[oldTx.Payer()]
	oldPayer.remove(oldTx)
	if oldPayer.size() == 0 {
		delete(pool.payer, oldTx.Payer())
	}

	delete(pool.all, oldTx.HexHash())
	pool.all[tx.HexHash()] = tx

	return nil
}

// TODO Refactoring
func (pool *AccountPendingPool) initAccountIfNotExist(tx *TxContext) {
	_, exist := pool.pending[tx.From()]
	if !exist {
		pending := newAccountPending(tx.From())
		pool.pending[tx.From()] = pending
	}

	_, exist = pool.payer[tx.Payer()]
	if !exist {
		payer := newAccountPayer(tx.Payer())
		pool.payer[tx.Payer()] = payer
	}

}

func (pool *AccountPendingPool) deleteAccountIfEmpty(tx *TxContext) {
	pending, exist := pool.pending[tx.From()]
	if exist && pending.size() == 0 {
		delete(pool.pending, tx.From())
		pool.selector.Remove(tx.From().Hex())
	}

	payer, exist := pool.payer[tx.Payer()]
	if exist && payer.size() == 0 {
		delete(pool.payer, tx.Payer())
	}
}

// Get gets a transaction.
func (pool *AccountPendingPool) Get(hash []byte) *Transaction {
	txc, exist := pool.all[byteutils.Bytes2Hex(hash)]
	if !exist {
		return nil
	}
	return txc.Transaction
}

// Remove removes a transaction.
func (pool *AccountPendingPool) Remove(tx *TxContext) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pending, exist := pool.pending[tx.From()]
	if !exist {
		return ErrNotFound
	}
	pending.remove(tx)
	if pending.size() == 0 {
		delete(pool.pending, tx.From())
		pool.selector.Remove(tx.From().Hex())
	}

	delete(pool.all, tx.HexHash())

	payer, exist := pool.payer[tx.Payer()]
	if !exist {
		return ErrNotFound
	}
	payer.remove(tx)
	if payer.size() == 0 {
		delete(pool.payer, tx.Payer())
	}
	return nil
}

// Prune prunes transactions by account's current nonce.
func (pool *AccountPendingPool) Prune(addr common.Address, nonceLowerLimit uint64) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pending, exist := pool.pending[addr]
	if !exist {
		return
	}

	removed := pending.prune(nonceLowerLimit)
	if pending.size() == 0 {
		delete(pool.pending, addr)
		pool.selector.Remove(addr.Hex())
	}
	for _, tx := range removed {
		delete(pool.all, tx.HexHash())
		payer := pool.payer[tx.Payer()]
		payer.remove(tx)
		if payer.size() == 0 {
			delete(pool.payer, tx.Payer())
		}
	}
}

func (pool *AccountPendingPool) NonceUpperLimit(acc *Account) uint64 {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pending, exist := pool.pending[acc.Address]
	if !exist {
		return acc.Nonce + 1
	}

	upperLimit := acc.Nonce + MaxPendingByAccount
	if pending.maxNonce+1 < upperLimit {
		upperLimit = pending.maxNonce + 1
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

		pending := pool.pending[addr]

		requiredNonce, exist := pool.nonceCache[addr]
		if !exist {
			tx := pending.nonceToTx[pending.minNonce]
			return tx.Transaction
		}

		tx, exist := pending.nonceToTx[requiredNonce]
		if !exist {
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

// AccountPending manages account's pending transactions.
type AccountPending struct {
	addr      common.Address
	minNonce  uint64
	maxNonce  uint64
	nonceToTx map[uint64]*TxContext
	hashToTx  map[string]*TxContext
}

func newAccountPending(addr common.Address) *AccountPending {
	return &AccountPending{
		addr:      addr,
		minNonce:  0,
		maxNonce:  0,
		nonceToTx: make(map[uint64]*TxContext),
		hashToTx:  make(map[string]*TxContext),
	}
}

func (pend *AccountPending) size() int {
	return len(pend.nonceToTx)
}

func (pend *AccountPending) isDuplicateNonce(nonce uint64) bool {
	_, exist := pend.nonceToTx[nonce]
	return exist
}

func (pend *AccountPending) isAcceptableNonce(nonce uint64, accNonce uint64) bool {
	if pend.size() == 0 {
		return accNonce+1 == nonce
	}

	lowerLimit := accNonce
	upperLimit := accNonce + MaxPendingByAccount
	if pend.maxNonce+1 < upperLimit {
		upperLimit = pend.maxNonce + 1
	}
	return lowerLimit < nonce && nonce <= upperLimit
}

func (pend *AccountPending) isReplaceDurationElapsed(nonce uint64) bool {
	old, exist := pend.nonceToTx[nonce]
	if !exist {
		return false
	}
	return old.incomeTime.Add(AllowReplacePendingDuration).Before(time.Now())
}

func (pend *AccountPending) add(tx *TxContext) {
	if pend.size() == 0 {
		pend.minNonce = tx.Nonce()
		pend.maxNonce = tx.Nonce()
	}
	if pend.minNonce > tx.Nonce() {
		pend.minNonce = tx.Nonce()
	}
	if pend.maxNonce < tx.Nonce() {
		pend.maxNonce = tx.Nonce()
	}
	pend.nonceToTx[tx.Nonce()] = tx
	pend.hashToTx[tx.HexHash()] = tx
}

func (pend *AccountPending) replace(tx *TxContext) (existing *TxContext) {
	old := pend.nonceToTx[tx.Nonce()]
	delete(pend.hashToTx, old.HexHash())

	pend.nonceToTx[tx.Nonce()] = tx
	pend.hashToTx[tx.HexHash()] = tx

	return old
}

func (pend *AccountPending) remove(tx *TxContext) {
	delete(pend.nonceToTx, tx.Nonce())
	delete(pend.hashToTx, tx.HexHash())
	if pend.minNonce == tx.Nonce() {
		pend.minNonce++
	}
	if pend.maxNonce == tx.Nonce() {
		pend.maxNonce--
	}
}

func (pend *AccountPending) prune(targetNonce uint64) (removed []*TxContext) {
	if pend.size() == 0 {
		return nil
	}

	upperLimit := targetNonce
	if pend.maxNonce < upperLimit {
		upperLimit = pend.maxNonce
	}
	for i := pend.minNonce; i <= upperLimit; i++ {
		tx, exist := pend.nonceToTx[i]
		if !exist {
			continue
		}
		delete(pend.nonceToTx, tx.Nonce())
		delete(pend.hashToTx, tx.HexHash())
		pend.minNonce = i + 1
		removed = append(removed, tx)
	}
	return removed
}

// AccountPayer manages payers bandwidth.
type AccountPayer struct {
	addr  common.Address
	bw    *Bandwidth
	txMap map[string]*TxContext
}

func newAccountPayer(payer common.Address) *AccountPayer {
	return &AccountPayer{
		addr:  payer,
		bw:    NewBandwidth(0, 0),
		txMap: make(map[string]*TxContext),
	}
}

func (payer *AccountPayer) size() int {
	return len(payer.txMap)
}

func (payer *AccountPayer) checkPointAvailable(tx *TxContext, acc *Account, price Price) error {
	bw := payer.bw.Clone()

	key := addrNonceKey(tx)
	old, exist := payer.txMap[key]
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

	// TODO Refactoring
	err = acc.UpdatePoints(time.Now().Unix())
	if err != nil {
		return err
	}
	if err := acc.checkAccountPoints(tx.Transaction, points); err != nil {
		return err
	}

	return nil
}

func addrNonceKey(tx *TxContext) string {
	return fmt.Sprintf("%s-%d", tx.From().Hex(), tx.Nonce())
}

func (payer *AccountPayer) addOrReplace(tx *TxContext) (existing *TxContext) {
	key := addrNonceKey(tx)
	old, exist := payer.txMap[key]
	if exist {
		payer.bw.Sub(NewBandwidth(old.exec.Bandwidth()))
	}
	payer.bw.Add(NewBandwidth(tx.exec.Bandwidth()))
	payer.txMap[key] = tx
	return old
}

func (payer *AccountPayer) remove(tx *TxContext) {
	key := addrNonceKey(tx)
	old, exist := payer.txMap[key]
	if !exist {
		return
	}
	payer.bw.Sub(NewBandwidth(old.exec.Bandwidth()))
	delete(payer.txMap, key)
}
