package proof

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/Nik-U/pbc"
)

func (prove PoDR2Prove) PoDR2ProofProve(spk []byte, sharedParams string, sharedG []byte, segmentSize int64) <-chan PoDR2ProveResponse {
	responseCh := make(chan PoDR2ProveResponse, 1)
	var res PoDR2ProveResponse
	// Verifica che tau sia corretto
	//var tau_zero_bytes bytes.Buffer
	pairing, _ := pbc.NewPairingFromString(sharedParams)
	g := pairing.NewG2().SetBytes(sharedG)
	publicKey := pairing.NewG2().SetBytes(spk)

	tmp, err := json.Marshal(prove.T.T0)
	//enc := gob.NewEncoder(&tau_zero_bytes)
	//err := enc.Encode(prove.T.T0)
	if err != nil {
		res.StatueMsg.StatusCode = ErrorParam
		res.StatueMsg.Msg = "T param error: " + err.Error()
		responseCh <- res
		return responseCh
	}
	hashed_t_0 := pairing.NewG2().SetFromStringHash(string(tmp), sha256.New())
	temp1 := pairing.NewGT().Pair(hashed_t_0, publicKey)
	temp2 := pairing.NewGT().Pair(pairing.NewG2().SetBytes(prove.T.Signature), g)
	if !temp1.Equals(temp2) {
		res.StatueMsg.StatusCode = ErrorParam
		res.StatueMsg.Msg = "Signature information verification error"
		fmt.Println("Signature information verification error")
		responseCh <- res
		return responseCh
	} else {
		fmt.Println("Signature information verification success")
	}

	//mu := make([]*pbc.Element, prove.S)
	U_num := prove.S / segmentSize
	if prove.S%segmentSize != 0 {
		U_num++
	}
	mu := make([][]byte, U_num)
	//for j := int64(0); j < prove.S; j++ {
	//	mu_j := pairing.NewZr()
	//	for _, qelem := range prove.QSlice {
	//		char := pairing.NewZr().SetFromHash([]byte{prove.Matrix[qelem.I-1][j]})
	//		product := pairing.NewZr().Mul(pairing.NewZr().SetBytes(qelem.V.Bytes()), char)
	//		mu_j.Add(mu_j, product)
	//	}
	//	mu[j] = mu_j
	//}
	for j := int64(0); j < U_num; j++ {
		mu_j := pairing.NewZr()
		for _, qelem := range prove.QSlice {
			if j == U_num-1 {
				char := pairing.NewZr().SetFromHash(prove.Matrix[qelem.I-1][j*segmentSize:])
				product := pairing.NewZr().Mul(pairing.NewZr().SetBytes(qelem.V), char)
				mu_j.Add(mu_j, product)
				continue
			}
			char := pairing.NewZr().SetFromHash(prove.Matrix[qelem.I-1][j*segmentSize : (j+1)*segmentSize])
			product := pairing.NewZr().Mul(pairing.NewZr().SetBytes(qelem.V), char)
			mu_j.Add(mu_j, product)
		}
		mu[j] = mu_j.Bytes()
	}

	sigma := pairing.NewG2()
	for _, qelem := range prove.QSlice {
		sigma.Mul(sigma, pairing.NewG2().PowZn(pairing.NewG2().SetBytes(prove.Sigmas[qelem.I-1]), pairing.NewZr().SetBytes(qelem.V)))
	}

	res.Sigma = sigma.Bytes()
	res.MU = mu
	res.StatueMsg.StatusCode = Success
	res.StatueMsg.Msg = "Success"
	responseCh <- res
	return responseCh
}
