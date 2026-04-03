//go:build !windows

package compiler

import "os"

func replaceFileAtomically(src, dst string) error {
	return os.Rename(src, dst)
}
