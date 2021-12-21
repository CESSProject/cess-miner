package proof

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/filecoin-project/go-state-types/abi"
	prf "github.com/filecoin-project/specs-actors/actors/runtime/proof"
	chunk "github.com/ipfs/go-ipfs-chunker"
)

//common constants and functions

//intermediate param for PoRep
type PublicPiece []abi.PieceInfo

//
type SectorID struct {
	PeerID    abi.ActorID
	SectorNum abi.SectorNumber
}

const (
	//size of each chunk
	PieceFileSize int64 = 8323072
	//PoRep params for different type of sector
	SealProofType        abi.RegisteredSealProof = abi.RegisteredSealProof_StackedDrg8MiBV1
	SealProofTypeForIdle abi.RegisteredSealProof = abi.RegisteredSealProof_StackedDrg8MiBV1
	//PoSt params for different type of sector
	windowPostProofType        abi.RegisteredPoStProof = abi.RegisteredPoStProof_StackedDrgWindow8MiBV1
	windowPostProofTypeForIdle abi.RegisteredPoStProof = abi.RegisteredPoStProof_StackedDrgWindow8MiBV1
)

// @title           Chunking
// @description     to slice the file
// @param           fileDir             path of source file to slice
// @param           dir                 path to store sliced file
// @param           cs                  size of each chunk
// @return          number of slices
func Chunking(fileDir, dir string, cs int64) (num int, poi int64) {
	content, err := ioutil.ReadFile(fileDir)
	RequireNoError(err)
	chunksize := cs
	splitter := chunk.NewSizeSplitter(bytes.NewReader(content), chunksize)

	for n := 0; ; n++ {
		ck, _ := splitter.NextBytes()
		if len(ck) == 0 {
			num = n
			return
		}
		//if cap(ck) > len(ck) {
		//    fmt.Println("err3 happend!")
		//}

		arr := strings.SplitAfterN(fileDir, "/", -1)
		file := requireFile(dir, arr[len(arr)-1]+"_"+strconv.Itoa(n), ck)
		defer file.Close()

		if poi = int64(len(ck)); poi < chunksize {
			os.Truncate(file.Name(), chunksize)
		}

		_, err = file.Seek(0, 0)
	}
}

// @title           Padding
// @description     to pad the numbers of file to full the sector (currently a multiple of 8)
// @param           fileDir             path of source file to slice
// @param           dir                 path to store padded file
// @param           num                 index of creating file
// @param           cs                  size of each chunk
// @return          times of padding
func Padding(fileDir, dir string, num int, cs int64) int {
	times := (8 - num%8) % 8
	for n := times; n > 0; n-- {
		arr := strings.SplitAfterN(fileDir, "/", -1)
		file, err := os.Create(dir + arr[len(arr)-1] + "_" + strconv.Itoa(num-1+n))
		RequireNoError(err)
		err = file.Truncate(cs)
		file.Sync()
		_, err = file.Seek(0, 0)
	}
	return times
}

// @title           NewSortedSectorInfo
// @description     sort by SealedCID
// @param           sectorInfo             raw sectorInfo
// @return          sorted sectorInfo
func NewSortedSectorInfo(sectorInfo []prf.SectorInfo) []prf.SectorInfo {
	fn := func(i, j int) bool {
		return bytes.Compare(sectorInfo[i].SealedCID.Bytes(), sectorInfo[j].SealedCID.Bytes()) == -1
	}

	sort.Slice(sectorInfo, fn)

	return sectorInfo
}

func RequireNoError(err error, msgAndArgs ...interface{}) {
	if err != nil {
		fmt.Println("error happened!", err)
	}
}

//create new file
func requireFile(dir, name string, cet []byte) *os.File {
	file, err := os.Create(dir + name)
	RequireNoError(err)
	_, err = io.Copy(file, bytes.NewReader(cet))
	RequireNoError(err)
	file.Sync()
	_, err = file.Seek(0, 0)
	return file
}

func requireTempDirPath(prefix string) string {
	dir, err := ioutil.TempDir("", prefix)
	RequireNoError(err)
	return dir
}
