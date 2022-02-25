package proof

import (
	ffi "github.com/CESSProject/cess-ffi"

	"github.com/filecoin-project/go-state-types/abi"
	prf "github.com/filecoin-project/specs-actors/actors/runtime/proof"
	cid "github.com/ipfs/go-cid"
)

//verify proof including PoRep and PoSt

// @title           VerifyFileOnce
// @description     verify the PoRep of service sectors (only execute once)
// @param           sectorId            			file info including peerID and fileID (now is sectorNum)
// @param           seed               				randomness
// @param           ticket              			randomness
// @param           SealProofType       			type of PoRep
// @param           preGeneratedUnsealedCIDs        PoRep intermediate params
// @param           sCIDs					        set of sealedCID
// @param           proofs					        set of PoRep proof
// @return          result
func VerifyFileOnce(sectorId SectorID, seed abi.InteractiveSealRandomness, ticket abi.SealRandomness, SealProofType abi.RegisteredSealProof, preGeneratedUnsealedCIDs []cid.Cid, sCIDs []cid.Cid, proofs [][]byte) bool {
	//loop for each service sector
	for i := 0; i < len(preGeneratedUnsealedCIDs); i++ {
		isValid, err := ffi.VerifySeal(prf.SealVerifyInfo{
			SectorID: abi.SectorID{
				Miner:  sectorId.PeerID,
				Number: sectorId.SectorNum,
			},
			SealedCID:             sCIDs[i],
			SealProof:             SealProofType,
			Proof:                 proofs[i],
			DealIDs:               []abi.DealID{},
			Randomness:            ticket,
			InteractiveRandomness: seed,
			UnsealedCID:           preGeneratedUnsealedCIDs[i],
		})
		RequireNoError(err)
		if isValid == false {
			return isValid
		}
	}
	return true
}

// @title           VerifyFileOnceForIdle
// @description     verify the PoRep of a idle sector (only execute once)
// @param           sectorId            			file info including peerID and fileID (now is sectorNum)
// @param           seed               				randomness
// @param           ticket              			randomness
// @param           SealProofType       			type of PoRep
// @param           sCIDs					        set of sealedCID
// @param           proofs					        set of PoRep proof
// @return          result
func VerifyFileOnceForIdle(sectorId SectorID, seed abi.InteractiveSealRandomness, ticket abi.SealRandomness, SealProofType abi.RegisteredSealProof, sCID cid.Cid, proof []byte) bool {

	//for idle sector, the unsealedCID can be computed in here
	preGeneratedUnsealedCIDForIdle, err := ffi.GenerateUnsealedCID(SealProofType, []abi.PieceInfo{})
	RequireNoError(err)
	isValid, err := ffi.VerifySeal(prf.SealVerifyInfo{
		SectorID: abi.SectorID{
			Miner:  sectorId.PeerID,
			Number: sectorId.SectorNum,
		},
		SealedCID:             sCID,
		SealProof:             SealProofType,
		Proof:                 proof,
		DealIDs:               []abi.DealID{},
		Randomness:            ticket,
		InteractiveRandomness: seed,
		UnsealedCID:           preGeneratedUnsealedCIDForIdle,
	})
	RequireNoError(err)

	return isValid
}

// @title           VerifyFileInterval
// @description     verify the PoSt of sectors (Periodically)
// @param           sectorId            			file info including peerID and fileID (now is sectorNum)
// @param           SealProofType       			type of PoRep
// @param           randomness               		randomness
// @param           sealedCIDs					    set of sealedCID
// @param           proofsWw					    set of PoSt proof
// @return          result
func VerifyFileInterval(sectorId SectorID, sealProofType abi.RegisteredSealProof, randomness []byte, sealedCIDs []cid.Cid, proofsWw []prf.PoStProof) bool {

	provingSet := make([]prf.SectorInfo, 0)

	for _, ps := range sealedCIDs {
		provingSet = append(provingSet, prf.SectorInfo{
			SealProof:    sealProofType,
			SectorNumber: sectorId.SectorNum,
			SealedCID:    ps,
		})
	}

	provingSet2 := NewSortedSectorInfo(provingSet)

	isValid, err := ffi.VerifyWindowPoSt(prf.WindowPoStVerifyInfo{
		Randomness:        randomness,
		Proofs:            proofsWw,
		ChallengedSectors: provingSet2,
		Prover:            sectorId.PeerID,
	})
	RequireNoError(err)

	return isValid
}
