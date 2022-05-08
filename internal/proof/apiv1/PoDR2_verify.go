package proof

import (
	"github.com/Nik-U/pbc"
)

func (verify PoDR2Verify) PoDR2ProofVerify(SharedG, spk []byte, sharedParams string) bool {

	pairing, _ := pbc.NewPairingFromString(sharedParams)
	G := pairing.NewG2().SetBytes(SharedG)
	V := pairing.NewG2().SetBytes(spk)
	first := pairing.NewG2()
	for _, qelem := range verify.QSlice {
		hash := hashNameI(pairing.NewZr().SetBytes(verify.T.T0.Name), qelem.I, pairing)
		hash.PowZn(&hash, pairing.NewZr().SetBytes(qelem.V))
		first.Mul(first, &hash)
	}

	second := pairing.NewG2()
	s := int64(len(verify.T.T0.U))
	for j := int64(0); j < s; j++ {
		second.Mul(second, pairing.NewG2().PowZn(pairing.NewG2().SetBytes(verify.T.T0.U[j]), pairing.NewZr().SetBytes(verify.MU[j])))
	}
	left := pairing.NewGT()
	right := pairing.NewGT()

	left.Pair(pairing.NewG2().SetBytes(verify.Sigma), pairing.NewG2().SetBytes(G.Bytes()))
	right.Pair(pairing.NewG2().Mul(first, second), pairing.NewG2().SetBytes(V.Bytes()))

	return left.Equals(right)
}
