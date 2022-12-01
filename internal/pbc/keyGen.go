package pbc

import (
	"github.com/Nik-U/pbc"
)

func KeyGen() PBCKeyPair {
	var keyPair PBCKeyPair

	params := pbc.GenerateA(160, 512)

	pairing := params.NewPairing()
	g := pairing.NewG1().Rand()

	privKey := pairing.NewZr().Rand()
	pubKey := pairing.NewG1().PowZn(g, privKey)
	keyPair.Spk = pubKey.Bytes()
	keyPair.Ssk = privKey.Bytes()
	keyPair.SharedParams = params.String()
	keyPair.SharedG = g.Bytes()
	keyPair.ZrLength = pairing.ZrLength()

	return keyPair
}
