/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package proof

import (
	"crypto/rsa"
)

func NewKey() *RSAKeyPair {
	return &RSAKeyPair{
		Spk: new(rsa.PublicKey),
	}
}
