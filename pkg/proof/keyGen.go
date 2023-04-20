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
