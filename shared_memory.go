// Copyright 2015 Aleksandr Demakin. All rights reserved.

package ipc

import (
	"bytes"
	"errors"
	"io"
	"os"
	"runtime"
	"unsafe"
)

// MemoryObject represents an object which can be used to
// map shared memory regions into the process' address space
type MemoryObject struct {
	*memoryObjectImpl
}

// MemoryRegion is a mmapped area of a memory object.
// Warning. The internal object has a finalizer set,
// so the region will be unmapped during the gc.
// Thus, you should be carefull getting internal data.
// For example, the following code may crash:
// func f() {
// 	region := NewMemoryRegion(...)
// 	return g(region.Data())
// region may be gc'ed while its data is used by g()
// To avoid this, you can use UseMemoryRegion() or region readers/writers
type MemoryRegion struct {
	*memoryRegionImpl
}

// MappableHandle is an object, which can return a handle,
// that can be used as a file descriptor for mmap
type MappableHandle interface {
	Fd() uintptr
}

// MemoryRegionReader is a reader for safe operations over a shared memory region.
// It holds a reference to the region, so the former can't be gc'ed
type MemoryRegionReader struct {
	region *MemoryRegion
	*bytes.Reader
}

func NewMemoryRegionReader(region *MemoryRegion) *MemoryRegionReader {
	return &MemoryRegionReader{
		region: region,
		Reader: bytes.NewReader(region.Data()),
	}
}

// MemoryRegionWriter is a writer for safe operations over a shared memory region.
// It holds a reference to the region, so the former can't be gc'ed
type MemoryRegionWriter struct {
	region *MemoryRegion
}

func NewMemoryRegionWriter(region *MemoryRegion) *MemoryRegionWriter {
	return &MemoryRegionWriter{region: region}
}

// WriteAt is to implement io.WriterAt
func (w *MemoryRegionWriter) WriteAt(p []byte, off int64) (n int, err error) {
	data := w.region.Data()
	n = len(data) - int(off)
	if n > 0 {
		if n > len(p) {
			n = len(p)
		}
		copy(data[off:], p[:n])
	}
	if n < len(p) {
		err = io.EOF
	}
	return
}

// NewMemoryObject creates a new shared memory object.
// name - a name of the object. should not contain '/' and exceed 255 symbols
// size - object size
// mode - open mode. see O_* constants
// perm - file's mode and permission bits.
func NewMemoryObject(name string, mode int, perm os.FileMode) (*MemoryObject, error) {
	impl, err := newMemoryObjectImpl(name, mode, perm)
	if err != nil {
		return nil, err
	}
	result := &MemoryObject{impl}
	runtime.SetFinalizer(impl, func(memObject *memoryObjectImpl) {
		memObject.Close()
	})
	return result, nil
}

// NewMemoryRegion creates a new shared memory region.
// object - an object containing a descriptor of the file, which can be mmaped
// size - object size
// mode - open mode. see SHM_* constants
// offset - offset in bytes from the beginning of the mmaped file
// size - region size
func NewMemoryRegion(object MappableHandle, mode int, offset int64, size int) (*MemoryRegion, error) {
	impl, err := newMemoryRegionImpl(object, mode, offset, size)
	if err != nil {
		return nil, err
	}
	result := &MemoryRegion{impl}
	runtime.SetFinalizer(impl, func(region *memoryRegionImpl) {
		region.Close()
	})
	return result, nil
}

// UseMemoryRegion ensures, that the object is still alive at the moment of the call.
// The usecase is when you use memory region's Data() and don't use the
// region itself anymore. In this case the region can be gc'ed, the memory mapping
// destroyed and you can get segfault.
// It can be used like the following:
// 	region := NewMemoryRegion(...)
//	UseMemoryRegion(region)
// 	data := region.Data()
//	{ work with data }
func UseMemoryRegion(region *MemoryRegion) {
	use(unsafe.Pointer(region))
}

// calcMmapOffsetFixup returns a value X,
// so that  offset - X is a multiplier of a system page size
func calcMmapOffsetFixup(offset int64) int64 {
	pageSize := int64(os.Getpagesize())
	return (offset - (offset/pageSize)*pageSize)
}

func checkMmapSize(fd uintptr, size int) (int, error) {
	if size == 0 {
		if fd == ^uintptr(0) {
			return 0, errors.New("must provide a valid file size")
		}
		file := os.NewFile(fd, "tempfile")
		fi, err := file.Stat()
		if err != nil {
			return 0, err
		}
		size = int(fi.Size())
	}
	return size, nil
}
