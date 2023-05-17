/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package proof

import (
	"crypto"
	"fmt"
	"io"
	"math/big"
	"os"
)

type HashSelf interface {
	New() *HashSelf
	LoadField(d []byte) error
	CHash() ([]byte, crypto.Hash)
}

func (keyPair RSAKeyPair) GenProof(QSlice []QElement, h HashSelf, Tag Tag, Matrix [][]byte) <-chan GenProofResponse {
	responseCh := make(chan GenProofResponse, 1)
	var res GenProofResponse

	//err := h.LoadField([]byte(Tag.T.Name))
	//if err != nil {
	//	res.StatueMsg.StatusCode = cess_pdp.ErrorInternal
	//	res.StatueMsg.Msg = err.Error()
	//	responseCh <- res
	//	return responseCh
	//}
	//err = h.LoadField([]byte(Tag.T.U))
	//if err != nil {
	//	res.StatueMsg.StatusCode = cess_pdp.ErrorInternal
	//	res.StatueMsg.Msg = err.Error()
	//	responseCh <- res
	//	return responseCh
	//}
	//err = h.LoadField([]byte(Tag.PhiHash))
	//if err != nil {
	//	res.StatueMsg.StatusCode = cess_pdp.ErrorInternal
	//	res.StatueMsg.Msg = err.Error()
	//	responseCh <- res
	//	return responseCh
	//}
	//
	//attest,err:=hex.DecodeString(Tag.Attest)
	//if err!=nil{
	//	res.StatueMsg.StatusCode = cess_pdp.ErrorInternal
	//	res.StatueMsg.Msg = err.Error()
	//	responseCh <- res
	//	return responseCh
	//}
	//tag_hash,hash_type := h.CHash()
	//err = rsa.VerifyPKCS1v15(keyPair.Spk, hash_type, tag_hash[:], attest)
	//if err != nil {
	//	panic(err)
	//}

	//Compute Mu
	mu := new(big.Int)
	sigma := new(big.Int).SetInt64(1)

	for i := 0; i < len(QSlice); i++ {
		//µ =Σ νi*mi ∈ Zp (i ∈ [1, n])
		mi := new(big.Int).SetBytes(Matrix[QSlice[i].I])
		vi, _ := new(big.Int).SetString(QSlice[i].V, 10)
		mu.Add(new(big.Int).Mul(mi, vi), mu)
		//σ =∏ σ^vi ∈ G (i ∈ [1, n])
		sigma_i, _ := new(big.Int).SetString(Tag.T.Phi[QSlice[i].I], 10)
		sigma.Mul(new(big.Int).Exp(sigma_i, vi, keyPair.Spk.N), sigma)
	}
	sigma.Mod(sigma, keyPair.Spk.N)

	res.MU = mu.String()
	res.Sigma = sigma.String()
	res.StatueMsg.StatusCode = Success
	res.StatueMsg.Msg = "Success"
	responseCh <- res

	return responseCh
}

func (keyPair RSAKeyPair) AggrGenProof(QSlice []QElement, Tag []Tag) string {
	sigma := new(big.Int).SetInt64(1)

	for _, tag := range Tag {
		for _, q := range QSlice {
			vi, _ := new(big.Int).SetString(q.V, 10)

			//σ =∏ σi^vi ∈ G (i ∈ [1, n])
			sigma_i, _ := new(big.Int).SetString(tag.T.Phi[q.I], 10)

			sigma_i.Exp(sigma_i, vi, keyPair.Spk.N)
			sigma.Mul(sigma, sigma_i)
		}
		sigma.Mod(sigma, keyPair.Spk.N)
	}
	return sigma.String()
}

func SplitByN(filePath string, N int64) (Data [][]byte, sep int64, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, err
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, 0, err
	}

	sep = int64(len(data)) / N
	if int64(len(data))%N != 0 {
		return nil, 0, fmt.Errorf("The file:%v ,size is %v can't divide by %v", filePath, len(data), N)
	}

	for i := int64(0); i < N; i++ {
		if i != N-1 {
			Data = append(Data, data[i*sep:(i+1)*sep])
			continue
		}
		Data = append(Data, data[i*sep:])
	}
	return
}
