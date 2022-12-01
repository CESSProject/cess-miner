package pbc

import (
	"crypto/sha256"
	"encoding/json"

	"github.com/Nik-U/pbc"
	"github.com/cbergoon/merkletree"
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

// SigGen need these parameters in keyPair
// Ssk: Private key
// SharedParams: Safety parameter
func (keyPair PBCKeyPair) SigGen(matrix [][]byte, n int64) <-chan SigGenResponse {
	responseCh := make(chan SigGenResponse, 1)
	var res SigGenResponse
	pairing, _ := pbc.NewPairingFromString(keyPair.SharedParams)
	privateKey := pairing.NewZr().SetBytes(keyPair.Ssk)

	var tag Tag
	tag.Name = pairing.NewZr().Rand().Bytes()
	tag.N = n
	tag.U = pairing.NewG1().Rand().Bytes()

	t := T{}
	t.Tag = tag

	tmp, err := json.Marshal(tag)
	if err != nil {
		res.StatueMsg.StatusCode = ErrorInternal
		res.StatueMsg.Msg = err.Error()
		responseCh <- res
		return responseCh
	}
	//var tau_zero_bytes bytes.Buffer
	//enc := gob.NewEncoder(&tau_zero_bytes)
	//err = enc.Encode(Tag)
	//if err != nil {
	//	res.StatueMsg.StatusCode = paramv1.Error
	//	res.StatueMsg.Msg = "encode tau_zero_bytes error" + err.Error()
	//	responseCh <- res
	//	return responseCh
	//}

	tagG1 := pairing.NewG1().SetFromStringHash(string(tmp), sha256.New())
	t.SigAbove = pairing.NewG1().PowZn(tagG1, privateKey).Bytes()
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

	treeRootG1 := pairing.NewG1().SetFromHash(tree.MerkleRoot())
	res.SigRootHash = pairing.NewG1().PowZn(treeRootG1, privateKey).Bytes()
	res.Phi = make([][]byte, n)

	g1wait := make(chan struct{}, n)
	for i := int64(0); i < n; i++ {
		go func(i int64) {
			res.Phi[i] = GenerateSigma(matrix[i], t.U, privateKey, pairing)
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

func GenerateSigma(data, u []byte, alpha *pbc.Element, pairing *pbc.Pairing) Sigma {
	productory := pairing.NewG1()
	data_hash_zr := pairing.NewG1().SetFromStringHash(string(data), sha256.New())
	data_byte_zr := pairing.NewZr().SetBytes(data)
	u_g1 := pairing.NewG1().SetBytes(u)

	//(H(mi) · u^mi )^α
	productory.PowZn(pairing.NewG1().Mul(data_hash_zr, pairing.NewG1().PowZn(u_g1, data_byte_zr)), alpha)

	return productory.Bytes()
}
