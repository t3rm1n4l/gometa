package common

import (
	"fmt"
	"sync"
)

type Txnid uint64

var gEpoch uint64 = 0
var gCounter uint64 = 0
var gMutex sync.Mutex
var gCurTxnid Txnid = 0

func GetNextTxnId() Txnid {
	gMutex.Lock()
	defer gMutex.Unlock()

	// Increment the epoch. If the counter overflows, panic.
	if gCounter == uint64(MAX_COUNTER) {
		panic(fmt.Sprintf("Counter overflows for epoch %d", gEpoch))
	}
	gCounter++

	epoch := uint64(gEpoch << 32)
	newTxnid := Txnid(epoch + gCounter)

	// gCurTxnid is initialized using the LastLoggedTxid in the local repository.  So if this node becomes master,
	// we want to make sure that the new txid is larger than the one that we saw before.
	if gCurTxnid >= newTxnid {
		// Assertion.  This is to ensure integrity of the system.  Wrong txnid can result in corruption.
		panic(fmt.Sprintf("GetNextTxnId(): Assertion: New Txnid %d is smaller than or equal to old txnid %d", newTxnid, gCurTxnid))
	}

	gCurTxnid = newTxnid

	return gCurTxnid
}

// Return true if txid2 is logically next in sequence from txid1.
// If txid2 and txid1 have different epoch, then only check if
// txid2 has a larger epoch.  Otherwise, compare the counter such
// that txid2 is txid1 + 1
func IsNextInSequence(new, old Txnid) bool {

	if new.GetEpoch() > old.GetEpoch() {
		return true
	}

	if new.GetEpoch() == old.GetEpoch() &&
		uint32(old.GetCounter()) != MAX_COUNTER &&
		new == old+1 {
		return true
	}

	return false
}

func SetEpoch(newEpoch uint32) {
	gMutex.Lock()
	defer gMutex.Unlock()

	if gEpoch >= uint64(newEpoch) {
		// Assertion.  This is to ensure integrity of the system.  We do not support epoch rollover yet.
		panic(fmt.Sprintf("SetEpoch(): Assertion: New Epoch %d is smaller than or equal to old epoch %d", newEpoch, gEpoch))
	}

	gEpoch = uint64(newEpoch)
	gCounter = 0
}

func InitCurrentTxnid(txnid Txnid) {
	gMutex.Lock()
	defer gMutex.Unlock()

	if txnid > gCurTxnid {
		gCurTxnid = txnid
	}
}

func (id Txnid) GetEpoch() uint64 {
	v := uint64(id)
	return (v >> 32)
}

func (id Txnid) GetCounter() uint64 {
	v := uint32(id)
	return uint64(v)
}

//
// Compare function to compare epoch1 with epoch2
//
// return common.EQUAL if epoch1 is the same as epoch2
// return common.MORE_RECENT if epoch1 is more recent
// return common.LESS_RECENT if epoch1 is less recent
//
// This is just to prepare in the future if we support
// rolling over the epoch (but it will also require
// changes to comparing txnid as well).
//
func CompareEpoch(epoch1, epoch2 uint32) CompareResult {

	if epoch1 == epoch2 {
		return EQUAL
	}

	if epoch1 > epoch2 {
		return MORE_RECENT
	}

	return LESS_RECENT
}

//
// Compare epoch1 and epoch2.  If epoch1 is equal or more recent, return
// the next more recent epoch value.   If epoch1 is less recent than
// epoch2, return epoch2 as it is.
//
func CompareAndIncrementEpoch(epoch1, epoch2 uint32) uint32 {

	result := CompareEpoch(epoch1, epoch2)
	if result == MORE_RECENT || result == EQUAL {
		if epoch1 != MAX_EPOCH {
			return epoch1 + 1
		}

		// TODO : Epoch has reached the max value. If we have a leader
		// election every second, it will take 135 years to overflow the epoch (32 bits).
		// Regardless, we should gracefully roll over the epoch eventually in our
		// implementation.
		panic("epoch limit is reached")
	}

	return epoch2
}