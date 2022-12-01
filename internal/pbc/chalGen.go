package pbc

import (
	"crypto/rand"
	"errors"
	"math/big"

	"github.com/Nik-U/pbc"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

func ChalGen(N int64, SharedParams string) []QElement {
	pairing, _ := pbc.NewPairingFromString(SharedParams)
	//Random number generated on the chain, length: len(Q)∈(0,Tag.N], number size: Q∈(0,Tag.N]
	l := new(big.Int)
	// Randomly select l blocks
	for {
		l, _ = rand.Int(rand.Reader, big.NewInt(N+1))
		if l.Cmp(big.NewInt(0)) == +1 {
			break
		}
	}
	challenge := make([]QElement, l.Int64())
	TagUnique := make(map[int64]struct{})
	for i := int64(0); i < l.Int64(); i++ {
		for {
			I, _ := rand.Int(rand.Reader, big.NewInt(N))
			_, ok := TagUnique[I.Int64()]
			if !ok {
				TagUnique[I.Int64()] = struct{}{}
				challenge[i].I = I.Int64()
				break
			} else {
				continue
			}
		}
		Q := pairing.NewZr().Rand().Bytes()
		challenge[i].V = Q
	}
	return challenge
}

// The key of ChallengeMap represents the serial number of the block to be challenged. Please start from 1 to represent the serial number of
// the block. For example, there are 40 files in total, and the serial number is [1,40]
// func PoDR2ChallengeGenerateFromChain(ChallengeMap *map[int]*big.Int, SharedParams string) []QElement {
// 	pairing, _ := pbc.NewPairingFromString(SharedParams)
// 	//Random number generated on the chain, length: len(Q)∈(0,Tag.N], number size: Q∈(0,Tag.N]
// 	l := new(big.Int)
// 	l.SetInt64(int64(len(*ChallengeMap)))
// 	challenge := make([]QElement, l.Int64())
// 	for i, q := range *ChallengeMap {
// 		I := big.NewInt(int64(i))
// 		if I.Cmp(big.NewInt(0)) == +1 {
// 			panic("Challenge block error, block sequence number cannot be 0")
// 		}
// 		challenge[i].I = I.Int64()
// 		Q := pairing.NewZr().SetBig(q).Bytes()
// 		challenge[i].V = Q
// 	}

// 	return challenge
// }

// The key of ChallengeMap represents the serial number of the block to be challenged. Please start from 1 to represent the serial number of
// the block. For example, there are 40 files in total, and the serial number is [1,40]
func PoDR2ChallengeGenerateFromChain(blockindex types.Bytes, blockrandom []types.Bytes) ([]QElement, error) {
	if len(blockindex) != len(blockrandom) {
		return nil, errors.New("The number of blocks and the number of random numbers are not equal")
	}
	challenge := make([]QElement, len(blockindex))
	for j := 0; j < len(blockindex); j++ {
		if int64(blockindex[j]) == 0 {
			challenge[j].I = 1
		} else {
			challenge[j].I = int64(blockindex[j])
		}
		challenge[j].V = blockrandom[j]
	}
	return challenge, nil
}
