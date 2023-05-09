/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package proof

import (
	"crypto/rsa"
)

const (
	Success            = 200
	Error              = 201
	ErrorParam         = 202
	ErrorParamNotFound = 203
	ErrorInternal      = 204
)

type RSAKeyPair struct {
	Spk *rsa.PublicKey
	Ssk *rsa.PrivateKey
}

//————————————————————————————————————————————————————————————————Implement SigGen()————————————————————————————————————————————————————————————————

// Sigma is σ
type Sigma = []byte

// T be the file tag for F
type T struct {
	Tag
	SigAbove []byte
}

// Tag belongs to T
type Tag struct {
	Name []byte `json:"name"`
	N    int64  `json:"n"`
	U    []byte `json:"u"`
}

// SigGenResponse is result of SigGen() step
type SigGenResponse struct {
	T           T         `json:"t"`
	Phi         []Sigma   `json:"phi"`           //Φ = {σi}
	SigRootHash []byte    `json:"sig_root_hash"` //BLS
	StatueMsg   StatueMsg `json:"statue_msg"`
}

type StatueMsg struct {
	StatusCode int    `json:"status"`
	Msg        string `json:"msg"`
}

//————————————————————————————————————————————————————————————————Implement ChalGen()————————————————————————————————————————————————————————————————

type QElement struct {
	I int64  `json:"i"`
	V []byte `json:"v"`
}

//————————————————————————————————————————————————————————————————Implement GenProof()————————————————————————————————————————————————————————————————

type GenProofResponse struct {
	Sigma Sigma  `json:"sigmas"`
	MU    []byte `json:"mu"`
	MHTInfo
	SigRootHash []byte    `json:"sig_root_hash"`
	StatueMsg   StatueMsg `json:"statue_msg"`
}

type MHTInfo struct {
	HashMi [][]byte `json:"hash_mi"`
	Omega  []byte   `json:"omega"`
}

//-----------------------------old type------------------------------------

type PoDR2Commit struct {
	FilePath  string `json:"file_path"`
	BlockSize int64  `json:"block_size"`
}

type PoDR2CommitResponse struct {
	T         FileTagT       `json:"file_tag_t"`
	Sigmas    [][]byte       `json:"sigmas"`
	StatueMsg PoDR2StatueMsg `json:"statue_msg"`
}
type PoDR2StatueMsg struct {
	StatusCode int    `json:"status"`
	Msg        string `json:"msg"`
}

type PoDR2Prove struct {
	QSlice []QElement `json:"q_slice"`
	T      FileTagT   `json:"file_tag_t"`
	Sigmas [][]byte   `json:"sigmas"`
	Matrix [][]byte   `json:"matrix"`
	S      int64      `json:"s"`
}

type PoDR2ProveResponse struct {
	Sigma     []byte         `json:"sigma"`
	MU        [][]byte       `json:"mu"`
	StatueMsg PoDR2StatueMsg `json:"statue_msg"`
}

type PoDR2Verify struct {
	T      FileTagT   `json:"file_tag_t"`
	QSlice []QElement `json:"q_slice"`
	MU     [][]byte   `json:"mu"`
	Sigma  []byte     `json:"sigma"`
}
type FileTagT struct {
	T0        `json:"t0"`
	Signature []byte `json:"signature"`
}

type T0 struct {
	Name []byte   `json:"name"`
	N    int64    `json:"n"`
	U    [][]byte `json:"u"`
}

type HashNameAndI struct {
	Name string
	I    int64
}

type StorageTagType struct {
	T           T
	Phi         []Sigma `json:"phi"`
	SigRootHash []byte  `json:"sig_root_hash"`
	E           string  `json:"e"`
	N           string  `json:"n"`
}
