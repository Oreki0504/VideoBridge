//go:build !windows && !darwin

package bridge

import "os"

func publishFile(source, target string) error { return os.Link(source, target) }
