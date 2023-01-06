/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package pbc

import (
	"errors"
	"io"
	"math"
	"math/big"
	"os"
)

func GenProof(sigmas [][]byte, QSlice []QElement, matrix [][]byte, s int64, seg int64) ([]byte, [][]byte) {
	miu := make([][]byte, s)
	for j := 0; j < len(miu); j++ {
		sum := new(big.Int)
		for i := 0; i < len(QSlice); i++ {
			mij := new(big.Int).SetBytes(matrix[QSlice[i].I][int64(j)*seg : int64(j+1)*seg])
			vi := big.NewInt(QSlice[i].V)
			//Σ νimij (i,νi)∈Q
			sum.Add(sum, vi.Mul(vi, mij))
		}
		miu[j] = sum.Bytes()
	}

	sigma := new(big.Int)
	for i := 0; i < len(QSlice); i++ {
		sigmaI := new(big.Int).SetBytes(sigmas[QSlice[i].I])
		vi := new(big.Int).SetInt64(QSlice[i].V)
		sigma.Add(sigma, new(big.Int).Mul(vi, sigmaI))
	}

	return sigma.Bytes(), miu
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

func SplitV2(filePath string, sep int64, seg int64) (Data [][]byte, N int64, S int64, err error) {
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

	S = sep / seg
	if sep%seg != 0 {
		err = errors.New("chunk length is not divisible by segment length")
	}
	return
}
