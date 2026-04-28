//go:build linux

package exec

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

const atEmptyPath = 0x1000

func archSyscalls() (sysMemfdCreate, sysExecveat uintptr) {
	switch runtime.GOARCH {
	case "amd64":
		return 319, 322
	case "arm64":
		return 279, 281
	case "arm":
		return 385, 387
	case "mips", "mipsle":
		return 4354, 4360
	default:
		panic("gomexec: unsupported arch: " + runtime.GOARCH)
	}
}

func newMemfd() (*os.File, error) {
	sysMemfdCreate, _ := archSyscalls()
	namePtr, err := syscall.BytePtrFromString("")
	if err != nil {
		return nil, err
	}
	// flags=0: no MFD_CLOEXEC, fd must outlive the write and reach execveat
	fd, _, errno := syscall.Syscall(sysMemfdCreate, uintptr(unsafe.Pointer(namePtr)), 0, 0)
	runtime.KeepAlive(namePtr)
	if errno != 0 {
		return nil, errno
	}
	return os.NewFile(fd, ""), nil
}

func doExecveat(fd int, argv, envv []string) error {
	_, sysExecveat := archSyscalls()

	emptyPath, err := syscall.BytePtrFromString("")
	if err != nil {
		return err
	}
	argvPtrs, err := strSliceToPtrs(argv)
	if err != nil {
		return fmt.Errorf("argv: %w", err)
	}
	envvPtrs, err := strSliceToPtrs(envv)
	if err != nil {
		return fmt.Errorf("envv: %w", err)
	}

	_, _, errno := syscall.Syscall6(
		sysExecveat,
		uintptr(fd),
		uintptr(unsafe.Pointer(emptyPath)),
		uintptr(unsafe.Pointer(&argvPtrs[0])),
		uintptr(unsafe.Pointer(&envvPtrs[0])),
		atEmptyPath,
		0,
	)
	runtime.KeepAlive(emptyPath)
	runtime.KeepAlive(argvPtrs)
	runtime.KeepAlive(envvPtrs)
	if errno != 0 {
		return errno
	}
	return nil
}

func strSliceToPtrs(ss []string) ([]*byte, error) {
	ptrs := make([]*byte, len(ss)+1) // null-terminated
	for i, s := range ss {
		p, err := syscall.BytePtrFromString(s)
		if err != nil {
			return nil, err
		}
		ptrs[i] = p
	}
	return ptrs, nil
}

// RunFromReader creates a memfd, streams r into it, then execveat's it.
// On execveat failure falls back to /proc/self/fd path exec.
func RunFromReader(r io.Reader, argv, envv []string) error {
	f, err := newMemfd()
	if err != nil {
		return fmt.Errorf("memfd_create: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("write memfd: %w", err)
	}

	fd := int(f.Fd())

	if err := doExecveat(fd, argv, envv); err != nil {
		procPath := fmt.Sprintf("/proc/self/fd/%d", fd)
		if execErr := syscall.Exec(procPath, argv, envv); execErr != nil {
			return fmt.Errorf("execveat: %w, /proc fallback: %v", err, execErr)
		}
	}
	return nil
}

// Run is a convenience wrapper over RunFromReader for in-memory payloads.
func Run(payload []byte, argv, envv []string) error {
	return RunFromReader(bytes.NewReader(payload), argv, envv)
}
