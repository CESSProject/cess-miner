/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/node/common"
	"github.com/CESSProject/cess-miner/pkg/com"
	"github.com/CESSProject/cess-miner/pkg/com/pb"
	"github.com/CESSProject/cess-miner/pkg/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

func (n *Node) idleChallenge(
	ch chan<- bool,
	idleProofSubmited bool,
	challStart uint32,
	minerChallFront int64,
	minerChallRear int64,
	spaceChallengeParam chain.SpaceChallengeParam,
	minerAccumulator chain.Accumulator,
	teeSign chain.TeeSig,
) {
	defer func() {
		ch <- true
		n.SetIdleChallenging(false)
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	var err error
	if idleProofSubmited {
		err = n.checkIdleProofRecord(challStart)
		if err == nil {
			return
		}
	}

	n.SetIdleChallenging(true)

	n.Ichal("info", fmt.Sprintf("Start counting idle challenges: %d", challStart))

	var idleProofRecord common.IdleProofInfo
	idleProofRecord.Start = challStart
	idleProofRecord.ChainFront = minerChallFront
	idleProofRecord.ChainRear = minerChallRear
	idleProofRecord.SubmintProof = true
	idleProofRecord.SubmintResult = true

	var acc = make([]byte, len(chain.Accumulator{}))
	for i := 0; i < len(acc); i++ {
		acc[i] = byte(minerAccumulator[i])
	}

	idleProofRecord.Acc = acc
	var minerPoisInfo = &pb.MinerPoisInfo{
		Acc:           acc,
		Front:         minerChallFront,
		Rear:          minerChallRear,
		KeyN:          n.KeyN,
		KeyG:          n.KeyG,
		StatusTeeSign: []byte(string(teeSign[:])),
	}

	if n.RSAKeyPair == nil || n.RSAKeyPair.Spk == nil {
		n.Ichal("err", "rsa public key is nil")
		return
	}

	err = n.Prover.SetChallengeState(*n.RsaKey, acc, minerChallFront, minerChallRear)
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[SetChallengeState] %v", err))
		return
	}

	var challRandom = make([]int64, chain.SpaceChallengeParamLen)
	for i := 0; i < chain.SpaceChallengeParamLen; i++ {
		challRandom[i] = int64(spaceChallengeParam[i])
	}

	idleProofRecord.ChallRandom = challRandom

	idleProofRecord.FileBlockProofInfo = make([]common.FileBlockProofInfo, 0)
	var idleproof = make([]byte, 0)

	teeID := make([]byte, 32)
	challengeHandle := n.Prover.NewChallengeHandle(teeID, challRandom)
	var previousHash []byte
	if minerChallFront == minerChallRear {
		for i := 0; i < 5; i++ {
			txhash, err := n.SubmitIdleProof([]types.U8{})
			if err != nil {
				n.Ichal("err", fmt.Sprintf("[SubmitIdleProof] %v", err))
				time.Sleep(time.Minute)
				continue
			}
			n.Ichal("info", fmt.Sprintf("SubmitIdleProof: %s", txhash))
			break
		}
		idleProofRecord.SubmintResult = false
		n.SaveIdleProve(idleProofRecord)
		return
	}
	n.Ichal("info", fmt.Sprintf("[start] %v", time.Now()))
	for {
		var fileBlockProofInfoEle common.FileBlockProofInfo
		left, right := challengeHandle(previousHash)
		if left == right {
			break
		}
		fileBlockProofInfoEle.FileBlockFront = left
		fileBlockProofInfoEle.FileBlockRear = right
		spaceProof, err := n.Prover.ProveSpace(challRandom, left, right)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("[ProveSpace] %v", err))
			return
		}
		n.Ichal("info", fmt.Sprintf("[end] %v", time.Now()))
		var mhtProofGroup = make([]*pb.MhtProofGroup, len(spaceProof.Proofs))

		for i := 0; i < len(spaceProof.Proofs); i++ {
			mhtProofGroup[i] = &pb.MhtProofGroup{}
			mhtProofGroup[i].Proofs = make([]*pb.MhtProof, len(spaceProof.Proofs[i]))
			for j := 0; j < len(spaceProof.Proofs[i]); j++ {
				mhtProofGroup[i].Proofs[j] = &pb.MhtProof{
					Index: int32(spaceProof.Proofs[i][j].Index),
					Label: spaceProof.Proofs[i][j].Label,
					Paths: spaceProof.Proofs[i][j].Paths,
					Locs:  spaceProof.Proofs[i][j].Locs,
				}
			}
		}

		var witChains = make([]*pb.AccWitnessNode, len(spaceProof.WitChains))

		for i := 0; i < len(spaceProof.WitChains); i++ {
			witChains[i] = &pb.AccWitnessNode{
				Elem: spaceProof.WitChains[i].Elem,
				Wit:  spaceProof.WitChains[i].Wit,
				Acc: &pb.AccWitnessNode{
					Elem: spaceProof.WitChains[i].Acc.Elem,
					Wit:  spaceProof.WitChains[i].Acc.Wit,
					Acc: &pb.AccWitnessNode{
						Elem: spaceProof.WitChains[i].Acc.Acc.Elem,
						Wit:  spaceProof.WitChains[i].Acc.Acc.Wit,
						Acc: &pb.AccWitnessNode{
							Elem: spaceProof.WitChains[i].Acc.Acc.Acc.Elem,
							Wit:  spaceProof.WitChains[i].Acc.Acc.Acc.Wit,
						},
					},
				},
			}
		}

		var proof = &pb.SpaceProof{
			Left:      spaceProof.Left,
			Right:     spaceProof.Right,
			Roots:     spaceProof.Roots,
			Proofs:    mhtProofGroup,
			WitChains: witChains,
		}

		fileBlockProofInfoEle.SpaceProof = proof

		b, err := proto.Marshal(proof)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("[proto.Marshal] %v", err))
			return
		}

		h := sha256.New()
		_, err = h.Write(b)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("[h.Write] %v", err))
			return
		}
		proofHash := h.Sum(nil)

		previousHash = proofHash

		fileBlockProofInfoEle.ProofHashSignOrigin = proofHash
		idleproof = append(idleproof, proofHash...)
		sign, err := n.Sign(proofHash)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("[n.Sign] %v", err))
			return
		}

		fileBlockProofInfoEle.ProofHashSign = sign
		idleProofRecord.FileBlockProofInfo = append(idleProofRecord.FileBlockProofInfo, fileBlockProofInfoEle)
	}
	h := sha256.New()
	_, err = h.Write(idleproof)
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[h.Write] %v", err))
		return
	}
	idleProof := h.Sum(nil)
	var idleProofChain = make([]types.U8, len(idleProof))
	for i := 0; i < len(idleProof); i++ {
		idleProofChain[i] = types.U8(idleProof[i])
	}

	err = n.submitIdleProof(idleProofChain)
	if err != nil {
		n.Ichal("err", err.Error())
		return
	}

	idleProofRecord.SubmintProof = false
	idleProofRecord.IdleProof = idleProofChain
	n.SaveIdleProve(idleProofRecord)

	teeSignBytes, pkchain, result, err := n.verifyIdleProof(minerChallFront, minerChallRear, minerPoisInfo, idleProofRecord, acc, challRandom)
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[verifyIdleProof] %v", err))
		return
	}

	for i := 0; i < 5; i++ {
		txHash, err := n.SubmitVerifyIdleResult(
			idleProofChain,
			types.U64(idleProofRecord.ChainFront),
			types.U64(idleProofRecord.ChainRear),
			minerAccumulator,
			types.Bool(result),
			teeSignBytes,
			pkchain,
		)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("[SubmitIdleProofResult] hash: %s, err: %v", txHash, err))
			time.Sleep(time.Minute)
			continue
		}
		n.Ichal("info", fmt.Sprintf("submit idle proof result suc: %s", txHash))
		break
	}
	idleProofRecord.SubmintResult = false
	n.SaveIdleProve(idleProofRecord)
}

func (n *Node) submitIdleProof(idleProof []types.U8) error {
	var (
		err       error
		blockHash string
		challInfo chain.ChallengeInfo
	)
	for i := 0; i < 5; i++ {
		n.Ichal("info", fmt.Sprintf("[start sub] %v", time.Now()))
		blockHash, err = n.SubmitIdleProof(idleProof)
		n.Ichal("info", fmt.Sprintf("[end sub]: %s", blockHash))
		if err == nil && blockHash != "" {
			return nil
		}

		time.Sleep(chain.BlockInterval * 3)

		_, challInfo, err = n.QueryChallengeSnapShot(n.GetSignatureAccPulickey(), -1)
		if err != nil {
			if err.Error() != chain.ERR_Empty {
				n.Ichal("err", fmt.Sprintf("[QueryChallengeInfo] %v", err))
			}
			return err
		}

		if challInfo.ProveInfo.IdleProve.HasValue() {
			_, idleProve := challInfo.ProveInfo.IdleProve.Unwrap()
			if len(idleProve.IdleProve) > 0 {
				return nil
			}
		}
	}
	return fmt.Errorf("submitIdleProof failed: %v", err)
}

func (n *Node) verifyIdleProof(
	minerChallFront int64,
	minerChallRear int64,
	minerPoisInfo *pb.MinerPoisInfo,
	idleProofRecord common.IdleProofInfo,
	acc []byte,
	challRandom []int64,
) (types.Bytes, chain.WorkerPublicKey, bool, error) {
	var blockProofs []*pb.BlocksProof
	var dialOptions []grpc.DialOption
	var teeSig chain.TeeSig
	var spaceProofVerifyTotal *pb.ResponseSpaceProofVerifyTotal
	var pkchain chain.WorkerPublicKey
	ctees := n.GetAllVerifierTeeEndpoint()
	requestSpaceProofVerify := &pb.RequestSpaceProofVerify{
		SpaceChals: idleProofRecord.ChallRandom,
		MinerId:    n.GetSignatureAccPulickey(),
		PoisInfo:   minerPoisInfo,
	}

	requestSpaceProofVerifyTotal := &pb.RequestSpaceProofVerifyTotal{
		MinerId:    n.GetSignatureAccPulickey(),
		Front:      minerChallFront,
		Rear:       minerChallRear,
		Acc:        acc,
		SpaceChals: challRandom,
	}

	for t := 0; t < len(ctees); t++ {
		n.Ichal("info", fmt.Sprintf("RequestSpaceProofVerifySingleBlock to tee: %s", ctees[t]))
		pk, err := n.TeeRecorder.GetTeePubkeyByEndpoint(ctees[t])
		if err != nil {
			n.Ichal("err", fmt.Sprintf("GetTeePubkeyByEndpoint err: %v", err))
			continue
		}
		pkchain, err = chain.BytesToWorkPublickey(pk)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("BytesToWorkPublickey err: %v", err))
			continue
		}
		if !strings.Contains(ctees[t], "443") {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		} else {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
		}

		blockProofs, err = verifyIdleSingleBlock(ctees[t], requestSpaceProofVerify, idleProofRecord.FileBlockProofInfo, dialOptions)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("verifyIdleSingleBlock err: %v", err))
			continue
		}
		idleProofRecord.BlocksProof = append(idleProofRecord.BlocksProof, blockProofs...)
		n.SaveIdleProve(idleProofRecord)

		n.Ichal("info", fmt.Sprintf("RequestVerifySpaceTotal to tee: %s", ctees[t]))

		requestSpaceProofVerifyTotal.ProofList = blockProofs
		spaceProofVerifyTotal, err = com.RequestVerifySpaceTotal(
			ctees[t],
			requestSpaceProofVerifyTotal,
			time.Duration(time.Minute*10),
			dialOptions,
			nil,
		)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("[RequestVerifySpaceTotal] %v", err))
			continue
		}

		if len(spaceProofVerifyTotal.Signature) != chain.TeeSigLen {
			n.Ichal("err", "invalid spaceProofVerifyTotal signature")
			continue
		}

		n.Ichal("info", fmt.Sprintf("SpaceProofVerifyTotal result: %v", spaceProofVerifyTotal.IdleResult))

		for i := 0; i < chain.TeeSigLen; i++ {
			teeSig[i] = types.U8(spaceProofVerifyTotal.Signature[i])
		}

		idleProofRecord.TotalSignature = spaceProofVerifyTotal.Signature
		idleProofRecord.IdleResult = spaceProofVerifyTotal.IdleResult
		idleProofRecord.AllocatedTeeWorkpuk = pkchain
		n.SaveIdleProve(idleProofRecord)
		var teeSignBytes = make(types.Bytes, len(teeSig))
		for j := 0; j < len(teeSig); j++ {
			teeSignBytes[j] = byte(teeSig[j])
		}
		return teeSignBytes, pkchain, spaceProofVerifyTotal.IdleResult, nil
	}
	return nil, chain.WorkerPublicKey{}, false, errors.New("verifyIdleProof failed")
}

func verifyIdleSingleBlock(teeEndpoint string, requestSpaceProofVerify *pb.RequestSpaceProofVerify, FileBlockProofInfo []common.FileBlockProofInfo, dialOptions []grpc.DialOption) ([]*pb.BlocksProof, error) {
	var err error
	var spaceProofVerify = &pb.ResponseSpaceProofVerify{}
	var blocksProof = make([]*pb.BlocksProof, len(FileBlockProofInfo))
	for i := 0; i < len(FileBlockProofInfo); i++ {
		requestSpaceProofVerify.Proof = FileBlockProofInfo[i].SpaceProof
		requestSpaceProofVerify.MinerSpaceProofHashPolkadotSig = FileBlockProofInfo[i].ProofHashSign
		spaceProofVerify, err = com.RequestSpaceProofVerifySingleBlock(
			teeEndpoint,
			requestSpaceProofVerify,
			time.Duration(time.Minute*10),
			dialOptions,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("[RequestSpaceProofVerifySingleBlock] %v", err)
		}
		blocksProof[i] = &pb.BlocksProof{
			ProofHashAndLeftRight: &pb.ProofHashAndLeftRight{
				SpaceProofHash: FileBlockProofInfo[i].ProofHashSignOrigin,
				Left:           FileBlockProofInfo[i].FileBlockFront,
				Right:          FileBlockProofInfo[i].FileBlockRear,
			},
			Signature: spaceProofVerify.Signature,
		}
	}
	return blocksProof, nil
}

func (n *Node) checkIdleProofRecord(challStart uint32) error {
	var err error
	var idleProofRecord common.IdleProofInfo

	idleProofRecord, err = n.LoadIdleProve()
	if err != nil {
		return err
	}

	if idleProofRecord.Start != challStart {
		os.Remove(n.GetIdleProve())
		n.Del("info", n.GetIdleProve())
		return errors.New("Local service file challenge record is outdated")
	}

	if !idleProofRecord.SubmintResult {
		return nil
	}

	if idleProofRecord.SubmintProof && idleProofRecord.TotalSignature != nil {
		var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
		for i := 0; i < len(idleProofRecord.IdleProof); i++ {
			idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
		}
		var teeSignBytes = make(types.Bytes, len(idleProofRecord.TotalSignature))
		for j := 0; j < len(idleProofRecord.TotalSignature); j++ {
			teeSignBytes[j] = byte(idleProofRecord.TotalSignature[j])
		}
		var minerAccumulator chain.Accumulator
		for i := 0; i < chain.AccumulatorLen; i++ {
			minerAccumulator[i] = types.U8(idleProofRecord.Acc[i])
		}
		for i := 0; i < 5; i++ {
			txHash, err := n.SubmitVerifyIdleResult(
				idleProve,
				types.U64(idleProofRecord.ChainFront),
				types.U64(idleProofRecord.ChainRear),
				minerAccumulator,
				types.Bool(idleProofRecord.IdleResult),
				teeSignBytes,
				idleProofRecord.AllocatedTeeWorkpuk,
			)
			if err != nil {
				n.Ichal("err", fmt.Sprintf("[SubmitIdleProofResult] hash: %s, err: %v", txHash, err))
				time.Sleep(time.Minute)
				continue
			}
			n.Ichal("info", fmt.Sprintf("submit idle proof result suc: %s", txHash))
			break
		}
		idleProofRecord.SubmintResult = false
		n.SaveIdleProve(idleProofRecord)
		return nil
	}

	return errors.New("Idle proof not submited")
}
