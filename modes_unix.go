//go:build android || darwin || linux || nacl || netbsd
// +build android darwin linux nacl netbsd

package main

import "syscall"

const modeExecute = syscall.S_IXUSR | syscall.S_IXGRP | syscall.S_IXOTH
