/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package configs

import (
	"os"
	"runtime"
)

const (
	// Name is the name of the program
	Name = "bucket"
	// version
	Version = "v0.6.2"
	// Description is the description of the program
	Description = "Storage node implementation in CESS networks"
	// NameSpace is the cached namespace
	NameSpaces = Name
)

// system init
func SysInit() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	if !RunOnLinuxSystem() {
		Err("Please run on a linux system")
		os.Exit(1)
	}
}

func RunOnLinuxSystem() bool {
	return runtime.GOOS == "linux"
}
