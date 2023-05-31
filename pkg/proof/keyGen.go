/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package proof

import (
	"crypto/rsa"
	"math/big"
)

var key RSAKeyPair

func NewKey() *RSAKeyPair {
	return &RSAKeyPair{
		Spk: new(rsa.PublicKey),
	}
}

func (k *RSAKeyPair) SetKeyN(n []byte) {
	if k != nil && k.Spk != nil && len(n) > 0 {
		k.Spk.N = new(big.Int).SetBytes(n)
	}
}
