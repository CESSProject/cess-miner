package proof

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"regexp"

	ffi "github.com/CESSProject/cess-ffi"

	"github.com/filecoin-project/go-state-types/abi"
	prf "github.com/filecoin-project/specs-actors/actors/runtime/proof"
	cid "github.com/ipfs/go-cid"
)

// @title           GetPoRep
// @description     generate the PoRep of service sectors (a file)
// @param           sectorId            			file info including peerID and fileID (now is sectorNum)
// @param           seed               				randomness
// @param           ticket              			randomness
// @param           SealProofType       			type of PoRep
// @param           publicPieces       				PoRep intermediate params(set of PublicPiece)
// @param           preGeneratedUnsealedCIDs        PoRep intermediate params(set of UnsealedCID)
// @param           targetPath					    source file
// @param           sealedDir					    path to generate sealedFile
// @param           cachedDir					    path to generate cachedFile
// @return          sCIDs							set of sealedCID
// @return          proofsPp						set of PoRep proof
func GetPoRep(sectorId SectorID, seed abi.InteractiveSealRandomness, ticket abi.SealRandomness, SealProofType abi.RegisteredSealProof, preGeneratedUnsealedCIDs []cid.Cid, targetPath, sealedDir, cachedDir string) (sCIDs []cid.Cid, proofsPp [][]byte) {

	tDir := requireTempDirPath("proof-cache-dir")
	defer os.RemoveAll(tDir)

	sCIDs = make([]cid.Cid, 0)
	proofsPp = make([][]byte, 0)

	tempDir := tDir + "/"

	//slice and pad
	_, _ = Chunking(targetPath, tempDir, PieceFileSize)
	//_ = Padding(targetPath, tempDir, s, PieceFileSize)

	files, err := ioutil.ReadDir(tDir)
	RequireNoError(err)

	//confirm the number of sectors for a file
	fs := make([]os.FileInfo, 0)
	for n := 0; n < len(files); n++ {
		fs = append(fs, files[n])
	}

	//loop the sectors to compute proofs stored in sCIDs and proofsPp
	for i := 0; i < len(fs); i++ {

		//put each sector's sealedFile and cachedFile into individual dir
		//generate tmp path
		var sectorCacheDirPath string
		if os.Mkdir(cachedDir+"/"+"tmp", os.ModePerm) == nil {
			sectorCacheDirPath = cachedDir + "/" + "tmp"
		}

		sealedSectorFile := requireFile(sealedDir+"/", "tmp", []byte{})
		defer sealedSectorFile.Close()

		stagedSectorFile := requireTempFile(bytes.NewReader([]byte{}), 0)
		defer stagedSectorFile.Close()

		//loop the slices
		osf, err := os.Open(tempDir + fs[i].Name())
		RequireNoError(err)
		fi, err := osf.Stat()
		RequireNoError(err)
		_, _, err = ffi.WriteWithoutAlignment(SealProofType, osf, abi.UnpaddedPieceSize(fi.Size()), stagedSectorFile)
		RequireNoError(err)

		publicPiece := []abi.PieceInfo{{
			Size:     abi.UnpaddedPieceSize(PieceFileSize).Padded(),
			PieceCID: preGeneratedUnsealedCIDs[i],
		}}

		sealedCID, proof, err := computePoRep(sectorId, seed, ticket, SealProofType, publicPiece, stagedSectorFile.Name(), sealedSectorFile.Name(), sectorCacheDirPath)

		sCIDs = append(sCIDs, sealedCID)
		proofsPp = append(proofsPp, proof)

		//regular matching, delete files that are no longer needed from the cache directory
		filesCa, _ := ioutil.ReadDir(sectorCacheDirPath)
		r := regexp.MustCompile("(^sc-02-data-tree-r)|(_aux$)")
		for _, f := range filesCa {
			if !(r.MatchString(f.Name())) {
				err = os.Remove(sectorCacheDirPath + "/" + f.Name())
				RequireNoError(err)
			}
		}

		//rename the tmp dir according its sealedCID
		err = os.Rename(sealedSectorFile.Name(), sealedDir+"/"+sealedCID.String())
		RequireNoError(err)
		err = os.Rename(sectorCacheDirPath, cachedDir+"/"+sealedCID.String())
		RequireNoError(err)
	}
	return
}

// @title           GetPoRepForIdle
// @description     generate the PoRep of a idle sector (file with zeros)
// @param           sectorId            			file info including peerID and fileID (now is sectorNum)
// @param           seed               				randomness
// @param           ticket              			randomness
// @param           SealProofType       			type of PoRep
// @param           targetPath					    source file
// @param           sealedDir					    path to generate sealedFile
// @param           cachedDir					    path to generate cachedFile
// @return          sealedCID						set of sealedCID
// @return          proof							set of PoRep proof
func GetPoRepForIdle(sectorId SectorID, seed abi.InteractiveSealRandomness, ticket abi.SealRandomness, SealProofType abi.RegisteredSealProof, targetPath, sealedDir, cachedDir string) (sealedCID cid.Cid, proof []byte) {

	//put each sector's sealedFile and cachedFile into individual dir
	//generate tmp path
	var sectorCacheDirPath string
	if os.Mkdir(cachedDir+"/"+"tmp", os.ModePerm) == nil {
		sectorCacheDirPath = cachedDir + "/" + "tmp"
	}

	sealedSectorFile := requireFile(sealedDir+"/", "tmp", []byte{})
	defer sealedSectorFile.Close()

	sealedCID, proof, err := computePoRep(sectorId, seed, ticket, SealProofType, []abi.PieceInfo{}, targetPath, sealedSectorFile.Name(), sectorCacheDirPath)
	RequireNoError(err)

	//regular matching, delete files that are no longer needed from the cache directory
	filesCa, _ := ioutil.ReadDir(sectorCacheDirPath)
	r := regexp.MustCompile("(^sc-02-data-tree-r)|(_aux$)")
	for _, f := range filesCa {
		if !(r.MatchString(f.Name())) {
			err = os.Remove(sectorCacheDirPath + "/" + f.Name())
			RequireNoError(err)
		}
	}

	//rename the tmp dir according its sealedCID
	err = os.Rename(sealedSectorFile.Name(), sealedDir+"/"+sealedCID.String())
	RequireNoError(err)
	err = os.Rename(sectorCacheDirPath, cachedDir+"/"+sealedCID.String())
	RequireNoError(err)
	return
}

// @title           GetPoSt
// @description     generate the PoSt of sectors (idle or service)
// @param           sectorId            			file info including peerID and fileID (now is sectorNum)
// @param           windowPostProofType       		type of PoSt
// @param           sealedCIDs       				set of sealedCID
// @param           randomness        				randomness for PoSt
// @param           sealedDir					    path to generate sealedFile
// @param           cachedDir					    path to generate cachedFile
// @return          proofsWw						set of PoSt proof
// @return          faultySectors					set of faulty proof
func GetPoSt(sectorId SectorID, windowPostProofType abi.RegisteredPoStProof, sealedCIDs []cid.Cid, randomness []byte, sealedDir, cachedDir string) (proofsWw []prf.PoStProof, faultySectors []abi.SectorNumber, err error) {

	psInfos := make([]ffi.PrivateSectorInfo, 0)

	for _, sc := range sealedCIDs {
		psInfos = append(psInfos, ffi.PrivateSectorInfo{
			SectorInfo: prf.SectorInfo{
				SealProof:    SealProofType,
				SectorNumber: sectorId.SectorNum,
				SealedCID:    sc,
			},
			CacheDirPath:     cachedDir + "/" + sc.String(),
			PoStProofType:    windowPostProofType,
			SealedSectorPath: sealedDir + "/" + sc.String(),
		})
	}

	privateInfo2 := ffi.NewSortedPrivateSectorInfo(psInfos)
	proofsWw, faultySectors, err = ffi.GenerateWindowPoSt(sectorId.PeerID, privateInfo2, randomness)
	RequireNoError(err)

	return
}

func UnsealToFile(fileName string, sealProofType abi.RegisteredSealProof, targetPath, sealedDir, cachedDir string, sectorId SectorID, ticket abi.SealRandomness, preGeneratedUnsealedCIDs, sealedCIDs []cid.Cid, poi int64) {

	//unsealingSectorsDir := requireTempDirPath("unsealing-sectors")
	//defer os.RemoveAll(unsealingSectorsDir)

	tempDir := targetPath + "/"

	unsealOutputFileA := requireFile(tempDir, fileName, []byte{})
	defer unsealOutputFileA.Close()

	for ix, scid := range sealedCIDs {
		sectorCacheDirPath := cachedDir + "/" + scid.String()
		sealedSectorFile, err := os.Open(sealedDir + "/" + scid.String())
		RequireNoError(err)
		RequireNoError(err)

		if ix < (len(sealedCIDs) - 1) {
			RequireNoError(ffi.Unseal(sealProofType, sectorCacheDirPath, sealedSectorFile, unsealOutputFileA, sectorId.SectorNum, sectorId.PeerID, ticket, preGeneratedUnsealedCIDs[ix]))
		} else {
			RequireNoError(ffi.UnsealRange(sealProofType, sectorCacheDirPath, sealedSectorFile, unsealOutputFileA, sectorId.SectorNum, sectorId.PeerID, ticket, preGeneratedUnsealedCIDs[ix], 0, uint64(poi)))
		}
	}

	_, err := unsealOutputFileA.Seek(0, 0)
	RequireNoError(err)
}

func computePoRep(sectorId SectorID, seed abi.InteractiveSealRandomness, ticket abi.SealRandomness, SealProofType abi.RegisteredSealProof, publicPieces PublicPiece, targetPath, sealedSectorFile, sectorCacheDirPath string) (sealedCID cid.Cid, proof []byte, err error) {
	sealPreCommitPhase1Output, err := ffi.SealPreCommitPhase1(SealProofType, sectorCacheDirPath, targetPath, sealedSectorFile, sectorId.SectorNum, sectorId.PeerID, ticket, publicPieces)
	RequireNoError(err)
	sealedCID, unsealedCID, err := ffi.SealPreCommitPhase2(sealPreCommitPhase1Output, sectorCacheDirPath, sealedSectorFile)
	RequireNoError(err)
	//if !(unsealedCID.Equals(preGeneratedUnsealedCIDs[i])) {
	//	fmt.Println("prover and verifier should agree on data commitment")
	//}
	// commit the sector
	sealCommitPhase1Output, err := ffi.SealCommitPhase1(SealProofType, sealedCID, unsealedCID, sectorCacheDirPath, sealedSectorFile, sectorId.SectorNum, sectorId.PeerID, ticket, seed, publicPieces)
	RequireNoError(err)
	proof, err = ffi.SealCommitPhase2(sealCommitPhase1Output, sectorId.SectorNum, sectorId.PeerID)
	RequireNoError(err)
	return
}

func requireTempFile(fileContentsReader io.Reader, size uint64) *os.File {
	file, err := ioutil.TempFile("", "")
	RequireNoError(err)

	_, err = io.Copy(file, fileContentsReader)
	RequireNoError(err)

	RequireNoError(file.Sync())

	// seek to the beginning
	_, err = file.Seek(0, 0)
	RequireNoError(err)

	return file
}
