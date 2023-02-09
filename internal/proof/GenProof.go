package proof

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"math/big"
	"os"

	"github.com/CESSProject/go-merkletree"
)

func (keyPair RSAKeyPair) GenProof(QSlice []QElement, t T, Phi []Sigma, Matrix [][]byte, SigRootHash []byte) <-chan GenProofResponse {
	responseCh := make(chan GenProofResponse, 1)
	var res GenProofResponse
	tmp, err := json.Marshal(t.Tag)
	if err != nil {
		res.StatueMsg.StatusCode = ErrorInternal
		res.StatueMsg.Msg = err.Error()
		responseCh <- res
		return responseCh
	}

	tag_hash := sha256.Sum256(tmp)
	err = rsa.VerifyPKCS1v15(keyPair.Spk, crypto.SHA256, tag_hash[:], t.SigAbove)
	if err != nil {
		res.StatueMsg.StatusCode = ErrorInternal
		res.StatueMsg.Msg = err.Error()
		responseCh <- res
		return responseCh
	}

	//Compute Mu
	mu := new(big.Int)
	sigma := new(big.Int).SetInt64(1)

	for i := 0; i < len(QSlice); i++ {
		//µ =Σ νi*mi ∈ Zp (i ∈ [1, n])
		mi := new(big.Int).SetBytes(Matrix[QSlice[i].I])
		vi := new(big.Int).SetBytes(QSlice[i].V)
		mu.Add(new(big.Int).Mul(mi, vi), mu)
		//σ =∏ σ^vi ∈ G (i ∈ [1, n])
		sigma_i := new(big.Int).SetBytes(Phi[QSlice[i].I])
		sigma.Mul(new(big.Int).Exp(sigma_i, vi, keyPair.Spk.N), sigma)
		hash_mi := sha256.New()
		hash_mi.Write(Matrix[QSlice[i].I])
		res.HashMi = append(res.HashMi, hash_mi.Sum([]byte{}))
	}

	//Generate MHT tree
	var list []merkletree.Content
	for _, v := range Matrix {
		list = append(list, Content{x: string(v)})
	}
	//Create a new Merkle Tree from the list of Content
	tree, err := merkletree.NewTree(list)
	if err != nil {
		res.StatueMsg.StatusCode = ErrorInternal
		res.StatueMsg.Msg = err.Error()
		responseCh <- res
		return responseCh
	}

	//Get auxiliary info omega
	var I []int64
	for _, i := range QSlice {
		I = append(I, i.I)
	}
	_, _, nodelist, err := tree.GetMerkleAuxiliaryNode(merkletree.GetContentMap(I))
	if err != nil {
		res.StatueMsg.StatusCode = ErrorInternal
		res.StatueMsg.Msg = err.Error()
		responseCh <- res
		return responseCh
	}
	var nodes []merkletree.NodeSerializable
	for _, v := range nodelist {
		var n merkletree.NodeSerializable
		n.Hash = v.Hash
		n.Index = v.Index
		n.Height = v.Height
		nodes = append(nodes, n)
	}
	res.MHTInfo.Omega, err = json.Marshal(nodes)
	if err != nil {
		res.StatueMsg.StatusCode = ErrorInternal
		res.StatueMsg.Msg = err.Error()
		responseCh <- res
		return responseCh
	}
	res.MU = mu.Bytes()
	res.Sigma = sigma.Bytes()
	res.SigRootHash = SigRootHash
	res.StatueMsg.StatusCode = Success
	res.StatueMsg.Msg = "Success"
	responseCh <- res

	return responseCh
}

func SplitV2(fpath string, sep int64) (Data [][]byte, N int64, err error) {
	data, err := os.ReadFile(fpath)
	if err != nil {
		return nil, 0, err
	}
	file_size := int64(len(data))
	if sep > file_size {
		Data = append(Data, data)
		N = 1
		return
	}

	N = file_size / sep
	if file_size%sep != 0 {
		N += 1
	}

	for i := int64(0); i < N; i++ {
		if i != N-1 {
			Data = append(Data, data[i*sep:(i+1)*sep])
			continue
		}
		Data = append(Data, data[i*sep:])
		if l := sep - int64(len(data[i*sep:])); l > 0 {
			Data[i] = append(Data[i], make([]byte, l, l)...)
		}
	}

	return
}
