package pbc

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/Nik-U/pbc"
)

func (keyPair PBCKeyPair) GenProof(QSlice []QElement, t T, Phi []Sigma, Matrix [][]byte, SigRootHash []byte) <-chan GenProofResponse {
	responseCh := make(chan GenProofResponse, 1)
	var res GenProofResponse

	pairing, _ := pbc.NewPairingFromString(keyPair.SharedParams)
	g := pairing.NewG1().SetBytes(keyPair.SharedG)
	publicKey := pairing.NewG1().SetBytes(keyPair.Spk)

	tmp, err := json.Marshal(t.Tag)
	if err != nil {
		res.StatueMsg.StatusCode = ErrorInternal
		res.StatueMsg.Msg = err.Error()
		responseCh <- res
		return responseCh
	}

	tagG1 := pairing.NewG1().SetFromStringHash(string(tmp), sha256.New())
	temp1 := pairing.NewGT().Pair(tagG1, publicKey)
	temp2 := pairing.NewGT().Pair(pairing.NewG1().SetBytes(t.SigAbove), g)
	if !temp1.Equals(temp2) {
		res.StatueMsg.StatusCode = ErrorParam
		res.StatueMsg.Msg = "Signature information verification error"
		responseCh <- res
		return responseCh
	} else {
		fmt.Println("Signature information verification success")
	}

	//Compute Mu
	mu := pairing.NewZr()
	sigma := pairing.NewG1()

	for i := 0; i < len(QSlice); i++ {
		//µ =Σ νi*mi ∈ Zp (i ∈ [1, n])
		mi := pairing.NewZr().SetBytes(Matrix[QSlice[i].I])
		vi := pairing.NewZr().SetBytes(QSlice[i].V)
		mu.Add(pairing.NewZr().Mul(mi, vi), mu)

		//σ =∏ σ^vi ∈ G (i ∈ [1, n])
		sigma_i := pairing.NewG1().SetBytes(Phi[QSlice[i].I])
		sigma.Mul(pairing.NewG1().PowZn(sigma_i, vi), sigma)

		hash_mi := pairing.NewG1().SetFromStringHash(string(Matrix[QSlice[i].I]), sha256.New())
		res.HashMi = append(res.HashMi, hash_mi.Bytes())
	}

	res.MU = mu.Bytes()
	res.Sigma = sigma.Bytes()
	res.SigRootHash = SigRootHash
	res.StatueMsg.StatusCode = Success
	res.StatueMsg.Msg = "Success"
	responseCh <- res
	return responseCh
}

func Split(filefullpath string, blocksize, filesize int64) ([][]byte, uint64, error) {
	file, err := os.Open(filefullpath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	if filesize/blocksize == 0 {
		return nil, 0, errors.New("filesize invalid")
	}
	n := uint64(math.Ceil(float64(filesize / blocksize)))
	if n == 0 {
		n = 1
	}
	// matrix is indexed as m_ij, so the first dimension has n items and the second has s.
	matrix := make([][]byte, n)
	for i := uint64(0); i < n; i++ {
		piece := make([]byte, blocksize)
		_, err := file.Read(piece)
		if err != nil {
			return nil, 0, err
		}
		matrix[i] = piece
	}
	return matrix, n, nil
}

func SplitV2(filePath string, sep int64) (Data [][]byte, N int64) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}
	if sep > int64(len(data)) {
		Data = append(Data, data)
		N = 1
		return
	}

	N = int64(len(data)) / sep
	if int64(len(data))%sep != 0 {
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
