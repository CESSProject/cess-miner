/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package proof

import (
	"crypto/rsa"
	"crypto/x509"
)

var key RSAKeyPair

func NewKey() *RSAKeyPair {
	return &RSAKeyPair{
		Spk: new(rsa.PublicKey),
	}
}

func (k *RSAKeyPair) SetPublickey(n []byte) error {
	pubkey, err := x509.ParsePKCS1PublicKey(n)
	if err != nil {
		return err
	}
	if k == nil {
		k = NewKey()
	}
	k.Spk = pubkey
	return nil
}
