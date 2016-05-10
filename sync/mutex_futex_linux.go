// Copyright 2016 Aleksandr Demakin. All rights reserved.

package sync

import (
	"os"
	"sync/atomic"
	"time"

	"bitbucket.org/avd/go-ipc"
	"bitbucket.org/avd/go-ipc/internal/common"

	"golang.org/x/sys/unix"
)

const (
	cFutexMutexUnlocked          = 0
	cFutexMutexLockedNoWaiters   = 1
	cFutexMutexLockedHaveWaiters = 2
)

type FutexMutex struct {
	futex *Futex
}

// FutexMutex creates a new futex-based mutex.
// name - object name.
// mode - object creation mode. must be one of the following:
//		O_CREATE_ONLY
//		O_OPEN_ONLY
//		O_OPEN_OR_CREATE
//	perm - file's mode and permission bits.
func NewFutexMutex(name string, mode int, perm os.FileMode) (*FutexMutex, error) {
	futex, err := NewFutex(name, mode, perm, cFutexMutexUnlocked, 0)
	if err != nil {
		return nil, err
	}
	return &FutexMutex{futex: futex}, nil
}

// Lock locks the mutex. It panics on an error.
func (f *FutexMutex) Lock() {
	if err := f.lockTimeout(-1); err != nil {
		panic(err)
	}
}

// LockTimeout tries to lock the locker, waiting for not more, than timeout.
func (f *FutexMutex) LockTimeout(timeout time.Duration) bool {
	err := f.lockTimeout(timeout)
	if err == nil {
		return true
	}
	if common.IsTimeoutErr(err) {
		return false
	}
	panic(err)
}

// Unlock releases the mutex. It panics on an error.
func (f *FutexMutex) Unlock() {
	addr := f.futex.Addr()
	if !atomic.CompareAndSwapUint32(addr, cFutexMutexLockedNoWaiters, cFutexMutexUnlocked) {
		*addr = 0
		if _, err := f.futex.Wake(1); err != nil {
			panic(err)
		}
	}
}

// Close indicates, that the object is no longer in use,
// and that the underlying resources can be freed.
func (f *FutexMutex) Close() error {
	return f.futex.Close()
}

// Destroy removes the mutex object.
func (f *FutexMutex) Destroy() error {
	return f.futex.Destroy()
}

// DestroyFutexMutex permanently removes mutex with the given name.
func DestroyFutexMutex(name string) error {
	m, err := NewFutexMutex(name, ipc.O_OPEN_ONLY, 0666)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return err
	}
	return m.Destroy()
}

func (f *FutexMutex) lockTimeout(timeout time.Duration) error {
	addr := f.futex.Addr()
	if !atomic.CompareAndSwapUint32(addr, cFutexMutexUnlocked, cFutexMutexLockedNoWaiters) {
		old := atomic.LoadUint32(addr)
		if old != cFutexMutexLockedHaveWaiters {
			old = atomic.SwapUint32(addr, cFutexMutexLockedHaveWaiters)
		}
		for old != cFutexMutexUnlocked {
			if err := f.futex.Wait(cFutexMutexLockedHaveWaiters, timeout); err != nil {
				if !common.SyscallErrHasCode(err, unix.EWOULDBLOCK) {
					return err
				}
			}
			old = atomic.SwapUint32(addr, cFutexMutexLockedHaveWaiters)
		}
	}
	return nil
}
