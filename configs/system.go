/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package configs

import "time"

const (
	// Name is the name of the program
	Name = "bucket"
	// version
	Version = "v0.6.0 sprint4 dev"
	// Description is the description of the program
	Description = "Mining service based on cess platform"
	// NameSpace is the cached namespace
	NameSpace = Name
)

const (
	// BlockInterval is the time interval for generating blocks, in seconds
	BlockInterval = time.Second * time.Duration(6)
	//
	DirMode = 0644
)
