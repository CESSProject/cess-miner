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

	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/cess-go-sdk/core/sdk"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/pkg/logger"
	"github.com/CESSProject/cess-miner/pkg/utils"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

type fileBlockProofInfo struct {
	ProofHashSign       []byte         `json:"proofHashSign"`
	ProofHashSignOrigin []byte         `json:"proofHashSignOrigin"`
	SpaceProof          *pb.SpaceProof `json:"spaceProof"`
	FileBlockFront      int64          `json:"fileBlockFront"`
	FileBlockRear       int64          `json:"fileBlockRear"`
}

type idleProofInfo struct {
	Start               uint32                  `json:"start"`
	ChainFront          int64                   `json:"chainFront"`
	ChainRear           int64                   `json:"chainRear"`
	IdleResult          bool                    `json:"idleResult"`
	AllocatedTeeWorkpuk pattern.WorkerPublicKey `json:"allocatedTeeWorkpuk"`
	IdleProof           []byte                  `json:"idleProof"`
	Acc                 []byte                  `json:"acc"`
	TotalSignature      []byte                  `json:"totalSignature"`
	ChallRandom         []int64                 `json:"challRandom"`
	FileBlockProofInfo  []fileBlockProofInfo
	BlocksProof         []*pb.BlocksProof
}

func idleChallenge(
	n sdk.SDK,
	r *RunningState,
	l logger.Logger,
	m *pb.MinerPoisInfo,
	rsa *RSAKeyPair,
	p *Pois,
	teeRecord *TeeRecord,
	peernode *core.PeerNode,
	ws *Workspace,
	ch chan<- bool,
	idleProofSubmited bool,
	latestBlock uint32,
	challVerifyExpiration uint32,
	challStart uint32,
	minerChallFront int64,
	minerChallRear int64,
	spaceChallengeParam pattern.SpaceChallengeParam,
	minerAccumulator pattern.Accumulator,
	teeSign pattern.TeeSig,
	teePubkey pattern.WorkerPublicKey,
) {
	defer func() {
		ch <- true
		r.SetIdleChallengeFlag(false)
		if err := recover(); err != nil {
			l.Pnc(utils.RecoverError(err))
		}
	}()

	if challVerifyExpiration <= latestBlock {
		l.Ichal("err", fmt.Sprintf("%d < %d", challVerifyExpiration, latestBlock))
		return
	}

	err := checkIdleProofRecord(
		n, l, ws, teeRecord, peernode, m,
		idleProofSubmited,
		challStart,
		minerChallFront,
		minerChallRear,
		minerAccumulator,
		teeSign,
		teePubkey,
	)
	if err == nil {
		return
	}

	l.Ichal("info", fmt.Sprintf("Idle file chain challenge: %v", challStart))

	var idleProofRecord idleProofInfo
	idleProofRecord.Start = challStart
	idleProofRecord.ChainFront = minerChallFront
	idleProofRecord.ChainRear = minerChallRear

	var acc = make([]byte, len(pattern.Accumulator{}))
	for i := 0; i < len(acc); i++ {
		acc[i] = byte(minerAccumulator[i])
	}

	idleProofRecord.Acc = acc
	var minerPoisInfo = &pb.MinerPoisInfo{
		Acc:           acc,
		Front:         minerChallFront,
		Rear:          minerChallRear,
		KeyN:          m.KeyN,
		KeyG:          m.KeyG,
		StatusTeeSign: []byte(string(teeSign[:])),
	}

	if rsa == nil || rsa.Spk == nil {
		l.Ichal("err", "rsa public key is nil")
		return
	}

	err = p.Prover.SetChallengeState(*p.RsaKey, acc, minerChallFront, minerChallRear)
	if err != nil {
		l.Ichal("err", fmt.Sprintf("[SetChallengeState] %v", err))
		return
	}

	var challRandom = make([]int64, pattern.SpaceChallengeParamLen)
	for i := 0; i < pattern.SpaceChallengeParamLen; i++ {
		challRandom[i] = int64(spaceChallengeParam[i])
	}

	idleProofRecord.ChallRandom = challRandom
	var blocksProof = make([]*pb.BlocksProof, 0)
	var teeEndPoint string
	l.Ichal("info", "start calc challenge...")
	idleProofRecord.FileBlockProofInfo = make([]fileBlockProofInfo, 0)
	var idleproof = make([]byte, 0)
	var dialOptions []grpc.DialOption
	var timeout time.Duration
	var requestSpaceProofVerify *pb.RequestSpaceProofVerify
	var requestSpaceProofVerifyTotal *pb.RequestSpaceProofVerifyTotal
	var spaceProofVerify *pb.ResponseSpaceProofVerify
	var spaceProofVerifyTotal *pb.ResponseSpaceProofVerifyTotal

	teeID := make([]byte, 32)
	challengeHandle := p.Prover.NewChallengeHandle(teeID, challRandom)
	var previousHash []byte
	if minerChallFront != minerChallRear {
		for {
			var fileBlockProofInfoEle fileBlockProofInfo
			left, right := challengeHandle(previousHash)
			if left == right {
				break
			}
			fileBlockProofInfoEle.FileBlockFront = left
			fileBlockProofInfoEle.FileBlockRear = right
			spaceProof, err := p.Prover.ProveSpace(challRandom, left, right)
			if err != nil {
				l.Ichal("err", fmt.Sprintf("[ProveSpace] %v", err))
				return
			}
			l.Ichal("info", fmt.Sprintf("[end] %v", time.Now()))
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
				l.Ichal("err", fmt.Sprintf("[proto.Marshal] %v", err))
				return
			}

			h := sha256.New()
			_, err = h.Write(b)
			if err != nil {
				l.Ichal("err", fmt.Sprintf("[h.Write] %v", err))
				return
			}
			proofHash := h.Sum(nil)

			previousHash = proofHash

			fileBlockProofInfoEle.ProofHashSignOrigin = proofHash
			idleproof = append(idleproof, proofHash...)
			sign, err := n.Sign(proofHash)
			if err != nil {
				l.Ichal("err", fmt.Sprintf("[n.Sign] %v", err))
				return
			}

			fileBlockProofInfoEle.ProofHashSign = sign
			idleProofRecord.FileBlockProofInfo = append(idleProofRecord.FileBlockProofInfo, fileBlockProofInfoEle)
		}
		h := sha256.New()
		_, err = h.Write(idleproof)
		if err != nil {
			l.Ichal("err", fmt.Sprintf("[h.Write] %v", err))
			return
		}
		idleProofRecord.IdleProof = h.Sum(nil)
		var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
		for i := 0; i < len(idleProofRecord.IdleProof); i++ {
			idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
		}

		ws.SaveIdleProve(idleProofRecord)
		l.Ichal("info", fmt.Sprintf("[start sub] %v", time.Now()))
		txhash, err := n.SubmitIdleProof(idleProve)
		if err != nil {
			l.Ichal("err", fmt.Sprintf("[SubmitIdleProof] %v", err))
			return
		}
		l.Ichal("info", fmt.Sprintf("SubmitIdleProof: %s", txhash))
		//

		time.Sleep(pattern.BlockInterval * 2)

		_, chall, err := n.QueryChallengeInfo(n.GetSignatureAccPulickey(), -1)
		if err != nil {
			return
		}
		if chall.ProveInfo.IdleProve.HasValue() {
			_, iProve := chall.ProveInfo.IdleProve.Unwrap()
			idleProofRecord.AllocatedTeeWorkpuk = iProve.TeePubkey
		} else {
			return
		}

		teeInfoType, err := teeRecord.GetTee(string(idleProofRecord.AllocatedTeeWorkpuk[:]))
		if err != nil {
			teeInfo, err := n.QueryTeeWorker(idleProofRecord.AllocatedTeeWorkpuk)
			if err != nil {
				l.Ichal("err", err.Error())
				return
			}
			endpoint, err := n.QueryTeeWorkEndpoint(teeInfo.Pubkey)
			if err != nil {
				l.Ichal("err", err.Error())
				return
			}
			teeRecord.SaveTee(string(idleProofRecord.AllocatedTeeWorkpuk[:]), endpoint, uint8(teeInfo.Role))
			teeEndPoint = endpoint
			if utils.ContainsIpv4(teeEndPoint) {
				teeEndPoint = strings.TrimPrefix(teeEndPoint, "http://")
			} else {
				teeEndPoint = strings.TrimSuffix(teeEndPoint, "/")
				teeEndPoint = teeEndPoint + ":443"
			}
		} else {
			teeEndPoint = teeInfoType.EndPoint
		}

		if !strings.Contains(teeEndPoint, "443") {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		} else {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
		}

		l.Ichal("info", fmt.Sprintf("RequestSpaceProofVerifySingleBlock to tee: %s", teeEndPoint))
		requestSpaceProofVerify = &pb.RequestSpaceProofVerify{
			SpaceChals: idleProofRecord.ChallRandom,
			MinerId:    n.GetSignatureAccPulickey(),
			PoisInfo:   minerPoisInfo,
		}
		for i := 0; i < len(idleProofRecord.FileBlockProofInfo); i++ {
			timeout = time.Minute * 10
			requestSpaceProofVerify.Proof = idleProofRecord.FileBlockProofInfo[i].SpaceProof
			requestSpaceProofVerify.MinerSpaceProofHashPolkadotSig = idleProofRecord.FileBlockProofInfo[i].ProofHashSign
			for try := 10; try <= 30; try += 10 {
				spaceProofVerify, err = peernode.RequestSpaceProofVerifySingleBlock(
					teeEndPoint,
					requestSpaceProofVerify,
					time.Duration(timeout),
					dialOptions,
					nil,
				)
				if err != nil {
					l.Ichal("err", fmt.Sprintf("[RequestSpaceProofVerifySingleBlock] %v", err))
					time.Sleep(time.Minute)
					if strings.Contains(err.Error(), configs.Err_ctx_exceeded) {
						timeout = time.Minute * time.Duration(10+try)
						continue
					}
					return
				}
				break
			}
			var block = &pb.BlocksProof{
				ProofHashAndLeftRight: &pb.ProofHashAndLeftRight{
					SpaceProofHash: idleProofRecord.FileBlockProofInfo[i].ProofHashSignOrigin,
					Left:           idleProofRecord.FileBlockProofInfo[i].FileBlockFront,
					Right:          idleProofRecord.FileBlockProofInfo[i].FileBlockRear,
				},
				Signature: spaceProofVerify.Signature,
			}
			blocksProof = append(blocksProof, block)
		}

		idleProofRecord.BlocksProof = blocksProof
		ws.SaveIdleProve(idleProofRecord)
		requestSpaceProofVerifyTotal = &pb.RequestSpaceProofVerifyTotal{
			MinerId:    n.GetSignatureAccPulickey(),
			ProofList:  blocksProof,
			Front:      minerChallFront,
			Rear:       minerChallRear,
			Acc:        acc,
			SpaceChals: challRandom,
		}
		l.Ichal("info", fmt.Sprintf("RequestVerifySpaceTotal to tee: %s", teeEndPoint))
		timeout = time.Minute * 3
		for try := 3; try <= 9; try += 3 {
			spaceProofVerifyTotal, err = peernode.RequestVerifySpaceTotal(
				teeEndPoint,
				requestSpaceProofVerifyTotal,
				time.Duration(timeout),
				dialOptions,
				nil,
			)
			if err != nil {
				l.Ichal("err", fmt.Sprintf("[RequestVerifySpaceTotal] %v", err))
				if strings.Contains(err.Error(), configs.Err_ctx_exceeded) {
					timeout = time.Minute * time.Duration(3+try)
					continue
				}
				return
			}
			break
		}
		l.Ichal("info", fmt.Sprintf("spaceProofVerifyTotal.IdleResult is %v", spaceProofVerifyTotal.IdleResult))

		var teeSig pattern.TeeSig
		if len(spaceProofVerifyTotal.Signature) != pattern.TeeSigLen {
			l.Ichal("err", "invalid spaceProofVerifyTotal signature")
			return
		}

		for i := 0; i < pattern.TeeSigLen; i++ {
			teeSig[i] = types.U8(spaceProofVerifyTotal.Signature[i])
		}

		idleProofRecord.TotalSignature = spaceProofVerifyTotal.Signature
		idleProofRecord.IdleResult = spaceProofVerifyTotal.IdleResult
		ws.SaveIdleProve(idleProofRecord)
		var teeSignBytes = make(types.Bytes, len(teeSig))
		for j := 0; j < len(teeSig); j++ {
			teeSignBytes[j] = byte(teeSig[j])
		}
		txHash, err := n.SubmitIdleProofResult(
			idleProve,
			types.U64(idleProofRecord.ChainFront),
			types.U64(idleProofRecord.ChainRear),
			minerAccumulator,
			types.Bool(spaceProofVerifyTotal.IdleResult),
			teeSignBytes,
			idleProofRecord.AllocatedTeeWorkpuk,
		)
		if err != nil {
			l.Ichal("err", fmt.Sprintf("[SubmitIdleProofResult] hash: %s, err: %v", txHash, err))
			return
		}

		l.Ichal("info", fmt.Sprintf("submit idle proof result suc: %s", txHash))
	} else {
		txhash, err := n.SubmitIdleProof([]types.U8{})
		if err != nil {
			l.Ichal("err", fmt.Sprintf("[SubmitIdleProof] %v", err))
			return
		}
		l.Ichal("info", fmt.Sprintf("SubmitIdleProof: %s", txhash))
	}
}

func checkIdleProofRecord(
	cli sdk.SDK,
	l logger.Logger,
	ws *Workspace,
	teeRecord *TeeRecord,
	peernode *core.PeerNode,
	m *pb.MinerPoisInfo,
	idleProofSubmited bool,
	challStart uint32,
	minerChallFront int64,
	minerChallRear int64,
	minerAccumulator pattern.Accumulator,
	teeSign pattern.TeeSig,
	teePubkey pattern.WorkerPublicKey,
) error {
	var err error
	var timeout time.Duration
	var teeEndPoint string
	var idleProofRecord idleProofInfo
	var dialOptions []grpc.DialOption
	var requestSpaceProofVerify *pb.RequestSpaceProofVerify
	var requestSpaceProofVerifyTotal *pb.RequestSpaceProofVerifyTotal
	var spaceProofVerifyTotal *pb.ResponseSpaceProofVerifyTotal
	idleProofRecord, err = ws.LoadIdleProve()
	if err != nil {
		return err
	}
	if idleProofRecord.Start != challStart {
		os.Remove(ws.GetIdleProve())
		l.Del("info", ws.GetIdleProve())
		return errors.New("Local service file challenge record is outdated")
	}

	l.Ichal("info", fmt.Sprintf("local idle proof file challenge: %v", idleProofRecord.Start))
	if !idleProofSubmited {
		return errors.New("Idle proof not submited")
	}

	if sutils.IsWorkerPublicKeyAllZero(teePubkey) {
		_, chall, err := cli.QueryChallengeInfo(cli.GetSignatureAccPulickey(), -1)
		if err != nil {
			return err
		}
		if chall.ProveInfo.IdleProve.HasValue() {
			_, iProve := chall.ProveInfo.IdleProve.Unwrap()
			idleProofRecord.AllocatedTeeWorkpuk = iProve.TeePubkey
		} else {
			return errors.New("The chain has not yet allocated a tee to verify the idle proof.")
		}
	} else {
		idleProofRecord.AllocatedTeeWorkpuk = teePubkey
	}

	var acc = make([]byte, len(pattern.Accumulator{}))
	for i := 0; i < len(acc); i++ {
		acc[i] = byte(minerAccumulator[i])
	}

	var minerPoisInfo = &pb.MinerPoisInfo{
		Acc:           acc,
		Front:         minerChallFront,
		Rear:          minerChallRear,
		KeyN:          m.KeyN,
		KeyG:          m.KeyG,
		StatusTeeSign: []byte(string(teeSign[:])),
	}

	for {
		if idleProofRecord.TotalSignature != nil {
			var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
			for i := 0; i < len(idleProofRecord.IdleProof); i++ {
				idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
			}
			var teeSig pattern.TeeSig
			if len(idleProofRecord.TotalSignature) != pattern.TeeSigLen {
				l.Ichal("err", "invalid spaceProofVerifyTotal signature")
				break
			}
			for i := 0; i < pattern.TeeSigLen; i++ {
				teeSig[i] = types.U8(idleProofRecord.TotalSignature[i])
			}
			var teeSignBytes = make(types.Bytes, len(teeSig))
			for j := 0; j < len(teeSig); j++ {
				teeSignBytes[j] = byte(teeSig[j])
			}
			txHash, err := cli.SubmitIdleProofResult(
				idleProve,
				types.U64(minerChallFront),
				types.U64(minerChallRear),
				minerAccumulator,
				types.Bool(idleProofRecord.IdleResult),
				teeSignBytes,
				idleProofRecord.AllocatedTeeWorkpuk,
			)
			if err != nil {
				l.Ichal("err", fmt.Sprintf("[SubmitIdleProofResult] hash: %s, err: %v", txHash, err))
				time.Sleep(time.Minute)
				break
			}
			l.Ichal("info", fmt.Sprintf("submit idle proof result suc: %s", txHash))
			return nil
		}
		break
	}

	teeInfoType, err := teeRecord.GetTee(string(idleProofRecord.AllocatedTeeWorkpuk[:]))
	if err != nil {
		teeInfo, err := cli.QueryTeeWorker(idleProofRecord.AllocatedTeeWorkpuk)
		if err != nil {
			l.Ichal("err", err.Error())
			return err
		}
		endpoint, err := cli.QueryTeeWorkEndpoint(idleProofRecord.AllocatedTeeWorkpuk)
		if err != nil {
			l.Ichal("err", err.Error())
			return err
		}
		teeRecord.SaveTee(string(idleProofRecord.AllocatedTeeWorkpuk[:]), endpoint, uint8(teeInfo.Role))
		teeEndPoint = endpoint
		if utils.ContainsIpv4(teeEndPoint) {
			teeEndPoint = strings.TrimPrefix(teeEndPoint, "http://")
		} else {
			teeEndPoint = strings.TrimSuffix(teeEndPoint, "/")
			teeEndPoint = teeEndPoint + ":443"
		}
	} else {
		teeEndPoint = teeInfoType.EndPoint
	}

	l.Ichal("info", fmt.Sprintf("Allocated tee: %v", teeEndPoint))
	requestSpaceProofVerifyTotal = &pb.RequestSpaceProofVerifyTotal{
		MinerId:    cli.GetSignatureAccPulickey(),
		ProofList:  idleProofRecord.BlocksProof,
		Front:      minerChallFront,
		Rear:       minerChallRear,
		Acc:        acc,
		SpaceChals: idleProofRecord.ChallRandom,
	}
	if !strings.Contains(teeEndPoint, "443") {
		dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	} else {
		dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
	}
	for {
		if idleProofRecord.BlocksProof != nil {
			timeout = time.Minute * 10
			l.Ichal("info", fmt.Sprintf("RequestVerifySpaceTotal to tee: %s", teeEndPoint))
			for try := 10; try < 30; try += 10 {
				spaceProofVerifyTotal, err = peernode.RequestVerifySpaceTotal(
					teeEndPoint,
					requestSpaceProofVerifyTotal,
					time.Duration(timeout),
					dialOptions,
					nil,
				)
				if err != nil {
					l.Ichal("err", fmt.Sprintf("[RequestVerifySpaceTotal] %v", err))
					if strings.Contains(err.Error(), configs.Err_ctx_exceeded) {
						timeout = time.Minute * time.Duration(10+try)
					}
					time.Sleep(time.Minute * 2)
					continue
				}
				break
			}
			if err != nil || spaceProofVerifyTotal == nil {
				break
			}
			idleProofRecord.TotalSignature = spaceProofVerifyTotal.Signature
			idleProofRecord.IdleResult = spaceProofVerifyTotal.IdleResult

			var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
			for i := 0; i < len(idleProofRecord.IdleProof); i++ {
				idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
			}
			var teeSig pattern.TeeSig
			if len(idleProofRecord.TotalSignature) != pattern.TeeSigLen {
				l.Ichal("err", "invalid spaceProofVerifyTotal signature")
				break
			}

			for i := 0; i < pattern.TeeSigLen; i++ {
				teeSig[i] = types.U8(idleProofRecord.TotalSignature[i])
			}
			ws.SaveIdleProve(idleProofRecord)
			var teeSignBytes = make(types.Bytes, len(teeSig))
			for j := 0; j < len(teeSig); j++ {
				teeSignBytes[j] = byte(teeSig[j])
			}
			txHash, err := cli.SubmitIdleProofResult(
				idleProve,
				types.U64(minerChallFront),
				types.U64(minerChallRear),
				minerAccumulator,
				types.Bool(idleProofRecord.IdleResult),
				teeSignBytes,
				idleProofRecord.AllocatedTeeWorkpuk,
			)
			if err != nil {
				l.Ichal("err", fmt.Sprintf("[SubmitIdleProofResult] hash: %s, err: %v", txHash, err))
				time.Sleep(time.Minute)
				break
			}
			l.Ichal("info", fmt.Sprintf("SubmitIdleProofResult: %s", txHash))
			return nil
		}
		break
	}

	var blocksProof = make([]*pb.BlocksProof, 0)
	requestSpaceProofVerify = &pb.RequestSpaceProofVerify{
		SpaceChals: idleProofRecord.ChallRandom,
		MinerId:    cli.GetSignatureAccPulickey(),
		PoisInfo:   minerPoisInfo,
	}
	l.Ichal("info", fmt.Sprintf("RequestSpaceProofVerifySingleBlock to tee: %s", teeEndPoint))
	for i := 0; i < len(idleProofRecord.FileBlockProofInfo); i++ {
		requestSpaceProofVerify.Proof = idleProofRecord.FileBlockProofInfo[i].SpaceProof
		requestSpaceProofVerify.MinerSpaceProofHashPolkadotSig = idleProofRecord.FileBlockProofInfo[i].ProofHashSign
		timeout = time.Minute * 10
		for try := 10; try <= 30; try += 10 {
			spaceProofVerify, err := peernode.RequestSpaceProofVerifySingleBlock(
				teeEndPoint,
				requestSpaceProofVerify,
				time.Duration(timeout),
				dialOptions,
				nil,
			)
			if err != nil {
				l.Ichal("err", fmt.Sprintf("[RequestSpaceProofVerifySingleBlock] %v", err))
				time.Sleep(time.Minute)
				if strings.Contains(err.Error(), configs.Err_ctx_exceeded) {
					timeout = time.Minute * time.Duration(10+try)
					continue
				}
				return nil
			}
			var block = &pb.BlocksProof{
				ProofHashAndLeftRight: &pb.ProofHashAndLeftRight{
					SpaceProofHash: idleProofRecord.FileBlockProofInfo[i].ProofHashSignOrigin,
					Left:           idleProofRecord.FileBlockProofInfo[i].FileBlockFront,
					Right:          idleProofRecord.FileBlockProofInfo[i].FileBlockRear,
				},
				Signature: spaceProofVerify.Signature,
			}
			blocksProof = append(blocksProof, block)
			break
		}
	}

	idleProofRecord.BlocksProof = blocksProof
	ws.SaveIdleProve(idleProofRecord)
	requestSpaceProofVerifyTotal = &pb.RequestSpaceProofVerifyTotal{
		MinerId:    cli.GetSignatureAccPulickey(),
		ProofList:  blocksProof,
		Front:      minerChallFront,
		Rear:       minerChallRear,
		Acc:        acc,
		SpaceChals: idleProofRecord.ChallRandom,
	}
	timeout = time.Minute * 10
	l.Ichal("info", fmt.Sprintf("RequestVerifySpaceTotal to tee: %s", teeEndPoint))
	for try := 10; try < 30; try += 10 {
		spaceProofVerifyTotal, err = peernode.RequestVerifySpaceTotal(
			teeEndPoint,
			requestSpaceProofVerifyTotal,
			time.Duration(timeout),
			dialOptions,
			nil,
		)
		if err != nil {
			l.Ichal("err", fmt.Sprintf("[RequestVerifySpaceTotal] %v", err))
			if strings.Contains(err.Error(), configs.Err_ctx_exceeded) {
				timeout = time.Minute * time.Duration(10+try)
				continue
			}
			return nil
		}
		break
	}

	var teeSig pattern.TeeSig
	if len(spaceProofVerifyTotal.Signature) != pattern.TeeSigLen {
		l.Ichal("err", "invalid spaceProofVerifyTotal signature")
		return nil
	}

	for i := 0; i < pattern.TeeSigLen; i++ {
		teeSig[i] = types.U8(spaceProofVerifyTotal.Signature[i])
	}

	var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
	for i := 0; i < len(idleProofRecord.IdleProof); i++ {
		idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
	}
	idleProofRecord.TotalSignature = spaceProofVerifyTotal.Signature
	idleProofRecord.IdleResult = spaceProofVerifyTotal.IdleResult
	ws.SaveIdleProve(idleProofRecord)
	var teeSignBytes = make(types.Bytes, len(teeSig))
	for j := 0; j < len(teeSig); j++ {
		teeSignBytes[j] = byte(teeSig[j])
	}
	var txHash string
	for j := 2; j < 10; j++ {
		txHash, err = cli.SubmitIdleProofResult(
			idleProve,
			types.U64(minerChallFront),
			types.U64(minerChallRear),
			minerAccumulator,
			types.Bool(spaceProofVerifyTotal.IdleResult),
			teeSignBytes,
			idleProofRecord.AllocatedTeeWorkpuk,
		)
		if err != nil {
			l.Ichal("err", fmt.Sprintf("[SubmitIdleProofResult] hash: %s, err: %v", txHash, err))
			time.Sleep(time.Minute * time.Duration(j))
			continue
		}
		l.Ichal("info", fmt.Sprintf("submit idle proof result suc: %s", txHash))
		return nil
	}
	return nil
}
