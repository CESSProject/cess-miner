package pbc

import (
	"github.com/Nik-U/pbc"
)

func (keyPair PBCKeyPair) VerifyProof(t T, qSlice []QElement, m, sigma Sigma, mht MHTInfo) bool {
	pairing, _ := pbc.NewPairingFromString(keyPair.SharedParams)
	g := pairing.NewG1().SetBytes(keyPair.SharedG)
	v := pairing.NewG1().SetBytes(keyPair.Spk)

	multiply := pairing.NewG1()
	for i := 0; i < len(qSlice); i++ {
		hashMi := pairing.NewG1().SetBytes(mht.HashMi[i])
		// ∏ H(mi)^νi (i ∈ [1, n])
		multiply.Mul(multiply, pairing.NewG1().PowZn(hashMi, pairing.NewZr().SetBytes(qSlice[i].V)))
	}

	//u^µ
	u := pairing.NewG1().SetBytes(t.U)
	mu := pairing.NewZr().SetBytes(m)
	uPowMu := pairing.NewG1().PowZn(u, mu)

	left := pairing.NewGT()
	right := pairing.NewGT()
	left.Pair(pairing.NewG1().SetBytes(sigma), g)
	right.Pair(pairing.NewG1().Mul(multiply, uPowMu), v)

	return left.Equals(right)
}
