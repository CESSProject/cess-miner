/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/cess_pois/acc"
	"github.com/CESSProject/cess_pois/pois"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

func (n *Node) replaceIdle(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	chainSt := n.GetChainState()
	if !chainSt {
		return
	}

	minerSt := n.GetMinerState()
	if minerSt != pattern.MINER_STATE_POSITIVE &&
		minerSt != pattern.MINER_STATE_FROZEN {
		return
	}

	replaceSize, err := n.QueryPendingReplacements(n.GetSignatureAccPulickey())
	if err != nil {
		if err.Error() != pattern.ERR_Empty {
			n.Replace("err", err.Error())
		}
		return
	}

	if replaceSize.CmpAbs(big.NewInt(0)) <= 0 {
		return
	}

	if !replaceSize.IsUint64() {
		n.Replace("err", "replaceSize is not uint64")
		return
	}

	n.Replace("info", fmt.Sprintf("replace size: %v", replaceSize.Uint64()))
	num := uint64(replaceSize.Uint64() / 1024 / 1024 / uint64(pois.FileSize))
	if num == 0 {
		n.Replace("info", "no files to replace")
		return
	}

	if int64(num) > int64((int64(acc.DEFAULT_ELEMS_NUM) - n.GetFront()%int64(acc.DEFAULT_ELEMS_NUM))) {
		num = uint64((int64(acc.DEFAULT_ELEMS_NUM) - n.GetFront()%int64(acc.DEFAULT_ELEMS_NUM)))
	}

	n.Replace("info", fmt.Sprintf("Will replace %d idle files", num))

	delProof, err := n.Prover.ProveDeletion(int64(num))
	if err != nil {
		n.Replace("err", err.Error())
		return
	}

	if delProof == nil {
		n.Replace("err", "delProof is nil")
		return
	}

	if delProof.Roots == nil || delProof.AccPath == nil || delProof.WitChain == nil {
		n.Replace("err", "delProof have nil field")
		return
	}

	minerInfo, err := n.QueryStorageMiner(n.GetSignatureAccPulickey())
	if err != nil {
		n.Replace("err", fmt.Sprintf("[QueryStorageMiner] %v", err))
		return
	}
	if minerInfo.SpaceProofInfo.HasValue() {
		_, spaceProofInfo := minerInfo.SpaceProofInfo.Unwrap()
		if spaceProofInfo.Front > types.U64(n.Prover.GetFront()) {
			err = n.Prover.SyncChainPoisStatus(int64(spaceProofInfo.Front), int64(spaceProofInfo.Rear))
			if err != nil {
				return
			}
		}
		n.MinerPoisInfo.Front = int64(spaceProofInfo.Front)
		n.MinerPoisInfo.Rear = int64(spaceProofInfo.Rear)
		n.MinerPoisInfo.Acc = []byte(string(spaceProofInfo.Accumulator[:]))
		n.MinerPoisInfo.StatusTeeSign = []byte(string(minerInfo.TeeSignature[:]))
	}

	var witChain = &pb.AccWitnessNode{
		Elem: delProof.WitChain.Elem,
		Wit:  delProof.WitChain.Wit,
		Acc: &pb.AccWitnessNode{
			Elem: delProof.WitChain.Acc.Elem,
			Wit:  delProof.WitChain.Acc.Wit,
			Acc: &pb.AccWitnessNode{
				Elem: delProof.WitChain.Acc.Acc.Elem,
				Wit:  delProof.WitChain.Acc.Acc.Wit,
			},
		},
	}
	var requestVerifyDeletionProof = &pb.RequestVerifyDeletionProof{
		Roots:    delProof.Roots,
		WitChain: witChain,
		AccPath:  delProof.AccPath,
		MinerId:  n.GetSignatureAccPulickey(),
		PoisInfo: n.MinerPoisInfo,
	}
	buf, err := proto.Marshal(requestVerifyDeletionProof)
	if err != nil {
		n.Prover.CommitRollback()
		n.Replace("err", fmt.Sprintf("[Marshal-2] %v", err))
		return
	}
	signData, err := n.Sign(buf)
	if err != nil {
		n.Prover.CommitRollback()
		n.Replace("err", fmt.Sprintf("[Sign-2] %v", err))
		return
	}
	requestVerifyDeletionProof.MinerSign = signData
	var verifyCommitOrDeletionProof *pb.ResponseVerifyCommitOrDeletionProof
	var usedTeeEndPoint string
	var usedTeeWorkAccount string
	var timeout time.Duration
	var timeoutStep time.Duration = 3
	var dialOptions []grpc.DialOption
	teeEndPoints := n.GetPriorityTeeList()
	teeEndPoints = append(teeEndPoints, n.GetAllMarkerTeeEndpoint()...)
	for _, t := range teeEndPoints {
		timeout = time.Duration(time.Minute * timeoutStep)
		n.Space("info", fmt.Sprintf("Will use tee: %v", t))
		if !strings.Contains(t, "443") {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		} else {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
		}
		for try := 2; try <= 6; try += 2 {
			verifyCommitOrDeletionProof, err = n.RequestVerifyDeletionProof(
				t,
				requestVerifyDeletionProof,
				time.Duration(timeout),
				dialOptions,
				nil,
			)
			if err != nil {
				if strings.Contains(err.Error(), configs.Err_ctx_exceeded) {
					timeoutStep += 2
					time.Sleep(time.Minute)
					continue
				}
				n.Replace("err", fmt.Sprintf("[RequestVerifyDeletionProof] %v", err))
				break
			}
			usedTeeEndPoint = t
			usedTeeWorkAccount, err = n.GetTeeWorkAccount(usedTeeEndPoint)
			if err != nil {
				n.Space("err", fmt.Sprintf("[GetTeeWorkAccount(%s)] %v", usedTeeEndPoint, err))
			}
			break
		}
		if usedTeeEndPoint != "" && usedTeeWorkAccount != "" {
			break
		}
	}

	if usedTeeEndPoint == "" || usedTeeWorkAccount == "" {
		n.AccRollback(true)
		n.Replace("err", "No available tee")
		return
	}

	var idleSignInfo pattern.SpaceProofInfo
	minerAcc, _ := types.NewAccountID(n.GetSignatureAccPulickey())
	idleSignInfo.Miner = *minerAcc
	idleSignInfo.Front = types.U64(verifyCommitOrDeletionProof.PoisStatus.Front)
	idleSignInfo.Rear = types.U64(verifyCommitOrDeletionProof.PoisStatus.Rear)

	if len(verifyCommitOrDeletionProof.StatusTeeSign) != pattern.TeeSignatureLen ||
		len(verifyCommitOrDeletionProof.SignatureWithTeeController) != pattern.TeeSignatureLen {
		n.AccRollback(true)
		n.Replace("err", "invalid tee sign length")
		return
	}

	for i := 0; i < len(verifyCommitOrDeletionProof.PoisStatus.Acc); i++ {
		idleSignInfo.Accumulator[i] = types.U8(verifyCommitOrDeletionProof.PoisStatus.Acc[i])
	}
	g_byte := n.Pois.RsaKey.G.Bytes()
	n_byte := n.Pois.RsaKey.N.Bytes()
	for i := 0; i < len(g_byte); i++ {
		idleSignInfo.PoisKey.G[i] = types.U8(g_byte[i])
	}
	for i := 0; i < len(n_byte); i++ {
		idleSignInfo.PoisKey.N[i] = types.U8(n_byte[i])
	}

	var sign pattern.TeeSignature
	for i := 0; i < pattern.TeeSignatureLen; i++ {
		sign[i] = types.U8(verifyCommitOrDeletionProof.StatusTeeSign[i])
	}
	var signWithAcc pattern.TeeSignature
	for i := 0; i < pattern.TeeSignatureLen; i++ {
		signWithAcc[i] = types.U8(verifyCommitOrDeletionProof.SignatureWithTeeController[i])
	}

	//
	txhash, err := n.ReplaceIdleSpace(idleSignInfo, signWithAcc, sign, usedTeeWorkAccount)
	if err != nil || txhash == "" {
		n.AccRollback(true)
		n.Replace("err", err.Error())
		return
	}

	n.Replace("info", fmt.Sprintf("Replace files suc: %v", txhash))

	err = n.Prover.UpdateStatus(int64(num), true)
	if err != nil {
		n.Replace("err", err.Error())
	}

	ok, challenge, err := n.QueryChallengeInfo(n.GetSignatureAccPulickey())
	if err != nil {
		if err.Error() != pattern.ERR_Empty {
			n.Replace("err", err.Error())
			return
		}
	}

	if ok {
		err = n.Prover.SetChallengeState(*n.Pois.RsaKey, []byte(string(challenge.MinerSnapshot.SpaceProofInfo.Accumulator[:])), int64(challenge.MinerSnapshot.SpaceProofInfo.Front), int64(challenge.MinerSnapshot.SpaceProofInfo.Rear))
		if err != nil {
			return
		}
	}

	err = n.Prover.DeleteFiles()
	if err != nil {
		n.Replace("err", err.Error())
	}
	n.Replace("info", fmt.Sprintf("Successfully replaced %d idle files", num))
}
