//go:build windows

package compiler

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func replaceFileAtomically(src, dst string) error {
	srcPtr, err := syscall.UTF16PtrFromString(src)
	if err != nil {
		return fmt.Errorf("encode src path: %w", err)
	}
	dstPtr, err := syscall.UTF16PtrFromString(dst)
	if err != nil {
		return fmt.Errorf("encode dst path: %w", err)
	}

	const (
		moveFileReplaceExisting = 0x1
		moveFileWriteThrough    = 0x8
	)

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	moveFileExW := kernel32.NewProc("MoveFileExW")
	r1, _, callErr := moveFileExW.Call(
		uintptr(unsafe.Pointer(srcPtr)),
		uintptr(unsafe.Pointer(dstPtr)),
		uintptr(moveFileReplaceExisting|moveFileWriteThrough),
	)
	if r1 == 0 {
		if callErr != syscall.Errno(0) {
			return os.NewSyscallError("MoveFileExW", callErr)
		}
		return os.NewSyscallError("MoveFileExW", syscall.EINVAL)
	}

	return nil
}
