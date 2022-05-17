package proof

import (
	"github.com/Nik-U/pbc"
)

func Keygen() PBCKeyPair {
	var keyPair PBCKeyPair
	params := pbc.GenerateA(160, 512)
	pairing := params.NewPairing()
	g := pairing.NewG2().Rand()

	privKey := pairing.NewZr().Rand()
	pubKey := pairing.NewG2().PowZn(g, privKey)
	keyPair.Spk = pubKey.Bytes()
	keyPair.Ssk = privKey.Bytes()
	keyPair.SharedParams = params.String()
	keyPair.SharedG = g.Bytes()

	keyPair.Alpha = privKey
	keyPair.G = g
	keyPair.V = pubKey

	return keyPair
}
