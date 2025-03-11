//go:build plan9 || windows
// +build plan9 windows

package main

import "os"

func getStat(fi os.FileInfo) (ok bool, inode, device, uid, gid uint64) {
	return false, 0, 0, 0, 0
}
