// Copyright 2015 Aleksandr Demakin. All rights reserved.
// ignore this for a while, as linux rw mutexes don't work,
// and windows mutexes are not ready yes.

package sync

import (
	"strconv"

	testutil "bitbucket.org/avd/go-ipc/internal/test"
	"bitbucket.org/avd/go-ipc/mmf"
	"bitbucket.org/avd/go-ipc/shm"
)

const (
	testProgPath = "./internal/test/sync/"
	testMemObj   = "go-ipc.sync-test.region"
)

var testProgFiles []string

func init() {
	var err error
	testProgFiles, err = testutil.LocatePackageFiles(testProgPath)
	if err != nil {
		panic(err)
	}
	if len(testProgFiles) == 0 {
		panic("no files to test mq")
	}
	for i, name := range testProgFiles {
		testProgFiles[i] = testProgPath + name
	}
}

func createMemoryRegionSimple(objMode, regionMode int, size int64, offset int64) (*mmf.MemoryRegion, error) {
	object, _, err := shm.NewMemoryObjectSize(testMemObj, objMode, 0666, size)
	if err != nil {
		return nil, err
	}
	defer func() {
		errClose := object.Close()
		if errClose != nil {
			panic(errClose.Error())
		}
	}()
	region, err := mmf.NewMemoryRegion(object, regionMode, offset, int(size))
	if err != nil {
		return nil, err
	}
	return region, nil
}

// Sync test program

func argsForSyncCreateCommand(name, t string) []string {
	return append(testProgFiles, "-object="+name, "-type="+t, "create")
}

func argsForSyncDestroyCommand(name string) []string {
	return append(testProgFiles, "-object="+name, "destroy")
}

func argsForSyncInc64Command(name, t string, jobs int, shmName string, n int, logFile string) []string {
	return append(testProgFiles,
		"-object="+name,
		"-type="+t,
		"-jobs="+strconv.Itoa(jobs),
		"-log="+logFile,
		"inc64",
		shmName,
		strconv.Itoa(n),
	)
}

func argsForSyncTestCommand(name, t string, jobs int, shmName string, n int, data []byte, log string) []string {
	return append(testProgFiles,
		"-object="+name,
		"-type="+t,
		"-jobs="+strconv.Itoa(jobs),
		"-log="+log,
		"test",
		shmName,
		strconv.Itoa(n),
		testutil.BytesToString(data),
	)
}
