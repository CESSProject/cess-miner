package proof

import (
	"cess-bucket/tools"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Nik-U/pbc"
)

func (commit PoDR2Commit) PoDR2ProofCommit(ssk []byte, sharedParams string, segmentSize int64) <-chan PoDR2CommitResponse {
	responseCh := make(chan PoDR2CommitResponse, 1)
	var res PoDR2CommitResponse
	pairing, _ := pbc.NewPairingFromString(sharedParams)
	privateKey := pairing.NewZr().SetBytes(ssk)
	file, err := os.Open(commit.FilePath)
	if err != nil {
		panic(err)
	}
	matrix, s, n, err := tools.Split(file, commit.BlockSize)
	T := FileTagT{}
	T.T0.N = int64(n)
	T.T0.Name = pairing.NewZr().Rand().Bytes()
	U_num := s / segmentSize
	if s%segmentSize != 0 {
		U_num++
	}
	T.T0.U = make([][]byte, U_num)
	fmt.Println("start generate U", U_num)
	//for i := int64(0); i < s; i++ {
	//	result := pairing.NewG2().Rand()
	//	T.T0.U = append(T.T0.U, result)
	//}

	for i := int64(0); i < U_num; i++ {
		result := pairing.NewG2().Rand().Bytes()
		T.T0.U[i] = result
	}
	fmt.Println("end generate U")
	tmp, err := json.Marshal(T.T0)
	if err != nil {
		panic(err)
	}
	//var tau_zero_bytes bytes.Buffer
	//enc := gob.NewEncoder(&tau_zero_bytes)
	//err = enc.Encode(T.T0)
	//if err != nil {
	//	res.StatueMsg.StatusCode = paramv1.Error
	//	res.StatueMsg.Msg = "encode tau_zero_bytes error" + err.Error()
	//	responseCh <- res
	//	return responseCh
	//}
	fmt.Println("start hash256")
	hashed_t_0 := pairing.NewG2().SetFromStringHash(string(tmp), sha256.New())
	t_0_signature := pairing.NewG2().PowZn(hashed_t_0, privateKey)
	T.Signature = t_0_signature.Bytes()
	fmt.Println("end hash256")
	res.T = T
	res.Sigmas = make([][]byte, n)
	//g1wait := make(chan struct{}, n)
	fmt.Println("start to calculate sigma,Alpha:", privateKey)
	fmt.Println("start to calculate sigma,n:", n)
	fmt.Println("start to calculate sigma,s:", s)
	for i := int64(0); i < int64(n); i++ {

		res.Sigmas[i] = GenerateAuthenticator(i, s, res.T.T0, matrix[i], privateKey, pairing, segmentSize)
		//g1wait <- struct{}{}
		//}(i)
	}
	//for i := int64(0); i < n; i++ {
	//	<-g1wait
	//}
	res.StatueMsg.StatusCode = Success
	res.StatueMsg.Msg = "PoDR2ProofCommit success"
	responseCh <- res
	return responseCh
}

func GenerateAuthenticator(i int64, s int64, T0 T0, piece []byte, Alpha *pbc.Element, pairing *pbc.Pairing, segmentSize int64) []byte {
	//H(name||i)
	hash_name_i := hashNameI(pairing.NewZr().SetBytes(T0.Name), i+1, pairing)

	productory := pairing.NewG2()
	//for j := int64(0); j < s; j++ {
	//	//mij
	//	piece_sigle := pairing.NewZr().SetFromHash([]byte{piece[j]})
	//	//uj^mij
	//	productory.Mul(productory, pairing.NewG2().PowZn(T0.U[j], piece_sigle))
	//}
	U_num := s / segmentSize
	if s%segmentSize != 0 {
		U_num++
	}
	for j := int64(0); j < U_num; j++ {
		if j == U_num-1 {
			piece_sigle := pairing.NewZr().SetFromHash(piece[j*segmentSize:])
			//uj^mij
			productory.Mul(productory, pairing.NewG2().PowZn(pairing.NewG2().SetBytes(T0.U[j]), piece_sigle))
			continue
		}
		//mij
		piece_sigle := pairing.NewZr().SetFromHash(piece[j*segmentSize : (j+1)*segmentSize])
		//uj^mij
		productory.Mul(productory, pairing.NewG2().PowZn(pairing.NewG2().SetBytes(T0.U[j]), piece_sigle))
	}
	//H(name||i) · uj^mij
	innerProduct := pairing.NewG2().Mul(productory, &hash_name_i)
	return pairing.NewG2().PowZn(innerProduct, Alpha).Bytes()
}

func hashNameI(name *pbc.Element, i int64, pairing *pbc.Pairing) pbc.Element {
	i_bytes := make([]byte, 4)
	binary.PutVarint(i_bytes, i)
	hashArgument := append(name.Bytes(), i_bytes...)
	hash_array := sha256.Sum256(hashArgument)

	hash_res := pairing.NewG2().SetFromHash(hash_array[:])
	//hash_res := new(pbc.Element).SetFromStringHash(string(HashTemp), sha256.New())
	return *hash_res
}
