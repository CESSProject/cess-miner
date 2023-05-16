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
	Cach_prefix_metadata     = "metadata:"
	Cach_prefix_report       = "report:"
	Cach_prefix_idle         = "idle:"
	Cach_prefix_idleSiama    = "sigmaidle:"
	Cach_prefix_serviceSiama = "sigmaservice:"
)

const P2PResponseOK uint32 = 200

type ProofFileType struct {
	Name []string `json:"name"`
	U    []string `json:"u"`
}

type ProofMuFileType struct {
	Mu []string `json:"mu"`
}
