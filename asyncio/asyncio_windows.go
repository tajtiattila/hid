package asyncio

import (
	"errors"
	"os"
	"sync"
	"syscall"
	"time"
)

var ErrTimeout = errors.New("asyncio timeout")

// File represents an open file descriptor that supports timeouts.
//
// Read and Write operations uses the timeout.
//
// Many functions, such as Fd, Stat, Close are supported
// directly through os.File.
type File struct {
	*os.File

	handle syscall.Handle

	timeout uint32 // milliseconds

	rl sync.Mutex
	ro *syscall.Overlapped

	wl sync.Mutex
	wo *syscall.Overlapped
}

func Open(name string) (*File, error) {
	n, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(n,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL|syscall.FILE_FLAG_OVERLAPPED,
		0)
	if err != nil {
		return nil, err
	}
	ro, err := newOverlapped()
	if err != nil {
		return nil, err
	}
	wo, err := newOverlapped()
	if err != nil {
		return nil, err
	}
	return &File{
		File:   os.NewFile(uintptr(h), name),
		handle: h,
		ro:     ro,
		wo:     wo}, nil
}

// SetTimeout sets the timeout for Read and Write operations.
func (f *File) SetTimeout(x time.Duration) {
	if x == 0 {
		f.timeout = 0
		return
	}
	t := uint32(x / time.Millisecond)
	if t <= 0 {
		t = 1
	}
	f.timeout = t
}

func (f *File) Read(p []byte) (n int, err error) {
	// https://support.microsoft.com/hu-hu/kb/156932
	f.rl.Lock()
	defer f.rl.Unlock()

	if err := resetEvent(f.ro.HEvent); err != nil {
		return 0, err
	}
	var nread uint32
	err = syscall.ReadFile(f.handle, p, &nread, f.ro)
	if err == nil {
		// completed synchronously
		return int(nread), nil
	}
	if err != nil && err != syscall.ERROR_IO_PENDING {
		return int(nread), err
	}
	// i/o pending
	return f.overlappedResult(f.ro)
}

func (f *File) Write(p []byte) (n int, err error) {
	f.wl.Lock()
	defer f.wl.Unlock()

	if err := resetEvent(f.wo.HEvent); err != nil {
		return 0, err
	}
	var nwritten uint32
	err = syscall.WriteFile(f.handle, p, &nwritten, f.wo)
	if err == nil {
		// completed synchronously
		return int(nwritten), nil
	}
	if err != nil && err != syscall.ERROR_IO_PENDING {
		return int(nwritten), err
	}
	// i/o pending
	return f.overlappedResult(f.wo)
}

func (f *File) overlappedResult(o *syscall.Overlapped) (n int, err error) {
	// https://blogs.msdn.microsoft.com/oldnewthing/20110202-00/?p=11613/
	var done uint32
	if f.timeout != 0 {
		evt, err := syscall.WaitForSingleObject(o.HEvent, f.timeout)
		if err != nil {
			return 0, err
		}
		if evt == syscall.WAIT_TIMEOUT {
			if err := syscall.CancelIo(f.handle); err != nil {
				// fatal?
				return 0, err
			}
			syscall.WaitForSingleObject(o.HEvent, syscall.INFINITE)
			return 0, ErrTimeout
		}
	}

	if err := getOverlappedResult(f.handle, o, &done, true); err != nil {
		return 0, err
	}
	return int(done), nil
}

func newOverlapped() (*syscall.Overlapped, error) {
	h, err := createEvent(nil, true, false, nil)
	if err != nil {
		return nil, err
	}
	return &syscall.Overlapped{HEvent: h}, nil
}

//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zasyncio_windows.go asyncio_windows.go

//sys createEvent(a *syscall.SecurityAttributes, m bool, i bool, n *uint16) (h syscall.Handle, err error) = CreateEventW
//sys resetEvent(h syscall.Handle) (err error) = ResetEvent
//sys getOverlappedResult(h syscall.Handle, o *syscall.Overlapped, n *uint32, wait bool) (err error) = GetOverlappedResult
