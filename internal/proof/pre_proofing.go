package proof

import (
	"io/ioutil"
	"os"

	ffi "github.com/CESSProject/cess-ffi"

	"github.com/filecoin-project/go-state-types/abi"
	cid "github.com/ipfs/go-cid"
)

//functions for platform to compute the PoRep intermediate params

// @title           GetPrePoRep
// @description     compute the PoRep intermediate params for one file
// @param           dir                         file for PoRep
// @return          publicPieces                set of PublicPiece
// @return          preGeneratedUnsealedCIDs    set of UnsealedCID
func GetPrePoRep(dir string) (publicPieces []PublicPiece, preGeneratedUnsealedCIDs []cid.Cid, poi int64) {

	tDir := requireTempDirPath("proof-cache-dir")
	defer os.RemoveAll(tDir)

	tempDir := tDir + "/"

	_, poi = Chunking(dir, tempDir, PieceFileSize)
	// _ = Padding(dir, tempDir, s, PieceFileSize)

	pieces, err := GeneratePieces(tempDir)
	RequireNoError(err)

	publicPieces, preGeneratedUnsealedCIDs = ComposePieces(pieces)

	return
}

//generate public piece for each sector
func GeneratePieces(dir string) (pi []abi.PieceInfo, err error) {
	pi = make([]abi.PieceInfo, 0)
	files, err := ioutil.ReadDir(dir)
	for _, f := range files {
		osf, err := os.Open(dir + f.Name())
		RequireNoError(err)
		fi, err := osf.Stat()
		RequireNoError(err)
		pieceCID_f, err := ffi.GeneratePieceCIDFromFile(SealProofType, osf, abi.UnpaddedPieceSize(fi.Size()))
		RequireNoError(err)
		piece := abi.PieceInfo{Size: abi.UnpaddedPieceSize(fi.Size()).Padded(), PieceCID: pieceCID_f}
		pi = append(pi, piece)
		_, err = osf.Seek(0, 0)
		RequireNoError(err)
	}
	return
}

//compose public pieces of a file
func ComposePieces(pis []abi.PieceInfo) (pps []PublicPiece, preCIDs []cid.Cid) {
	pps = make([]PublicPiece, 0)
	preCIDs = make([]cid.Cid, 0)
	for n := 0; n < len(pis); n++ {
		pps = append(pps, pis[n:n+1])
		preGeneratedUnsealedCID, err := ffi.GenerateUnsealedCID(SealProofType, pis[n:n+1])
		RequireNoError(err)
		preCIDs = append(preCIDs, preGeneratedUnsealedCID)
	}
	return
}
