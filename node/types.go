/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

const MaxReplaceFiles = 30

const (
	Active = iota
	Calculate
	Missing
	Recovery
)

const (
	Cach_prefix_metadata = "metadata:"
	Cach_prefix_report   = "report:"
	Cach_prefix_idle     = "idle:"
)

const P2PResponseOK uint32 = 200
