package proof

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"math/big"

	"github.com/CESSProject/go-merkletree"
)

// TestContent implements the Content interface provided by merkletree and represents the content stored in the tree.
type Content struct {
	x string
}

// CalculateHash hashes the values of a TestContent
func (t Content) CalculateHash() ([]byte, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(t.x)); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

// Equals tests for equality of two Contents
func (t Content) Equals(other merkletree.Content) (bool, error) {
	return t.x == other.(Content).x, nil
}

func (keyPair RSAKeyPair) SigGen(matrix [][]byte, n int64) <-chan SigGenResponse {
	responseCh := make(chan SigGenResponse, 1)
	var res SigGenResponse
	var tag Tag

	tag.Name = make([]byte, 512)
	tag.N = n
	result, err := rand.Int(rand.Reader, keyPair.Ssk.PublicKey.N)
	if err != nil {
		panic(err)
	}
	tag.U = result.Bytes()
	_, err = rand.Read(tag.Name)
	if err != nil {
		panic(err)
	}

	t := T{}
	t.Tag = tag

	tmp, err := json.Marshal(tag)
	if err != nil {
		res.StatueMsg.StatusCode = ErrorInternal
		res.StatueMsg.Msg = err.Error()
		responseCh <- res
		return responseCh
	}
	tag_hash := sha256.Sum256(tmp)
	t.SigAbove, err = rsa.SignPKCS1v15(nil, keyPair.Ssk, crypto.SHA256, tag_hash[:])
	res.T = t

	//Generate MHT root
	var list []merkletree.Content
	for _, v := range matrix {
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
	res.SigRootHash = tree.MerkleRoot()
	res.Phi = make([][]byte, n)

	g1wait := make(chan struct{}, n)
	for i := int64(0); i < n; i++ {
		go func(i int64) {
			res.Phi[i] = GenerateSigma(matrix[i], t.Tag.U, keyPair.Ssk)
			g1wait <- struct{}{}
		}(i)
	}
	for i := int64(0); i < n; i++ {
		<-g1wait
	}

	res.StatueMsg.StatusCode = Success
	res.StatueMsg.Msg = "PoDR2ProofCommit success"
	responseCh <- res
	return responseCh
}

func GenerateSigma(data, u []byte, ssk *rsa.PrivateKey) Sigma {
	productory := big.NewInt(1)
	data_hash := sha256.Sum256(data)
	data_bigint := new(big.Int).SetBytes(data[:])
	data_hash_bigint := new(big.Int).SetBytes(data_hash[:])
	data_hash_bigint.Mod(data_hash_bigint, ssk.N)
	u_bigint := new(big.Int).SetBytes(u)

	//(H(mi) · u^mi )^α
	umi := new(big.Int).Exp(u_bigint, data_bigint, ssk.N)
	summary := new(big.Int).Mul(data_hash_bigint, umi)
	summary.Mod(summary, ssk.N)
	productory.Exp(summary, ssk.D, ssk.PublicKey.N)
	return productory.Bytes()
}
