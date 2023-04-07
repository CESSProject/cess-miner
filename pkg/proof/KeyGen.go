package proof

import (
	"crypto/rand"
	"crypto/rsa"
	"math/big"
)

var key *RSAKeyPair

func init() {
	key = &RSAKeyPair{
		Spk: new(rsa.PublicKey),
		Ssk: new(rsa.PrivateKey),
	}
}

func KeyGen() RSAKeyPair {
	ssk, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	return RSAKeyPair{
		Spk: &ssk.PublicKey,
		Ssk: ssk,
	}
}

func GetKey(e int, n *big.Int) *RSAKeyPair {
	key.Spk.E = e
	key.Spk.N = n
	return key
}
