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

type StatueMsg struct {
	StatusCode int    `json:"status"`
	Msg        string `json:"msg"`
}

type QElement struct {
	I int64  `json:"i"`
	V string `json:"v"`
}

//————————————————————————————————————————————————————————————————Implement GenProof()————————————————————————————————————————————————————————————————

type Tag struct {
	T       T      `json:"t"`
	PhiHash string `json:"phi_hash"`
	Attest  string `json:"attest"`
}

type T struct {
	Name string   `json:"name"`
	U    string   `json:"u"`
	Phi  []string `json:"phi"`
}

type GenProofResponse struct {
	Sigma     string    `json:"sigma"`
	MU        string    `json:"mu"`
	StatueMsg StatueMsg `json:"statue_msg"`
}
