/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/pkg/com"
	"github.com/CESSProject/cess-miner/pkg/com/pb"
	"github.com/CESSProject/cess-miner/pkg/utils"
	"github.com/CESSProject/cess_pois/acc"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

func (n *Node) CertIdle(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()
	for {
		err := n.attestationidle()
		n.SetCertifyingIdle(false)
		if err != nil {
			n.Space("err", err.Error())
			time.Sleep(time.Minute)
		}
		time.Sleep(chain.BlockInterval)
	}
}

func (n *Node) attestationidle() error {
	for {
		for {
			if n.CommitDataIsReady() {
				break
			}
			time.Sleep(chain.BlockInterval)
		}
		n.SetCertifyingIdle(true)

		minerInfo, err := n.QueryMinerItems(n.GetSignatureAccPulickey(), -1)
		if err != nil {
			n.Space("err", fmt.Sprintf("[QueryStorageMiner] %v", err))
			time.Sleep(chain.BlockInterval)
			continue
		}
		var ok bool
		var spaceProofInfo chain.SpaceProofInfo
		if minerInfo.SpaceProofInfo.HasValue() {
			ok, spaceProofInfo = minerInfo.SpaceProofInfo.Unwrap()
			if !ok {
				return errors.New("minerInfo.SpaceProofInfo.Unwrap() failed")
			}
			if spaceProofInfo.Rear > types.U64(n.Prover.GetRear()) {
				err = n.Prover.SyncChainPoisStatus(int64(spaceProofInfo.Front), int64(spaceProofInfo.Rear))
				if err != nil {
					n.Space("err", fmt.Sprintf("[SyncChainPoisStatus] %v", err))
					return err
				}
			}
		}

		n.Space("info", "Get idle file commits")
		commits, err := n.Prover.GetIdleFileSetCommits()
		if err != nil {
			return errors.Wrapf(err, "[GetIdleFileSetCommits]")
		}

		n.Space("info", fmt.Sprintf("FileIndexs[0]: %v ", commits.FileIndexs[0]))
		var commit_pb = &pb.Commits{
			FileIndexs: commits.FileIndexs,
			Roots:      commits.Roots,
		}

		var chall_pb *pb.Challenge
		var commitGenChall = &pb.RequestMinerCommitGenChall{
			MinerId: n.GetSignatureAccPulickey(),
			Commit:  commit_pb,
		}

		buf, err := proto.Marshal(commitGenChall)
		if err != nil {
			n.Prover.CommitRollback()
			n.Space("err", fmt.Sprintf("[Marshal] %v", err))
			return err
		}

		commitGenChall.MinerSign, err = n.Sign(buf)
		if err != nil {
			n.Prover.CommitRollback()
			n.Space("err", fmt.Sprintf("[Sign] %v", err))
			return err
		}

		n.Space("info", fmt.Sprintf("front: %v rear: %v", n.Prover.GetFront(), n.Prover.GetRear()))
		var teeEndPoints = n.ReadPriorityTeeList()
		if len(teeEndPoints) > 0 {
			teeEndPoints = append(teeEndPoints, n.ReadPriorityTeeList()...)
			teeEndPoints = append(teeEndPoints, n.ReadPriorityTeeList()...)
		}
		teeEndPoints = append(teeEndPoints, n.GetAllMarkerTeeEndpoint()...)

		var usedTeeEndPoint string
		var usedTeeWorkAccount string
		var timeout time.Duration
		var timeoutStep = 0
		var dialOptions []grpc.DialOption
		for i := 0; i < len(teeEndPoints); i++ {
			timeout = time.Duration(time.Minute * 3)
			n.Space("info", fmt.Sprintf("Will use tee: %v", teeEndPoints[i]))
			if !strings.Contains(teeEndPoints[i], "443") {
				dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
			} else {
				dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
			}
			for try := 2; try <= 6; try += 2 {
				chall_pb, err = com.RequestMinerCommitGenChall(
					teeEndPoints[i],
					commitGenChall,
					time.Duration(timeout),
					dialOptions,
					nil,
				)
				if err != nil {
					n.Space("err", fmt.Sprintf("[RequestMinerCommitGenChall] %v", err))
					if strings.Contains(err.Error(), configs.Err_ctx_exceeded) {
						timeout = time.Duration(time.Minute * time.Duration(3+try))
						continue
					}
					break
				}
				usedTeeEndPoint = teeEndPoints[i]
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
			n.Prover.CommitRollback()
			return errors.New("no worked tee")
		}

		var chals = make([][]int64, len(chall_pb.Rows))
		for i := 0; i < len(chall_pb.Rows); i++ {
			chals[i] = chall_pb.Rows[i].Values
		}

		n.Space("info", fmt.Sprintf("Commit idle file commits to %s", usedTeeEndPoint))

		commitProofs, accProof, err := n.Prover.ProveCommitAndAcc(chals)
		if err != nil {
			n.Prover.AccRollback(false)
			return errors.Wrapf(err, "[ProveCommitAndAcc]")
		}

		if commitProofs == nil && accProof == nil {
			n.Prover.AccRollback(false)
			return errors.New("other programs are updating the data of the prover object")
		}

		if spaceProofInfo.Front == types.U64(n.Prover.GetFront()) {
			n.Front = int64(spaceProofInfo.Front)
			n.Rear = int64(spaceProofInfo.Rear)
			n.Acc = []byte(string(spaceProofInfo.Accumulator[:]))
			n.StatusTeeSign = []byte(string(minerInfo.TeeSig[:]))
		} else {
			minerInfo, err = n.QueryMinerItems(n.GetSignatureAccPulickey(), -1)
			if err != nil {
				n.Space("err", fmt.Sprintf("[QueryStorageMiner] %v", err))
				time.Sleep(chain.BlockInterval)
				return err
			}
			if minerInfo.SpaceProofInfo.HasValue() {
				ok, spaceProofInfo = minerInfo.SpaceProofInfo.Unwrap()
				if !ok {
					return errors.New("minerInfo.SpaceProofInfo.Unwrap() failed")
				}
				n.Front = int64(spaceProofInfo.Front)
				n.Rear = int64(spaceProofInfo.Rear)
				n.Acc = []byte(string(spaceProofInfo.Accumulator[:]))
				n.StatusTeeSign = []byte(string(minerInfo.TeeSig[:]))
			}
		}

		var commitProofGroupInner = make([]*pb.CommitProofGroupInner, len(commitProofs))
		for i := 0; i < len(commitProofs); i++ {
			commitProofGroupInner[i] = &pb.CommitProofGroupInner{}
			commitProofGroupInner[i].CommitProof = make([]*pb.CommitProof, len(commitProofs[i]))
			for j := 0; j < len(commitProofs[i]); j++ {
				commitProofGroupInner[i].CommitProof[j] = &pb.CommitProof{}
				commitProofGroupInner[i].CommitProof[j].Node = &pb.MhtProof{}
				commitProofGroupInner[i].CommitProof[j].Node.Index = int32(commitProofs[i][j].Node.Index)
				commitProofGroupInner[i].CommitProof[j].Node.Label = commitProofs[i][j].Node.Label
				commitProofGroupInner[i].CommitProof[j].Node.Locs = commitProofs[i][j].Node.Locs
				commitProofGroupInner[i].CommitProof[j].Node.Paths = commitProofs[i][j].Node.Paths
				commitProofGroupInner[i].CommitProof[j].Elders = make([]*pb.MhtProof, len(commitProofs[i][j].Elders))
				for k := 0; k < len(commitProofs[i][j].Elders); k++ {
					commitProofGroupInner[i].CommitProof[j].Elders[k] = &pb.MhtProof{}
					commitProofGroupInner[i].CommitProof[j].Elders[k].Index = int32(commitProofs[i][j].Elders[k].Index)
					commitProofGroupInner[i].CommitProof[j].Elders[k].Label = commitProofs[i][j].Elders[k].Label
					commitProofGroupInner[i].CommitProof[j].Elders[k].Locs = commitProofs[i][j].Elders[k].Locs
					commitProofGroupInner[i].CommitProof[j].Elders[k].Paths = commitProofs[i][j].Elders[k].Paths
				}
				commitProofGroupInner[i].CommitProof[j].Parents = make([]*pb.MhtProof, len(commitProofs[i][j].Parents))
				for k := 0; k < len(commitProofs[i][j].Parents); k++ {
					commitProofGroupInner[i].CommitProof[j].Parents[k] = &pb.MhtProof{}
					commitProofGroupInner[i].CommitProof[j].Parents[k].Index = int32(commitProofs[i][j].Parents[k].Index)
					commitProofGroupInner[i].CommitProof[j].Parents[k].Label = commitProofs[i][j].Parents[k].Label
					commitProofGroupInner[i].CommitProof[j].Parents[k].Locs = commitProofs[i][j].Parents[k].Locs
					commitProofGroupInner[i].CommitProof[j].Parents[k].Paths = commitProofs[i][j].Parents[k].Paths
				}
			}
		}
		var commitProofGroup_pb = &pb.CommitProofGroup{
			CommitProofGroupInner: commitProofGroupInner,
		}

		var accProof_pb = &pb.AccProof{
			Indexs:  accProof.Indexs,
			Labels:  accProof.Labels,
			AccPath: accProof.AccPath,
			WitChains: &pb.AccWitnessNode{
				Elem: accProof.WitChains.Elem,
				Wit:  accProof.WitChains.Wit,
				Acc: &pb.AccWitnessNode{
					Elem: accProof.WitChains.Acc.Elem,
					Wit:  accProof.WitChains.Acc.Wit,
					Acc: &pb.AccWitnessNode{
						Elem: accProof.WitChains.Acc.Acc.Elem,
						Wit:  accProof.WitChains.Acc.Acc.Wit,
					},
				},
			},
		}

		var requestVerifyCommitAndAccProof = &pb.RequestVerifyCommitAndAccProof{
			CommitProofGroup: commitProofGroup_pb,
			AccProof:         accProof_pb,
			MinerId:          n.GetSignatureAccPulickey(),
			PoisInfo:         n.MinerPoisInfo,
		}

		buf, err = proto.Marshal(requestVerifyCommitAndAccProof)
		if err != nil {
			n.Prover.AccRollback(false)
			n.Space("err", fmt.Sprintf("[Marshal-2] %v", err))
			return err
		}

		requestVerifyCommitAndAccProof.MinerSign, err = n.Sign(buf)
		if err != nil {
			n.Prover.AccRollback(false)
			n.Space("err", fmt.Sprintf("[Sign-2] %v", err))
			return err
		}

		n.Space("info", "Verify idle file commits")
		var tryCount uint8
		var verifyCommitOrDeletionProof *pb.ResponseVerifyCommitOrDeletionProof
		timeoutStep = 10
		for {
			timeout = time.Minute * time.Duration(timeoutStep)
			if tryCount >= 5 {
				n.Prover.AccRollback(false)
				return errors.Wrapf(err, "[RequestVerifyCommitProof]")
			}
			verifyCommitOrDeletionProof, err = com.RequestVerifyCommitProof(
				usedTeeEndPoint,
				requestVerifyCommitAndAccProof,
				time.Duration(timeout),
				dialOptions,
				nil,
			)
			if err != nil {
				if strings.Contains(err.Error(), "busy") {
					tryCount++
					time.Sleep(time.Minute)
					continue
				}
				if strings.Contains(err.Error(), configs.Err_ctx_exceeded) {
					timeoutStep += 3
					tryCount++
					time.Sleep(time.Minute)
					continue
				}
				n.Prover.AccRollback(false)
				return errors.Wrapf(err, "[RequestVerifyCommitProof]")
			}
			break
		}

		// If the challenge is failure, need to roll back the prover to the previous status,
		// this method will return whether the rollback is successful, and its parameter is also whether it is a delete operation be rolled back.

		if len(verifyCommitOrDeletionProof.StatusTeeSign) != chain.TeeSigLen {
			n.Prover.AccRollback(false)
			return errors.Wrapf(err, "[verifyCommitOrDeletionProof.Sign length err]")
		}

		var idleSignInfo chain.SpaceProofInfo
		var sign chain.TeeSig
		for i := 0; i < chain.TeeSigLen; i++ {
			sign[i] = types.U8(verifyCommitOrDeletionProof.StatusTeeSign[i])
		}
		var signWithAcc chain.TeeSig
		for i := 0; i < chain.TeeSigLen; i++ {
			signWithAcc[i] = types.U8(verifyCommitOrDeletionProof.SignatureWithTeeController[i])
		}
		if len(verifyCommitOrDeletionProof.PoisStatus.Acc) != len(chain.Accumulator{}) {
			n.Prover.AccRollback(false)
			return errors.Wrapf(err, "[verifyCommitOrDeletionProof.PoisStatus.Acc length err]")
		}
		for i := 0; i < len(verifyCommitOrDeletionProof.PoisStatus.Acc); i++ {
			idleSignInfo.Accumulator[i] = types.U8(verifyCommitOrDeletionProof.PoisStatus.Acc[i])
		}
		idleSignInfo.Front = types.U64(verifyCommitOrDeletionProof.PoisStatus.Front)
		idleSignInfo.Rear = types.U64(verifyCommitOrDeletionProof.PoisStatus.Rear)
		accountid, _ := types.NewAccountID(n.GetSignatureAccPulickey())
		idleSignInfo.Miner = *accountid
		g_byte := n.RsaKey.G.Bytes()
		n_byte := n.RsaKey.N.Bytes()
		for i := 0; i < len(g_byte); i++ {
			idleSignInfo.PoisKey.G[i] = types.U8(g_byte[i])
		}
		for i := 0; i < len(n_byte); i++ {
			idleSignInfo.PoisKey.N[i] = types.U8(n_byte[i])
		}

		n.Space("info", "Submit idle space")
		var wpuk chain.WorkerPublicKey
		for i := 0; i < chain.WorkerPublicKeyLen; i++ {
			wpuk[i] = types.U8(usedTeeWorkAccount[i])
		}
		var teeSignBytes = make(types.Bytes, len(sign))
		for j := 0; j < len(sign); j++ {
			teeSignBytes[j] = byte(sign[j])
		}
		var signWithAccBytes = make(types.Bytes, len(signWithAcc))
		for j := 0; j < len(sign); j++ {
			signWithAccBytes[j] = byte(signWithAcc[j])
		}
		txhash, err := n.CertIdleSpace(idleSignInfo, signWithAccBytes, teeSignBytes, wpuk)
		if err != nil || txhash == "" {
			n.Space("err", fmt.Sprintf("[%s] [CertIdleSpace]: %s", txhash, err))
			time.Sleep(chain.BlockInterval)
			time.Sleep(chain.BlockInterval)
			minerInfo, err := n.QueryMinerItems(n.GetSignatureAccPulickey(), -1)
			if err != nil {
				n.Prover.AccRollback(false)
				return fmt.Errorf("QueryStorageMiner err:[%v]", err)
			}
			if minerInfo.SpaceProofInfo.HasValue() {
				_, spaceProofInfo := minerInfo.SpaceProofInfo.Unwrap()
				if int64(spaceProofInfo.Rear) <= n.Prover.GetRear() {
					n.Prover.AccRollback(false)
					return fmt.Errorf("AccRollbak: [%v] < [%v]", int64(spaceProofInfo.Rear), n.Prover.GetRear())
				}
			}
		}

		if txhash != "" {
			n.Space("info", fmt.Sprintf("Certified space transactions: %s", txhash))
		}

		// If the challenge is successful, update the prover status, fileNum is challenged files number,
		// the second parameter represents whether it is a delete operation, and the commit proofs should belong to the joining files, so it is false
		err = n.Prover.UpdateStatus(acc.DEFAULT_ELEMS_NUM, false)
		if err != nil {
			return errors.Wrapf(err, "[UpdateStatus]")
		}

		n.Space("info", fmt.Sprintf("update status success, new acc value: %s", hex.EncodeToString(n.Prover.GetAccValue())))

		n.Front = verifyCommitOrDeletionProof.PoisStatus.Front
		n.Rear = verifyCommitOrDeletionProof.PoisStatus.Rear
		n.Acc = verifyCommitOrDeletionProof.PoisStatus.Acc
		n.StatusTeeSign = verifyCommitOrDeletionProof.StatusTeeSign
	}
}
