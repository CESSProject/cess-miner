/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package proof

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
)

func NewKey() *RSAKeyPair {
	return &RSAKeyPair{
		Spk: new(rsa.PublicKey),
	}
}

func (r *RSAKeyPair) VerifyAttest(name, u, phiHash, attest, customData string) (bool, error) {
	bytesHash, err := hex.DecodeString(phiHash)
	if err != nil {
		return false, err
	}
	bytesAttest, err := hex.DecodeString(attest)
	if err != nil {
		return false, err
	}
	hash := sha256.New()
	if customData != "" {
		hash.Write([]byte(customData))
	}
	hash.Write([]byte(name))
	hash.Write([]byte(u))
	hash.Write(bytesHash)
	hdata := hash.Sum(nil)
	hash.Reset()
	hash.Write(hdata)
	err = rsa.VerifyPKCS1v15(r.Spk, crypto.SHA256, hash.Sum(nil), bytesAttest)
	if err != nil {
		return false, err
	}
	return true, nil
}
