//go:build cgo
// +build cgo

package ffi

// #cgo LDFLAGS: ${SRCDIR}/libcesscrypto.a
// #cgo pkg-config: ${SRCDIR}/cesscrypto.pc
// #include "./cesscrypto.h"
import "C"
import (
	"os"
	"runtime"
	"unsafe"

	"github.com/filecoin-project/go-address"
	commcid "github.com/filecoin-project/go-fil-commcid"
	"github.com/filecoin-project/go-state-types/abi"
	proof5 "github.com/filecoin-project/specs-actors/v5/actors/runtime/proof"
	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"golang.org/x/xerrors"

	"storage-mining/internal/cess-ffi/generated"
)

// VerifySeal returns true if the sealing operation from which its inputs were
// derived was valid, and false if not.
func VerifySeal(info proof5.SealVerifyInfo) (bool, error) {
	sp, err := toCessRegisteredSealProof(info.SealProof)
	if err != nil {
		return false, err
	}

	commR, err := to32ByteCommR(info.SealedCID)
	if err != nil {
		return false, err
	}

	commD, err := to32ByteCommD(info.UnsealedCID)
	if err != nil {
		return false, err
	}

	proverID, err := toProverID(info.Miner)
	if err != nil {
		return false, err
	}

	resp := generated.CessVerifySeal(sp, commR, commD, proverID, to32ByteArray(info.Randomness), to32ByteArray(info.InteractiveRandomness), uint64(info.SectorID.Number), info.Proof, uint(len(info.Proof)))
	resp.Deref()

	defer generated.CessDestroyVerifySealResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return false, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	return resp.IsValid, nil
}

func VerifyAggregateSeals(aggregate proof5.AggregateSealVerifyProofAndInfos) (bool, error) {
	if len(aggregate.Infos) == 0 {
		return false, xerrors.New("no seal verify infos")
	}

	spt := aggregate.SealProof // todo assuming this needs to be the same for all sectors, potentially makes sense to put in AggregateSealVerifyProofAndInfos
	inputs := make([]generated.CessAggregationInputs, len(aggregate.Infos))

	for i, info := range aggregate.Infos {
		commR, err := to32ByteCommR(info.SealedCID)
		if err != nil {
			return false, err
		}

		commD, err := to32ByteCommD(info.UnsealedCID)
		if err != nil {
			return false, err
		}

		inputs[i] = generated.CessAggregationInputs{
			CommR:    commR,
			CommD:    commD,
			SectorId: uint64(info.Number),
			Ticket:   to32ByteArray(info.Randomness),
			Seed:     to32ByteArray(info.InteractiveRandomness),
		}
	}

	sp, err := toCessRegisteredSealProof(spt)
	if err != nil {
		return false, err
	}

	proverID, err := toProverID(aggregate.Miner)
	if err != nil {
		return false, err
	}

	rap, err := toCessRegisteredAggregationProof(aggregate.AggregateProof)
	if err != nil {
		return false, err
	}

	resp := generated.CessVerifyAggregateSealProof(sp, rap, proverID, aggregate.Proof, uint(len(aggregate.Proof)), inputs, uint(len(inputs)))
	resp.Deref()

	defer generated.CessDestroyVerifyAggregateSealResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return false, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	return resp.IsValid, nil
}

// VerifyWinningPoSt returns true if the Winning PoSt-generation operation from which its
// inputs were derived was valid, and false if not.
func VerifyWinningPoSt(info proof5.WinningPoStVerifyInfo) (bool, error) {
	cessPublicReplicaInfos, cessPublicReplicaInfosLen, err := toCessPublicReplicaInfos(info.ChallengedSectors, "winning")
	if err != nil {
		return false, errors.Wrap(err, "failed to create public replica info array for FFI")
	}

	cessPoStProofs, cessPoStProofsLen, free, err := toCessPoStProofs(info.Proofs)
	if err != nil {
		return false, errors.Wrap(err, "failed to create PoSt proofs array for FFI")
	}
	defer free()

	proverID, err := toProverID(info.Prover)
	if err != nil {
		return false, err
	}

	resp := generated.CessVerifyWinningPost(
		to32ByteArray(info.Randomness),
		cessPublicReplicaInfos,
		cessPublicReplicaInfosLen,
		cessPoStProofs,
		cessPoStProofsLen,
		proverID,
	)
	resp.Deref()

	defer generated.CessDestroyVerifyWinningPostResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return false, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	return resp.IsValid, nil
}

// VerifyWindowPoSt returns true if the Winning PoSt-generation operation from which its
// inputs were derived was valid, and false if not.
func VerifyWindowPoSt(info proof5.WindowPoStVerifyInfo) (bool, error) {
	cessPublicReplicaInfos, cessPublicReplicaInfosLen, err := toCessPublicReplicaInfos(info.ChallengedSectors, "window")
	if err != nil {
		return false, errors.Wrap(err, "failed to create public replica info array for FFI")
	}

	cessPoStProofs, cessPoStProofsLen, free, err := toCessPoStProofs(info.Proofs)
	if err != nil {
		return false, errors.Wrap(err, "failed to create PoSt proofs array for FFI")
	}
	defer free()

	proverID, err := toProverID(info.Prover)
	if err != nil {
		return false, err
	}

	resp := generated.CessVerifyWindowPost(
		to32ByteArray(info.Randomness),
		cessPublicReplicaInfos, cessPublicReplicaInfosLen,
		cessPoStProofs, cessPoStProofsLen,
		proverID,
	)
	resp.Deref()

	defer generated.CessDestroyVerifyWindowPostResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return false, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	return resp.IsValid, nil
}

// GeneratePieceCommitment produces a piece commitment for the provided data
// stored at a given path.
func GeneratePieceCID(proofType abi.RegisteredSealProof, piecePath string, pieceSize abi.UnpaddedPieceSize) (cid.Cid, error) {
	pieceCesse, err := os.Open(piecePath)
	if err != nil {
		return cid.Undef, err
	}

	pcd, err := GeneratePieceCIDFromFile(proofType, pieceCesse, pieceSize)
	if err != nil {
		return cid.Undef, pieceCesse.Close()
	}

	return pcd, pieceCesse.Close()
}

// GenerateDataCommitment produces a commitment for the sector containing the
// provided pieces.
func GenerateUnsealedCID(proofType abi.RegisteredSealProof, pieces []abi.PieceInfo) (cid.Cid, error) {
	sp, err := toCessRegisteredSealProof(proofType)
	if err != nil {
		return cid.Undef, err
	}

	cessPublicPieceInfos, cessPublicPieceInfosLen, err := toCessPublicPieceInfos(pieces)
	if err != nil {
		return cid.Undef, err
	}

	resp := generated.CessGenerateDataCommitment(sp, cessPublicPieceInfos, cessPublicPieceInfosLen)
	resp.Deref()

	defer generated.CessDestroyGenerateDataCommitmentResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return cid.Undef, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	return commcid.DataCommitmentV1ToCID(resp.CommD[:])
}

// GeneratePieceCIDFromFile produces a piece CID for the provided data stored in
//a given file.
func GeneratePieceCIDFromFile(proofType abi.RegisteredSealProof, pieceCesse *os.File, pieceSize abi.UnpaddedPieceSize) (cid.Cid, error) {
	sp, err := toCessRegisteredSealProof(proofType)
	if err != nil {
		return cid.Undef, err
	}

	pieceFd := pieceCesse.Fd()
	defer runtime.KeepAlive(pieceCesse)

	resp := generated.CessGeneratePieceCommitment(sp, int32(pieceFd), uint64(pieceSize))
	resp.Deref()

	defer generated.CessDestroyGeneratePieceCommitmentResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return cid.Undef, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	return commcid.PieceCommitmentV1ToCID(resp.CommP[:])
}

// WriteWithAlignment
func WriteWithAlignment(
	proofType abi.RegisteredSealProof,
	pieceCesse *os.File,
	pieceBytes abi.UnpaddedPieceSize,
	stagedSectorCesse *os.File,
	existingPieceSizes []abi.UnpaddedPieceSize,
) (leftAlignment, total abi.UnpaddedPieceSize, pieceCID cid.Cid, retErr error) {
	sp, err := toCessRegisteredSealProof(proofType)
	if err != nil {
		return 0, 0, cid.Undef, err
	}

	pieceFd := pieceCesse.Fd()
	defer runtime.KeepAlive(pieceCesse)

	stagedSectorFd := stagedSectorCesse.Fd()
	defer runtime.KeepAlive(stagedSectorCesse)

	cessExistingPieceSizes, cessExistingPieceSizesLen := toCessExistingPieceSizes(existingPieceSizes)

	resp := generated.CessWriteWithAlignment(sp, int32(pieceFd), uint64(pieceBytes), int32(stagedSectorFd), cessExistingPieceSizes, cessExistingPieceSizesLen)
	resp.Deref()

	defer generated.CessDestroyWriteWithAlignmentResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return 0, 0, cid.Undef, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	commP, errCommpSize := commcid.PieceCommitmentV1ToCID(resp.CommP[:])
	if errCommpSize != nil {
		return 0, 0, cid.Undef, errCommpSize
	}

	return abi.UnpaddedPieceSize(resp.LeftAlignmentUnpadded), abi.UnpaddedPieceSize(resp.TotalWriteUnpadded), commP, nil
}

// WriteWithoutAlignment
func WriteWithoutAlignment(
	proofType abi.RegisteredSealProof,
	pieceCesse *os.File,
	pieceBytes abi.UnpaddedPieceSize,
	stagedSectorCesse *os.File,
) (abi.UnpaddedPieceSize, cid.Cid, error) {
	sp, err := toCessRegisteredSealProof(proofType)
	if err != nil {
		return 0, cid.Undef, err
	}

	pieceFd := pieceCesse.Fd()
	defer runtime.KeepAlive(pieceCesse)

	stagedSectorFd := stagedSectorCesse.Fd()
	defer runtime.KeepAlive(stagedSectorCesse)

	resp := generated.CessWriteWithoutAlignment(sp, int32(pieceFd), uint64(pieceBytes), int32(stagedSectorFd))
	resp.Deref()

	defer generated.CessDestroyWriteWithoutAlignmentResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return 0, cid.Undef, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	commP, errCommpSize := commcid.PieceCommitmentV1ToCID(resp.CommP[:])
	if errCommpSize != nil {
		return 0, cid.Undef, errCommpSize
	}

	return abi.UnpaddedPieceSize(resp.TotalWriteUnpadded), commP, nil
}

// SealPreCommitPhase1
func SealPreCommitPhase1(
	proofType abi.RegisteredSealProof,
	cacheDirPath string,
	stagedSectorPath string,
	sealedSectorPath string,
	sectorNum abi.SectorNumber,
	minerID abi.ActorID,
	ticket abi.SealRandomness,
	pieces []abi.PieceInfo,
) (phase1Output []byte, err error) {
	sp, err := toCessRegisteredSealProof(proofType)
	if err != nil {
		return nil, err
	}

	proverID, err := toProverID(minerID)
	if err != nil {
		return nil, err
	}

	cessPublicPieceInfos, cessPublicPieceInfosLen, err := toCessPublicPieceInfos(pieces)
	if err != nil {
		return nil, err
	}

	resp := generated.CessSealPreCommitPhase1(sp, cacheDirPath, stagedSectorPath, sealedSectorPath, uint64(sectorNum), proverID, to32ByteArray(ticket), cessPublicPieceInfos, cessPublicPieceInfosLen)
	resp.Deref()

	defer generated.CessDestroySealPreCommitPhase1Response(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return nil, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	return copyBytes(resp.SealPreCommitPhase1OutputPtr, resp.SealPreCommitPhase1OutputLen), nil
}

// SealPreCommitPhase2
func SealPreCommitPhase2(
	phase1Output []byte,
	cacheDirPath string,
	sealedSectorPath string,
) (sealedCID cid.Cid, unsealedCID cid.Cid, err error) {
	resp := generated.CessSealPreCommitPhase2(phase1Output, uint(len(phase1Output)), cacheDirPath, sealedSectorPath)
	resp.Deref()

	defer generated.CessDestroySealPreCommitPhase2Response(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return cid.Undef, cid.Undef, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	commR, errCommrSize := commcid.ReplicaCommitmentV1ToCID(resp.CommR[:])
	if errCommrSize != nil {
		return cid.Undef, cid.Undef, errCommrSize
	}
	commD, errCommdSize := commcid.DataCommitmentV1ToCID(resp.CommD[:])
	if errCommdSize != nil {
		return cid.Undef, cid.Undef, errCommdSize
	}

	return commR, commD, nil
}

// SealCommitPhase1
func SealCommitPhase1(
	proofType abi.RegisteredSealProof,
	sealedCID cid.Cid,
	unsealedCID cid.Cid,
	cacheDirPath string,
	sealedSectorPath string,
	sectorNum abi.SectorNumber,
	minerID abi.ActorID,
	ticket abi.SealRandomness,
	seed abi.InteractiveSealRandomness,
	pieces []abi.PieceInfo,
) (phase1Output []byte, err error) {
	sp, err := toCessRegisteredSealProof(proofType)
	if err != nil {
		return nil, err
	}

	proverID, err := toProverID(minerID)
	if err != nil {
		return nil, err
	}

	commR, err := to32ByteCommR(sealedCID)
	if err != nil {
		return nil, err
	}

	commD, err := to32ByteCommD(unsealedCID)
	if err != nil {
		return nil, err
	}

	cessPublicPieceInfos, cessPublicPieceInfosLen, err := toCessPublicPieceInfos(pieces)
	if err != nil {
		return nil, err
	}

	resp := generated.CessSealCommitPhase1(sp, commR, commD, cacheDirPath, sealedSectorPath, uint64(sectorNum), proverID, to32ByteArray(ticket), to32ByteArray(seed), cessPublicPieceInfos, cessPublicPieceInfosLen)
	resp.Deref()

	defer generated.CessDestroySealCommitPhase1Response(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return nil, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	return copyBytes(resp.SealCommitPhase1OutputPtr, resp.SealCommitPhase1OutputLen), nil
}

// SealCommitPhase2
func SealCommitPhase2(
	phase1Output []byte,
	sectorNum abi.SectorNumber,
	minerID abi.ActorID,
) ([]byte, error) {
	proverID, err := toProverID(minerID)
	if err != nil {
		return nil, err
	}

	resp := generated.CessSealCommitPhase2(phase1Output, uint(len(phase1Output)), uint64(sectorNum), proverID)
	resp.Deref()

	defer generated.CessDestroySealCommitPhase2Response(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return nil, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	return copyBytes(resp.ProofPtr, resp.ProofLen), nil
}

// TODO AggregateSealProofs it only needs InteractiveRandomness out of the aggregateInfo.Infos
func AggregateSealProofs(aggregateInfo proof5.AggregateSealVerifyProofAndInfos, proofs [][]byte) (out []byte, err error) {
	sp, err := toCessRegisteredSealProof(aggregateInfo.SealProof)
	if err != nil {
		return nil, err
	}

	commRs := make([]generated.Cess32ByteArray, len(aggregateInfo.Infos))
	seeds := make([]generated.Cess32ByteArray, len(aggregateInfo.Infos))
	for i, info := range aggregateInfo.Infos {
		seeds[i] = to32ByteArray(info.InteractiveRandomness)
		commRs[i], err = to32ByteCommR(info.SealedCID)
		if err != nil {
			return nil, err
		}
	}

	pfs := make([]generated.CessSealCommitPhase2Response, len(proofs))
	for i := range proofs {
		pfs[i] = generated.CessSealCommitPhase2Response{
			ProofPtr: proofs[i],
			ProofLen: uint(len(proofs[i])),
		}
	}

	rap, err := toCessRegisteredAggregationProof(aggregateInfo.AggregateProof)
	if err != nil {
		return nil, err
	}

	resp := generated.CessAggregateSealProofs(sp, rap, commRs, uint(len(commRs)), seeds, uint(len(seeds)), pfs, uint(len(pfs)))
	resp.Deref()

	defer generated.CessDestroyAggregateProof(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return nil, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	return copyBytes(resp.ProofPtr, resp.ProofLen), nil
}

// Unseal
func Unseal(
	proofType abi.RegisteredSealProof,
	cacheDirPath string,
	sealedSector *os.File,
	unsealOutput *os.File,
	sectorNum abi.SectorNumber,
	minerID abi.ActorID,
	ticket abi.SealRandomness,
	unsealedCID cid.Cid,
) error {
	sectorSize, err := proofType.SectorSize()
	if err != nil {
		return err
	}

	unpaddedBytesAmount := abi.PaddedPieceSize(sectorSize).Unpadded()

	return UnsealRange(proofType, cacheDirPath, sealedSector, unsealOutput, sectorNum, minerID, ticket, unsealedCID, 0, uint64(unpaddedBytesAmount))
}

// UnsealRange
func UnsealRange(
	proofType abi.RegisteredSealProof,
	cacheDirPath string,
	sealedSector *os.File,
	unsealOutput *os.File,
	sectorNum abi.SectorNumber,
	minerID abi.ActorID,
	ticket abi.SealRandomness,
	unsealedCID cid.Cid,
	unpaddedByteIndex uint64,
	unpaddedBytesAmount uint64,
) error {
	sp, err := toCessRegisteredSealProof(proofType)
	if err != nil {
		return err
	}

	proverID, err := toProverID(minerID)
	if err != nil {
		return err
	}

	commD, err := to32ByteCommD(unsealedCID)
	if err != nil {
		return err
	}

	sealedSectorFd := sealedSector.Fd()
	defer runtime.KeepAlive(sealedSector)

	unsealOutputFd := unsealOutput.Fd()
	defer runtime.KeepAlive(unsealOutput)

	resp := generated.CessUnsealRange(sp, cacheDirPath, int32(sealedSectorFd), int32(unsealOutputFd), uint64(sectorNum), proverID, to32ByteArray(ticket), commD, unpaddedByteIndex, unpaddedBytesAmount)
	resp.Deref()

	defer generated.CessDestroyUnsealRangeResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	return nil
}

// GenerateWinningPoStSectorChallenge
func GenerateWinningPoStSectorChallenge(
	proofType abi.RegisteredPoStProof,
	minerID abi.ActorID,
	randomness abi.PoStRandomness,
	eligibleSectorsLen uint64,
) ([]uint64, error) {
	proverID, err := toProverID(minerID)
	if err != nil {
		return nil, err
	}

	pp, err := toCessRegisteredPoStProof(proofType)
	if err != nil {
		return nil, err
	}

	resp := generated.CessGenerateWinningPostSectorChallenge(
		pp, to32ByteArray(randomness),
		eligibleSectorsLen, proverID,
	)
	resp.Deref()
	resp.IdsPtr = make([]uint64, resp.IdsLen)
	resp.Deref()

	defer generated.CessDestroyGenerateWinningPostSectorChallenge(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return nil, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	// copy from C memory space to Go
	out := make([]uint64, resp.IdsLen)
	for idx := range out {
		out[idx] = resp.IdsPtr[idx]
	}

	return out, nil
}

// GenerateWinningPoSt
func GenerateWinningPoSt(
	minerID abi.ActorID,
	privateSectorInfo SortedPrivateSectorInfo,
	randomness abi.PoStRandomness,
) ([]proof5.PoStProof, error) {
	cessReplicas, cessReplicasLen, free, err := toCessPrivateReplicaInfos(privateSectorInfo.Values(), "winning")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create private replica info array for FFI")
	}
	defer free()

	proverID, err := toProverID(minerID)
	if err != nil {
		return nil, err
	}

	resp := generated.CessGenerateWinningPost(
		to32ByteArray(randomness),
		cessReplicas, cessReplicasLen,
		proverID,
	)
	resp.Deref()
	resp.ProofsPtr = make([]generated.CessPoStProof, resp.ProofsLen)
	resp.Deref()

	defer generated.CessDestroyGenerateWinningPostResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return nil, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	proofs, err := fromCessPoStProofs(resp.ProofsPtr)
	if err != nil {
		return nil, err
	}

	return proofs, nil
}

// GenerateWindowPoSt
func GenerateWindowPoSt(
	minerID abi.ActorID,
	privateSectorInfo SortedPrivateSectorInfo,
	randomness abi.PoStRandomness,
) ([]proof5.PoStProof, []abi.SectorNumber, error) {
	cessReplicas, cessReplicasLen, free, err := toCessPrivateReplicaInfos(privateSectorInfo.Values(), "window")
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create private replica info array for FFI")
	}
	defer free()

	proverID, err := toProverID(minerID)
	if err != nil {
		return nil, nil, err
	}

	resp := generated.CessGenerateWindowPost(to32ByteArray(randomness), cessReplicas, cessReplicasLen, proverID)
	resp.Deref()
	resp.ProofsPtr = make([]generated.CessPoStProof, resp.ProofsLen)
	resp.Deref()
	resp.FaultySectorsPtr = resp.FaultySectorsPtr[:resp.FaultySectorsLen]

	defer generated.CessDestroyGenerateWindowPostResponse(resp)

	faultySectors, err := fromCessPoStFaultySectors(resp.FaultySectorsPtr, resp.FaultySectorsLen)
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to parse faulty sectors list: %w", err)
	}

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return nil, faultySectors, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	proofs, err := fromCessPoStProofs(resp.ProofsPtr)
	if err != nil {
		return nil, nil, err
	}

	return proofs, faultySectors, nil
}

// GetGPUDevices produces a slice of strings, each representing the name of a
// detected GPU device.
func GetGPUDevices() ([]string, error) {
	resp := generated.CessGetGpuDevices()
	resp.Deref()
	resp.DevicesPtr = make([]string, resp.DevicesLen)
	resp.Deref()

	defer generated.CessDestroyGpuDeviceResponse(resp)

	out := make([]string, len(resp.DevicesPtr))
	for idx := range out {
		out[idx] = generated.RawString(resp.DevicesPtr[idx]).Copy()
	}

	return out, nil
}

// GetSealVersion
func GetSealVersion(proofType abi.RegisteredSealProof) (string, error) {
	sp, err := toCessRegisteredSealProof(proofType)
	if err != nil {
		return "", err
	}

	resp := generated.CessGetSealVersion(sp)
	resp.Deref()

	defer generated.CessDestroyStringResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return "", errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	return generated.RawString(resp.StringVal).Copy(), nil
}

// GetPoStVersion
func GetPoStVersion(proofType abi.RegisteredPoStProof) (string, error) {
	pp, err := toCessRegisteredPoStProof(proofType)
	if err != nil {
		return "", err
	}

	resp := generated.CessGetPostVersion(pp)
	resp.Deref()

	defer generated.CessDestroyStringResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return "", errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	return generated.RawString(resp.StringVal).Copy(), nil
}

func GetNumPartitionForFallbackPost(proofType abi.RegisteredPoStProof, numSectors uint) (uint, error) {
	pp, err := toCessRegisteredPoStProof(proofType)
	if err != nil {
		return 0, err
	}
	resp := generated.CessGetNumPartitionForFallbackPost(pp, numSectors)
	defer generated.CessDestroyGetNumPartitionForFallbackPostResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return 0, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	return resp.NumPartition, nil
}

// ClearCache
func ClearCache(sectorSize uint64, cacheDirPath string) error {
	resp := generated.CessClearCache(sectorSize, cacheDirPath)
	resp.Deref()

	defer generated.CessDestroyClearCacheResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	return nil
}

func FauxRep(proofType abi.RegisteredSealProof, cacheDirPath string, sealedSectorPath string) (cid.Cid, error) {
	sp, err := toCessRegisteredSealProof(proofType)
	if err != nil {
		return cid.Undef, err
	}

	resp := generated.CessFauxrep(sp, cacheDirPath, sealedSectorPath)
	resp.Deref()

	defer generated.CessDestroyFauxrepResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return cid.Undef, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	return commcid.ReplicaCommitmentV1ToCID(resp.Commitment[:])
}

func FauxRep2(proofType abi.RegisteredSealProof, cacheDirPath string, existingPAuxPath string) (cid.Cid, error) {
	sp, err := toCessRegisteredSealProof(proofType)
	if err != nil {
		return cid.Undef, err
	}

	resp := generated.CessFauxrep2(sp, cacheDirPath, existingPAuxPath)
	resp.Deref()

	defer generated.CessDestroyFauxrepResponse(resp)

	if resp.StatusCode != generated.FCPResponseStatusFCPNoError {
		return cid.Undef, errors.New(generated.RawString(resp.ErrorMsg).Copy())
	}

	return commcid.ReplicaCommitmentV1ToCID(resp.Commitment[:])
}

func toCessExistingPieceSizes(src []abi.UnpaddedPieceSize) ([]uint64, uint) {
	out := make([]uint64, len(src))

	for idx := range out {
		out[idx] = uint64(src[idx])
	}

	return out, uint(len(out))
}

func toCessPublicPieceInfos(src []abi.PieceInfo) ([]generated.CessPublicPieceInfo, uint, error) {
	out := make([]generated.CessPublicPieceInfo, len(src))

	for idx := range out {
		commP, err := to32ByteCommP(src[idx].PieceCID)
		if err != nil {
			return nil, 0, err
		}

		out[idx] = generated.CessPublicPieceInfo{
			NumBytes: uint64(src[idx].Size.Unpadded()),
			CommP:    commP.Inner,
		}
	}

	return out, uint(len(out)), nil
}

func toCessPublicReplicaInfos(src []proof5.SectorInfo, typ string) ([]generated.CessPublicReplicaInfo, uint, error) {
	out := make([]generated.CessPublicReplicaInfo, len(src))

	for idx := range out {
		commR, err := to32ByteCommR(src[idx].SealedCID)
		if err != nil {
			return nil, 0, err
		}

		out[idx] = generated.CessPublicReplicaInfo{
			CommR:    commR.Inner,
			SectorId: uint64(src[idx].SectorNumber),
		}

		switch typ {
		case "window":
			p, err := src[idx].SealProof.RegisteredWindowPoStProof()
			if err != nil {
				return nil, 0, err
			}

			out[idx].RegisteredProof, err = toCessRegisteredPoStProof(p)
			if err != nil {
				return nil, 0, err
			}
		case "winning":
			p, err := src[idx].SealProof.RegisteredWinningPoStProof()
			if err != nil {
				return nil, 0, err
			}

			out[idx].RegisteredProof, err = toCessRegisteredPoStProof(p)
			if err != nil {
				return nil, 0, err
			}
		}
	}

	return out, uint(len(out)), nil
}

func toCessPrivateReplicaInfo(src PrivateSectorInfo) (generated.CessPrivateReplicaInfo, func(), error) {
	commR, err := to32ByteCommR(src.SealedCID)
	if err != nil {
		return generated.CessPrivateReplicaInfo{}, func() {}, err
	}

	pp, err := toCessRegisteredPoStProof(src.PoStProofType)
	if err != nil {
		return generated.CessPrivateReplicaInfo{}, func() {}, err
	}

	out := generated.CessPrivateReplicaInfo{
		RegisteredProof: pp,
		CacheDirPath:    src.CacheDirPath,
		CommR:           commR.Inner,
		ReplicaPath:     src.SealedSectorPath,
		SectorId:        uint64(src.SectorNumber),
	}
	_, allocs := out.PassRef()
	return out, allocs.Free, nil
}

func toCessPrivateReplicaInfos(src []PrivateSectorInfo, typ string) ([]generated.CessPrivateReplicaInfo, uint, func(), error) {
	allocs := make([]AllocationManager, len(src))

	out := make([]generated.CessPrivateReplicaInfo, len(src))

	for idx := range out {
		commR, err := to32ByteCommR(src[idx].SealedCID)
		if err != nil {
			return nil, 0, func() {}, err
		}

		pp, err := toCessRegisteredPoStProof(src[idx].PoStProofType)
		if err != nil {
			return nil, 0, func() {}, err
		}

		out[idx] = generated.CessPrivateReplicaInfo{
			RegisteredProof: pp,
			CacheDirPath:    src[idx].CacheDirPath,
			CommR:           commR.Inner,
			ReplicaPath:     src[idx].SealedSectorPath,
			SectorId:        uint64(src[idx].SectorNumber),
		}

		_, allocs[idx] = out[idx].PassRef()
	}

	return out, uint(len(out)), func() {
		for idx := range allocs {
			allocs[idx].Free()
		}
	}, nil
}

func fromCessPoStFaultySectors(ptr []uint64, l uint) ([]abi.SectorNumber, error) {
	if l == 0 {
		return nil, nil
	}

	type sliceHeader struct {
		Data unsafe.Pointer
		Len  int
		Cap  int
	}

	(*sliceHeader)(unsafe.Pointer(&ptr)).Len = int(l) // don't worry about it

	snums := make([]abi.SectorNumber, 0, l)
	for i := uint(0); i < l; i++ {
		snums = append(snums, abi.SectorNumber(ptr[i]))
	}

	return snums, nil
}

func fromCessPoStProofs(src []generated.CessPoStProof) ([]proof5.PoStProof, error) {
	out := make([]proof5.PoStProof, len(src))

	for idx := range out {
		src[idx].Deref()

		pp, err := fromCessRegisteredPoStProof(src[idx].RegisteredProof)
		if err != nil {
			return nil, err
		}

		out[idx] = proof5.PoStProof{
			PoStProof:  pp,
			ProofBytes: copyBytes(src[idx].ProofPtr, src[idx].ProofLen),
		}
	}

	return out, nil
}

func toCessPoStProofs(src []proof5.PoStProof) ([]generated.CessPoStProof, uint, func(), error) {
	allocs := make([]AllocationManager, len(src))

	out := make([]generated.CessPoStProof, len(src))
	for idx := range out {
		pp, err := toCessRegisteredPoStProof(src[idx].PoStProof)
		if err != nil {
			return nil, 0, func() {}, err
		}

		out[idx] = generated.CessPoStProof{
			RegisteredProof: pp,
			ProofLen:        uint(len(src[idx].ProofBytes)),
			ProofPtr:        src[idx].ProofBytes,
		}

		_, allocs[idx] = out[idx].PassRef()
	}

	return out, uint(len(out)), func() {
		for idx := range allocs {
			allocs[idx].Free()
		}
	}, nil
}

func to32ByteArray(in []byte) generated.Cess32ByteArray {
	var out generated.Cess32ByteArray
	copy(out.Inner[:], in)
	return out
}

func toProverID(minerID abi.ActorID) (generated.Cess32ByteArray, error) {
	maddr, err := address.NewIDAddress(uint64(minerID))
	if err != nil {
		return generated.Cess32ByteArray{}, errors.Wrap(err, "failed to convert ActorID to prover id ([32]byte) for FFI")
	}

	return to32ByteArray(maddr.Payload()), nil
}

func fromCessRegisteredPoStProof(p generated.CessRegisteredPoStProof) (abi.RegisteredPoStProof, error) {
	switch p {
	case generated.CessRegisteredPoStProofStackedDrgWinning2KiBV1:
		return abi.RegisteredPoStProof_StackedDrgWinning2KiBV1, nil
	case generated.CessRegisteredPoStProofStackedDrgWinning8MiBV1:
		return abi.RegisteredPoStProof_StackedDrgWinning8MiBV1, nil
	case generated.CessRegisteredPoStProofStackedDrgWinning512MiBV1:
		return abi.RegisteredPoStProof_StackedDrgWinning512MiBV1, nil
	case generated.CessRegisteredPoStProofStackedDrgWinning32GiBV1:
		return abi.RegisteredPoStProof_StackedDrgWinning32GiBV1, nil
	case generated.CessRegisteredPoStProofStackedDrgWinning64GiBV1:
		return abi.RegisteredPoStProof_StackedDrgWinning64GiBV1, nil

	case generated.CessRegisteredPoStProofStackedDrgWindow2KiBV1:
		return abi.RegisteredPoStProof_StackedDrgWindow2KiBV1, nil
	case generated.CessRegisteredPoStProofStackedDrgWindow8MiBV1:
		return abi.RegisteredPoStProof_StackedDrgWindow8MiBV1, nil
	case generated.CessRegisteredPoStProofStackedDrgWindow512MiBV1:
		return abi.RegisteredPoStProof_StackedDrgWindow512MiBV1, nil
	case generated.CessRegisteredPoStProofStackedDrgWindow32GiBV1:
		return abi.RegisteredPoStProof_StackedDrgWindow32GiBV1, nil
	case generated.CessRegisteredPoStProofStackedDrgWindow64GiBV1:
		return abi.RegisteredPoStProof_StackedDrgWindow64GiBV1, nil
	default:
		return 0, errors.Errorf("no mapping to abi.RegisteredPoStProof value available for: %v", p)
	}
}

func toCessRegisteredPoStProof(p abi.RegisteredPoStProof) (generated.CessRegisteredPoStProof, error) {
	switch p {
	case abi.RegisteredPoStProof_StackedDrgWinning2KiBV1:
		return generated.CessRegisteredPoStProofStackedDrgWinning2KiBV1, nil
	case abi.RegisteredPoStProof_StackedDrgWinning8MiBV1:
		return generated.CessRegisteredPoStProofStackedDrgWinning8MiBV1, nil
	case abi.RegisteredPoStProof_StackedDrgWinning512MiBV1:
		return generated.CessRegisteredPoStProofStackedDrgWinning512MiBV1, nil
	case abi.RegisteredPoStProof_StackedDrgWinning32GiBV1:
		return generated.CessRegisteredPoStProofStackedDrgWinning32GiBV1, nil
	case abi.RegisteredPoStProof_StackedDrgWinning64GiBV1:
		return generated.CessRegisteredPoStProofStackedDrgWinning64GiBV1, nil

	case abi.RegisteredPoStProof_StackedDrgWindow2KiBV1:
		return generated.CessRegisteredPoStProofStackedDrgWindow2KiBV1, nil
	case abi.RegisteredPoStProof_StackedDrgWindow8MiBV1:
		return generated.CessRegisteredPoStProofStackedDrgWindow8MiBV1, nil
	case abi.RegisteredPoStProof_StackedDrgWindow512MiBV1:
		return generated.CessRegisteredPoStProofStackedDrgWindow512MiBV1, nil
	case abi.RegisteredPoStProof_StackedDrgWindow32GiBV1:
		return generated.CessRegisteredPoStProofStackedDrgWindow32GiBV1, nil
	case abi.RegisteredPoStProof_StackedDrgWindow64GiBV1:
		return generated.CessRegisteredPoStProofStackedDrgWindow64GiBV1, nil
	default:
		return 0, errors.Errorf("no mapping to abi.RegisteredPoStProof value available for: %v", p)
	}
}

func toCessRegisteredSealProof(p abi.RegisteredSealProof) (generated.CessRegisteredSealProof, error) {
	switch p {
	case abi.RegisteredSealProof_StackedDrg2KiBV1:
		return generated.CessRegisteredSealProofStackedDrg2KiBV1, nil
	case abi.RegisteredSealProof_StackedDrg8MiBV1:
		return generated.CessRegisteredSealProofStackedDrg8MiBV1, nil
	case abi.RegisteredSealProof_StackedDrg512MiBV1:
		return generated.CessRegisteredSealProofStackedDrg512MiBV1, nil
	case abi.RegisteredSealProof_StackedDrg32GiBV1:
		return generated.CessRegisteredSealProofStackedDrg32GiBV1, nil
	case abi.RegisteredSealProof_StackedDrg64GiBV1:
		return generated.CessRegisteredSealProofStackedDrg64GiBV1, nil

	case abi.RegisteredSealProof_StackedDrg2KiBV1_1:
		return generated.CessRegisteredSealProofStackedDrg2KiBV11, nil
	case abi.RegisteredSealProof_StackedDrg8MiBV1_1:
		return generated.CessRegisteredSealProofStackedDrg8MiBV11, nil
	case abi.RegisteredSealProof_StackedDrg512MiBV1_1:
		return generated.CessRegisteredSealProofStackedDrg512MiBV11, nil
	case abi.RegisteredSealProof_StackedDrg32GiBV1_1:
		return generated.CessRegisteredSealProofStackedDrg32GiBV11, nil
	case abi.RegisteredSealProof_StackedDrg64GiBV1_1:
		return generated.CessRegisteredSealProofStackedDrg64GiBV11, nil
	default:
		return 0, errors.Errorf("no mapping to C.FFIRegisteredSealProof value available for: %v", p)
	}
}

func toCessRegisteredAggregationProof(p abi.RegisteredAggregationProof) (generated.CessRegisteredAggregationProof, error) {
	switch p {
	case abi.RegisteredAggregationProof_SnarkPackV1:
		return generated.CessRegisteredAggregationProofSnarkPackV1, nil
	default:
		return 0, errors.Errorf("no mapping to abi.RegisteredAggregationProof value available for: %v", p)
	}
}

func to32ByteCommD(unsealedCID cid.Cid) (generated.Cess32ByteArray, error) {
	commD, err := commcid.CIDToDataCommitmentV1(unsealedCID)
	if err != nil {
		return generated.Cess32ByteArray{}, errors.Wrap(err, "failed to transform sealed CID to CommD")
	}

	return to32ByteArray(commD), nil
}

func to32ByteCommR(sealedCID cid.Cid) (generated.Cess32ByteArray, error) {
	commD, err := commcid.CIDToReplicaCommitmentV1(sealedCID)
	if err != nil {
		return generated.Cess32ByteArray{}, errors.Wrap(err, "failed to transform sealed CID to CommR")
	}

	return to32ByteArray(commD), nil
}

func to32ByteCommP(pieceCID cid.Cid) (generated.Cess32ByteArray, error) {
	commP, err := commcid.CIDToPieceCommitmentV1(pieceCID)
	if err != nil {
		return generated.Cess32ByteArray{}, errors.Wrap(err, "failed to transform sealed CID to CommP")
	}

	return to32ByteArray(commP), nil
}

func copyBytes(v []byte, vLen uint) []byte {
	buf := make([]byte, vLen)
	if n := copy(buf, v[:vLen]); n != int(vLen) {
		panic("partial read")
	}

	return buf
}

type stringHeader struct {
	Data unsafe.Pointer
	Len  int
}

func toVanillaProofs(src [][]byte) ([]generated.CessVanillaProof, func()) {
	allocs := make([]AllocationManager, len(src))

	out := make([]generated.CessVanillaProof, len(src))
	for idx := range out {
		out[idx] = generated.CessVanillaProof{
			ProofLen: uint(len(src[idx])),
			ProofPtr: src[idx],
		}

		_, allocs[idx] = out[idx].PassRef()
	}

	return out, func() {
		for idx := range allocs {
			allocs[idx].Free()
		}
	}
}

func toPartitionProofs(src []PartitionProof) ([]generated.CessPartitionSnarkProof, func(), error) {
	allocs := make([]AllocationManager, len(src))
	cleanup := func() {
		for idx := range allocs {
			allocs[idx].Free()
		}
	}

	out := make([]generated.CessPartitionSnarkProof, len(src))
	for idx := range out {
		rp, err := toCessRegisteredPoStProof(src[idx].PoStProof)
		if err != nil {
			return nil, cleanup, err
		}

		out[idx] = generated.CessPartitionSnarkProof{
			RegisteredProof: rp,
			ProofLen:        uint(len(src[idx].ProofBytes)),
			ProofPtr:        src[idx].ProofBytes,
		}

		_, allocs[idx] = out[idx].PassRef()
	}

	return out, cleanup, nil
}
