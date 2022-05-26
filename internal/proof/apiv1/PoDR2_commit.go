package proof

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/Nik-U/pbc"
)

func GenerateAuthenticator(i int64, s int64, T0 T0, piece []byte, Alpha *pbc.Element, pairing *pbc.Pairing, segmentSize int64) []byte {
	//H(name||i)
	hash_name_i := hashNameI(pairing.NewZr().SetBytes(T0.Name), i+1, pairing)

	productory := pairing.NewG2()
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
	return *hash_res
}
