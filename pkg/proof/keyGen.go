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

func GetKey(n []byte) *RSAKeyPair {
	if key.Spk == nil {
		key.Spk = new(rsa.PublicKey)
		key.Spk.N = new(big.Int).SetBytes(n)
	}
	return &key
}
