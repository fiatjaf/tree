//go:build dragonfly || freebsd || openbsd || solaris || windows
// +build dragonfly freebsd openbsd solaris windows

package main

import "syscall"

const modeExecute = syscall.S_IXUSR
