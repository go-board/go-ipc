// Copyright 2016 Aleksandr Demakin. All rights reserved.

// +build linux freebsd

package sync

import (
	"os"
	"testing"
)

func sysvMutexCtor(name string, flag int, perm os.FileMode) (IPCLocker, error) {
	return NewSemaMutex(name, flag, perm)
}

func sysvMutexDtor(name string) error {
	return DestroySemaMutex(name)
}

func TestSysvMutexOpenMode(t *testing.T) {
	testLockerOpenMode(t, sysvMutexCtor, sysvMutexDtor)
}

func TestSysvMutexOpenMode2(t *testing.T) {
	testLockerOpenMode2(t, sysvMutexCtor, sysvMutexDtor)
}

func TestSysvMutexOpenMode3(t *testing.T) {
	testLockerOpenMode3(t, sysvMutexCtor, sysvMutexDtor)
}

func TestSysvMutexOpenMode4(t *testing.T) {
	testLockerOpenMode4(t, sysvMutexCtor, sysvMutexDtor)
}

func TestSysvMutexOpenMode5(t *testing.T) {
	testLockerOpenMode5(t, sysvMutexCtor, sysvMutexDtor)
}

func TestSysvMutexLock(t *testing.T) {
	testLockerLock(t, sysvMutexCtor, sysvMutexDtor)
}

func TestSysvMutexMemory(t *testing.T) {
	testLockerMemory(t, "msysv", false, sysvMutexCtor, sysvMutexDtor)
}

func TestSysvMutexValueInc(t *testing.T) {
	testLockerValueInc(t, "msysv", sysvMutexCtor, sysvMutexDtor)
}

func TestSysvMutexLockTimeout(t *testing.T) {
	testLockerLockTimeout(t, "msysv", sysvMutexCtor, sysvMutexDtor)
}

func TestSysvMutexLockTimeout2(t *testing.T) {
	testLockerLockTimeout2(t, "msysv", sysvMutexCtor, sysvMutexDtor)
}

func TestSysvMutexPanicsOnDoubleUnlock(t *testing.T) {
	testLockerTwiceUnlock(t, sysvMutexCtor, sysvMutexDtor)
}
