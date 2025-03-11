//go:build !linux && !openbsd && !dragonfly && !android && !solaris && !darwin && !freebsd && !netbsd
// +build !linux,!openbsd,!dragonfly,!android,!solaris,!darwin,!freebsd,!netbsd

package main

// CtimeSort for unsupported OS - just compare ModTime
var CTimeSort = ModSort
