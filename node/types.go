/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

const MaxReplaceFiles = 5

const (
	Active = iota
	Calculate
	Missing
	Recovery
)

const (
	Cach_prefix_metadata      = "metadata:"
	Cach_prefix_report        = "report:"
	Cach_prefix_idle          = "idle:"
	Cach_prefix_idleSiama     = "sigmaidle:"
	Cach_prefix_serviceSiama  = "sigmaservice:"
	Cach_AggrProof_Reported   = "AggrProof_Reported"
	Cach_AggrProof_Transfered = "AggrProof_Transfered"
)

const P2PResponseOK uint32 = 200

type ProofFileType struct {
	Names []string `json:"names"`
	Us    []string `json:"us"`
	Mus   []string `json:"mus"`
	Sigma string   `json:"sigma"`
}

type RandomList struct {
	Index  []uint32 `json:"index"`
	Random [][]byte `json:"random"`
}
