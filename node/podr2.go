/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"os"
)

const (
	Success            = 200
	Error              = 201
	ErrorParam         = 202
	ErrorParamNotFound = 203
	ErrorInternal      = 204
)

type RSAKeyPair struct {
	Spk *rsa.PublicKey
}

type StatueMsg struct {
	StatusCode int    `json:"status"`
	Msg        string `json:"msg"`
}

type QElement struct {
	I int64  `json:"i"`
	V string `json:"v"`
}

type Tag struct {
	T       T      `json:"t"`
	PhiHash string `json:"phi_hash"`
	Attest  string `json:"attest"`
}

type T struct {
	Name string   `json:"name"`
	U    string   `json:"u"`
	Phi  []string `json:"phi"`
}

type GenProofResponse struct {
	Sigma     string    `json:"sigma"`
	MU        string    `json:"mu"`
	StatueMsg StatueMsg `json:"statue_msg"`
}

type HashSelf interface {
	New() *HashSelf
	LoadField(d []byte) error
	CHash() ([]byte, crypto.Hash)
}

func NewRsaKey(pubkey []byte) (*RSAKeyPair, error) {
	rsaPubkey, err := x509.ParsePKCS1PublicKey(pubkey)
	if err != nil {
		return nil, err
	}
	raskey := &RSAKeyPair{
		Spk: rsaPubkey,
	}
	return raskey, nil
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

func (keyPair RSAKeyPair) GenProof(QSlice []QElement, h HashSelf, Phi []string, Matrix [][]byte) GenProofResponse {
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
		sigma_i, _ := new(big.Int).SetString(Phi[QSlice[i].I], 10)
		sigma.Mul(new(big.Int).Exp(sigma_i, vi, keyPair.Spk.N), sigma)
	}
	sigma.Mod(sigma, keyPair.Spk.N)

	res.MU = mu.String()
	res.Sigma = sigma.String()
	res.StatueMsg.StatusCode = Success
	res.StatueMsg.Msg = "Success"

	return res
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

func (keyPair RSAKeyPair) AggrAppendProof(AggrSigma string, aSigma string) (string, bool) {
	if AggrSigma == "" {
		AggrSigma = "1"
	}

	sigma, ok := new(big.Int).SetString(AggrSigma, 10)
	if !ok {
		return "", false
	}
	subSigma, ok := new(big.Int).SetString(aSigma, 10)
	if !ok {
		return "", false
	}
	sigma.Mul(sigma, subSigma)
	sigma.Mod(sigma, keyPair.Spk.N)

	return sigma.String(), true
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
