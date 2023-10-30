/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package configs

import (
	"os"
	"runtime"

	"github.com/CESSProject/p2p-go/out"
)

const (
	// Name is the name of the program
	Name = "bucket"
	// version
	Version = "v0.7.3"
	// Description is the description of the program
	Description = "Storage node implementation in CESS networks"
	// NameSpace is the cached namespace
	NameSpaces = Name
)

// system init
func SysInit() int {
	cpuCore := runtime.NumCPU() * 8 / 10
	if cpuCore == 0 {
		cpuCore = 1
	}
	runtime.GOMAXPROCS(cpuCore)
	if !RunOnLinuxSystem() {
		out.Err("Please run on a linux system")
		os.Exit(1)
	}
	return cpuCore
}

func RunOnLinuxSystem() bool {
	return runtime.GOOS == "linux"
}
