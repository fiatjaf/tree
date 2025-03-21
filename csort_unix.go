//go:build linux || openbsd || dragonfly || android || solaris
// +build linux openbsd dragonfly android solaris

package main

import (
	"os"
	"syscall"
)

func CTimeSort(f1, f2 os.FileInfo) bool {
	if f1 == nil || f2 == nil {
		return f2 == nil
	}
	s1, ok1 := f1.Sys().(*syscall.Stat_t)
	s2, ok2 := f2.Sys().(*syscall.Stat_t)
	// If this type of node isn't an os node then revert to ModSort
	if !ok1 || !ok2 {
		return ModSort(f1, f2)
	}
	return s1.Ctim.Sec < s2.Ctim.Sec
}
