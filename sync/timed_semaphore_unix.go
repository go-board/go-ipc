// Copyright 2015 Aleksandr Demakin. All rights reserved.

// +build linux,amd64

package sync

import (
	"time"

	"bitbucket.org/avd/go-ipc/internal/common"
)

// AddTimeout add the given value to the semaphore's value.
// If the operation locks, it waits for not more, than timeout.
func (s *Semaphore) AddTimeout(timeout time.Duration, value int) error {
	f := func(curTimeout time.Duration) error {
		b := sembuf{semnum: 0, semop: int16(value), semflg: 0}
		return semtimedop(s.id, []sembuf{b}, common.TimeoutToTimeSpec(curTimeout))
	}
	return common.UninterruptedSyscallTimeout(f, timeout)
}

// LockTimeout tries to lock the locker, waiting for not more, than timeout.
func (m *SemaMutex) LockTimeout(timeout time.Duration) bool {
	err := m.s.AddTimeout(timeout, -1)
	if err == nil {
		return true
	}
	if common.IsTimeoutErr(err) {
		return false
	}
	panic(err)
}