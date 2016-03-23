// Copyright 2016 Aleksandr Demakin. All rights reserved.

package ipc

import (
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"

	"bitbucket.org/avd/go-ipc/internal/allocator"
	"bitbucket.org/avd/go-ipc/internal/test"

	"github.com/stretchr/testify/assert"
)

const (
	testMqName = "go-ipc.mq"
)

type mqCtor func(name string, perm os.FileMode) (Messenger, error)
type mqOpener func(name string, flags int) (Messenger, error)
type mqDtor func(name string) error

func testCreateMq(t *testing.T, ctor mqCtor, dtor mqDtor) {
	a := assert.New(t)
	if dtor != nil {
		a.NoError(dtor(testMqName))
	}
	mq, err := ctor(testMqName, 0666)
	if a.NoError(err) {
		if dtor != nil {
			a.NoError(dtor(testMqName))
		} else {
			a.NoError(mq.Close())
		}
	}
}

func testCreateMqExcl(t *testing.T, ctor mqCtor, dtor mqDtor) {
	a := assert.New(t)
	if dtor != nil {
		a.NoError(dtor(testMqName))
	}
	mq, err := ctor(testMqName, 0666)
	if !a.NoError(err) || !a.NotNil(mq) {
		return
	}
	_, err = ctor(testMqName, 0666)
	a.Error(err)
	if d, ok := mq.(Destroyer); ok {
		a.NoError(d.Destroy())
	} else {
		a.NoError(mq.Close())
	}
}

func testCreateMqInvalidPerm(t *testing.T, ctor mqCtor, dtor mqDtor) {
	a := assert.New(t)
	if dtor != nil {
		a.NoError(dtor(testMqName))
	}
	_, err := ctor(testMqName, 0777)
	a.Error(err)
}

func testOpenMq(t *testing.T, ctor mqCtor, opener mqOpener, dtor mqDtor) {
	a := assert.New(t)
	if dtor != nil {
		a.NoError(dtor(testMqName))
	}
	mq, err := ctor(testMqName, 0666)
	if !a.NoError(err) {
		return
	}
	if dtor != nil {
		a.NoError(dtor(testMqName))
	} else {
		a.NoError(mq.Close())
	}
	_, err = opener(testMqName, O_READ_ONLY)
	a.Error(err)
}

func testMqSendIntSameProcess(t *testing.T, ctor mqCtor, opener mqOpener, dtor mqDtor) {
	var message = uint64(0xDEADBEEFDEADBEEF)
	a := assert.New(t)
	if dtor != nil {
		a.NoError(dtor(testMqName))
	}
	mq, err := ctor(testMqName, 0666)
	if !a.NoError(err) {
		return
	}
	defer func() {
		if dtor != nil {
			a.NoError(dtor(testMqName))
		} else {
			a.NoError(mq.Close())
		}
	}()
	data, _ := allocator.ObjectData(&message)
	if !a.NoError(mq.Send(data)) {
		return
	}
	var received uint64
	mqr, err := opener(testMqName, O_READ_ONLY)
	if !a.NoError(err) {
		return
	}
	data, _ = allocator.ObjectData(&received)
	err = mqr.Receive(data)
	a.NoError(err)
	a.Equal(message, received)
	allocator.UseValue(data)
}

func testMqSendStructSameProcess(t *testing.T, ctor mqCtor, opener mqOpener, dtor mqDtor) {
	type testStruct struct {
		arr [16]int
		c   complex128
		s   struct {
			a, b byte
		}
		f float64
	}
	a := assert.New(t)
	if dtor != nil {
		a.NoError(dtor(testMqName))
	}
	message := testStruct{c: complex(2, -3), f: 11.22, s: struct{ a, b byte }{127, 255}}
	mq, err := ctor(testMqName, 0666)
	if !a.NoError(err) {
		return
	}
	go func() {
		data, _ := allocator.ObjectData(message)
		a.NoError(mq.Send(data))
	}()
	received := testStruct{}
	mqr, err := opener(testMqName, O_READ_ONLY)
	if !a.NoError(err) {
		return
	}
	defer func() {
		a.NoError(mqr.Close())
		a.NoError(dtor(testMqName))
	}()
	data, _ := allocator.ObjectData(&received)
	a.NoError(mqr.Receive(data))
	a.Equal(message, received)
	a.NoError(mq.Close())
	allocator.UseValue(data)
}

func testMqSendMessageLessThenBuffer(t *testing.T, ctor mqCtor, opener mqOpener, dtor mqDtor) {
	a := assert.New(t)
	if dtor != nil {
		a.NoError(dtor(testMqName))
	}
	mq, err := ctor(testMqName, 0666)
	if !a.NoError(err) {
		return
	}
	message := make([]byte, 512)
	for i := range message {
		message[i] = byte(i)
	}
	go func() {
		a.NoError(mq.Send(message))
	}()
	received := make([]byte, 1024)
	mqr, err := opener(testMqName, O_READ_ONLY)
	if !a.NoError(err) {
		return
	}
	defer func() {
		a.NoError(mqr.Close())
		a.NoError(dtor(testMqName))
	}()
	a.NoError(mqr.Receive(received))
	a.Equal(message, received[:512])
	a.Equal(received[512:], make([]byte, 512))
	a.NoError(mq.Close())
}

func testMqSendNonBlock(t *testing.T, ctor mqCtor, dtor mqDtor) {
	a := assert.New(t)
	if dtor != nil {
		a.NoError(dtor(testMqName))
	}
	mq, err := ctor(testMqName, 0666)
	if !assert.NoError(t, err) {
		return
	}
	defer func() {
		a.NoError(dtor(testMqName))
	}()
	if blocker, ok := mq.(Blocker); ok {
		a.NoError(blocker.SetBlocking(false))
		endChan := make(chan bool, 1)
		go func() {
			data := make([]byte, 8)
			for i := 0; i < 100; i++ {
				a.NoError(mq.Send(data))
			}
			endChan <- true
		}()
		select {
		case <-endChan:
		case <-time.After(time.Millisecond * 300):
			t.Errorf("send on non-blocking mq blocked")
		}
	} else {
		t.Skipf("current mq impl on %s does not implement Blocker", runtime.GOOS)
	}
}

func testMqSendTimeout(t *testing.T, ctor mqCtor, dtor mqDtor) {
	a := assert.New(t)
	if dtor != nil {
		a.NoError(dtor(testMqName))
	}
	mq, err := ctor(testMqName, 0666)
	if !assert.NoError(t, err) {
		return
	}
	defer func() {
		a.NoError(dtor(testMqName))
	}()
	if tmq, ok := mq.(TimedMessenger); ok {
		data := make([]byte, 8)
		tm := time.Millisecond * 200
		if buf, ok := mq.(Buffered); ok {
			cap, err := buf.Cap()
			if !a.NoError(err) {
				return
			}
			for i := 0; i < cap; i++ {
				if !a.NoError(mq.Send(data)) {
					return
				}
			}
		}
		now := time.Now()
		err := tmq.SendTimeout(data, tm)
		a.Error(err)
		if sysErr, ok := err.(syscall.Errno); ok {
			a.True(sysErr.Temporary())
		}
		a.Condition(func() bool {
			return time.Now().Sub(now) >= tm
		})
	} else {
		t.Skipf("current mq impl on %s does not implement TimedMessenger", runtime.GOOS)
	}
}

func testMqReceiveTimeout(t *testing.T, ctor mqCtor, dtor mqDtor) {
	a := assert.New(t)
	if dtor != nil {
		a.NoError(dtor(testMqName))
	}
	mq, err := ctor(testMqName, 0666)
	if !assert.NoError(t, err) {
		return
	}
	defer func() {
		a.NoError(dtor(testMqName))
	}()
	if tmq, ok := mq.(TimedMessenger); ok {
		received := make([]byte, 8)
		tm := time.Millisecond * 200
		now := time.Now()
		err := tmq.ReceiveTimeout(received, tm)
		a.Error(err)
		if sysErr, ok := err.(syscall.Errno); ok {
			a.True(sysErr.Temporary())
		}
		a.Condition(func() bool {
			return time.Now().Sub(now) >= tm
		})
	} else {
		t.Skipf("current mq impl on %s does not implement TimedMessenger", runtime.GOOS)
	}
}

func testMqReceiveNonBlock(t *testing.T, ctor mqCtor, dtor mqDtor) {
	a := assert.New(t)
	if dtor != nil {
		a.NoError(dtor(testMqName))
	}
	mq, err := ctor(testMqName, 0666)
	if !assert.NoError(t, err) {
		return
	}
	defer func() {
		a.NoError(dtor(testMqName))
	}()
	if blocker, ok := mq.(Blocker); ok {
		a.NoError(blocker.SetBlocking(false))
		endChan := make(chan bool, 1)
		go func() {
			data := make([]byte, 8)
			for i := 0; i < 32; i++ {
				a.Error(mq.Receive(data))
			}
			endChan <- true
		}()
		select {
		case <-endChan:
		case <-time.After(time.Millisecond * 300):
			t.Errorf("receive on non-blocking mq blocked")
		}
	} else {
		t.Skipf("current mq impl on %s does not implement Blocker", runtime.GOOS)
	}
}

func testMqSendToAnotherProcess(t *testing.T, ctor mqCtor, dtor mqDtor, typ string) {
	a := assert.New(t)
	if dtor != nil {
		a.NoError(dtor(testMqName))
	}
	mq, err := ctor(testMqName, 0666)
	if !assert.NoError(t, err) {
		return
	}
	defer func() {
		a.NoError(dtor(testMqName))
	}()
	data := make([]byte, 2048)
	for i := range data {
		data[i] = byte(i)
	}
	args := argsForMqTestCommand(testMqName, -1, typ, "", data)
	go func() {
		a.NoError(mq.Send(data))
	}()
	result := ipc_testing.RunTestApp(args, nil)
	if !a.NoError(result.Err) {
		t.Logf("program output is: %s", result.Output)
	}
}

func testMqReceiveFromAnotherProcess(t *testing.T, ctor mqCtor, dtor mqDtor, typ string) {
	a := assert.New(t)
	if dtor != nil {
		a.NoError(dtor(testMqName))
	}
	mq, err := ctor(testMqName, 0666)
	if !a.NoError(err) {
		return
	}
	defer func() {
		a.NoError(dtor(testMqName))
	}()
	data := make([]byte, 2048)
	for i := range data {
		data[i] = byte(i)
	}
	args := argsForMqSendCommand(testMqName, -1, typ, "", data)
	result := ipc_testing.RunTestApp(args, nil)
	if !a.NoError(result.Err) {
		t.Logf("program output is %s", result.Output)
	}
	received := make([]byte, 2048)
	err = mq.Receive(received)
	a.NoError(err)
	a.Equal(data, received)
}

func TestMqXXX(t *testing.T) {
	for i := 0; i < 15; i++ {
		a := make([]byte, 1024*1024)
		_ = a
		runtime.GC()
		time.Sleep(time.Millisecond * 50)
	}
}