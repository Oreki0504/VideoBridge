//go:build darwin

package bridge

import (
	"golang.org/x/sys/unix"
	"os"
)

func publishFile(source, target string) error {
	if err := unix.RenamexNp(source, target, unix.RENAME_EXCL); err != nil {
		return &os.LinkError{Op: "rename", Old: source, New: target, Err: err}
	}
	return nil
}
