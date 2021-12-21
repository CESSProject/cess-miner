//go:build cgo
// +build cgo

package ffi

import (
	"storage-mining/internal/cess-ffi/generated"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v5/actors/runtime/proof"
	"github.com/pkg/errors"
)

type FallbackChallenges struct {
	Sectors    []abi.SectorNumber
	Challenges map[abi.SectorNumber][]uint64
}

type VanillaProof []byte

// GenerateWinningPoStSectorChallenge
func GeneratePoStFallbackSectorChallenges(
	proofType abi.RegisteredPoStProof,
	minerID abi.ActorID,
	randomness abi.PoStRandomness,
	sectorIds []abi.SectorNumber,
) (*FallbackChallenges, error) {
	proverID, err := toProverID(minerID)
	if err != nil {
		return nil, err
	}

	pp, err := toCessRegisteredPoStProof(proofType)
	if err != nil {
		return nil, err
	}

	secIds := make([]uint64, len(sectorIds))
	for i, sid := range sectorIds {
		secIds[i] = uint64(sid)
	}

	resp := generated.CessGenerateFallbackSectorChallenges(
		pp, to32ByteArray(randomness), secIds, uint(len(secIds)),
		proverID,
	)
	resp.Deref()
	resp.IdsPtr = resp.IdsPtr[:resp.IdsLen]
	resp.ChallengesPtr = resp.ChallengesPtr[:resp.ChallengesLen]

	defer generated.CessDestroyGenerateFallbackSectorChallengesResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return nil, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	// copy from C memory space to Go

	var out FallbackChallenges
	out.Sectors = make([]abi.SectorNumber, resp.IdsLen)
	out.Challenges = make(map[abi.SectorNumber][]uint64)
	stride := int(resp.ChallengesStride)
	for idx := range resp.IdsPtr {
		secNum := abi.SectorNumber(resp.IdsPtr[idx])
		out.Sectors[idx] = secNum
		out.Challenges[secNum] = append([]uint64{}, resp.ChallengesPtr[idx*stride:(idx+1)*stride]...)
	}

	return &out, nil
}

func GenerateSingleVanillaProof(
	replica PrivateSectorInfo,
	challange []uint64,
) ([]byte, error) {

	rep, free, err := toCessPrivateReplicaInfo(replica)
	if err != nil {
		return nil, err
	}
	defer free()

	resp := generated.CessGenerateSingleVanillaProof(rep, challange, uint(len(challange)))
	resp.Deref()
	defer generated.CessDestroyGenerateSingleVanillaProofResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return nil, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	resp.VanillaProof.Deref()

	return copyBytes(resp.VanillaProof.ProofPtr, resp.VanillaProof.ProofLen), nil
}

func GenerateWinningPoStWithVanilla(
	proofType abi.RegisteredPoStProof,
	minerID abi.ActorID,
	randomness abi.PoStRandomness,
	proofs [][]byte,
) ([]proof.PoStProof, error) {
	pp, err := toCessRegisteredPoStProof(proofType)
	if err != nil {
		return nil, err
	}

	proverID, err := toProverID(minerID)
	if err != nil {
		return nil, err
	}
	fproofs, discard := toVanillaProofs(proofs)
	defer discard()

	resp := generated.CessGenerateWinningPostWithVanilla(
		pp,
		to32ByteArray(randomness),
		proverID,
		fproofs, uint(len(proofs)),
	)
	resp.Deref()
	resp.ProofsPtr = make([]generated.CessPoStProof, resp.ProofsLen)
	resp.Deref()

	defer generated.CessDestroyGenerateWinningPostResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return nil, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	out, err := fromCessPoStProofs(resp.ProofsPtr)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func GenerateWindowPoStWithVanilla(
	proofType abi.RegisteredPoStProof,
	minerID abi.ActorID,
	randomness abi.PoStRandomness,
	proofs [][]byte,
) ([]proof.PoStProof, error) {
	pp, err := toCessRegisteredPoStProof(proofType)
	if err != nil {
		return nil, err
	}

	proverID, err := toProverID(minerID)
	if err != nil {
		return nil, err
	}
	fproofs, discard := toVanillaProofs(proofs)
	defer discard()

	resp := generated.CessGenerateWindowPostWithVanilla(
		pp,
		to32ByteArray(randomness),
		proverID,
		fproofs, uint(len(proofs)),
	)
	resp.Deref()
	resp.ProofsPtr = make([]generated.CessPoStProof, resp.ProofsLen)
	resp.Deref()

	defer generated.CessDestroyGenerateWindowPostResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return nil, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	out, err := fromCessPoStProofs(resp.ProofsPtr)
	if err != nil {
		return nil, err
	}

	return out, nil
}

type PartitionProof proof.PoStProof

func GenerateSinglePartitionWindowPoStWithVanilla(
	proofType abi.RegisteredPoStProof,
	minerID abi.ActorID,
	randomness abi.PoStRandomness,
	proofs [][]byte,
	partitionIndex uint,
) (*PartitionProof, error) {
	pp, err := toCessRegisteredPoStProof(proofType)
	if err != nil {
		return nil, err
	}

	proverID, err := toProverID(minerID)
	if err != nil {
		return nil, err
	}
	fproofs, discard := toVanillaProofs(proofs)
	defer discard()

	resp := generated.CessGenerateSingleWindowPostWithVanilla(
		pp,
		to32ByteArray(randomness),
		proverID,
		fproofs, uint(len(proofs)),
		partitionIndex,
	)
	resp.Deref()

	defer generated.CessDestroyGenerateSingleWindowPostWithVanillaResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return nil, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	dpp, err := fromCessRegisteredPoStProof(resp.PartitionProof.RegisteredProof)
	if err != nil {
		return nil, err
	}

	out := PartitionProof{
		PoStProof:  dpp,
		ProofBytes: copyBytes(resp.PartitionProof.ProofPtr, resp.PartitionProof.ProofLen),
	}

	return &out, nil
}

func MergeWindowPoStPartitionProofs(
	proofType abi.RegisteredPoStProof,
	partitionProofs []PartitionProof,
) (*proof.PoStProof, error) {
	pp, err := toCessRegisteredPoStProof(proofType)
	if err != nil {
		return nil, err
	}

	fproofs, discard, err := toPartitionProofs(partitionProofs)
	defer discard()
	if err != nil {
		return nil, err
	}

	resp := generated.CessMergeWindowPostPartitionProofs(
		pp,
		fproofs, uint(len(fproofs)),
	)
	resp.Deref()

	defer generated.CessDestroyMergeWindowPostPartitionProofsResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return nil, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	dpp, err := fromCessRegisteredPoStProof(resp.Proof.RegisteredProof)
	if err != nil {
		return nil, err
	}

	out := proof.PoStProof{
		PoStProof:  dpp,
		ProofBytes: copyBytes(resp.Proof.ProofPtr, resp.Proof.ProofLen),
	}

	return &out, nil
}
