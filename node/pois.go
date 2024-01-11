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
	"github.com/CESSProject/p2p-go/out"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

type Pois struct {
	*pois.Prover
	*acc.RsaKey
	pattern.ExpendersInfo
	teePeerid string
	front     int64
	rear      int64
}

const poisSignalBlockNum = 1024

var minSpace = uint64(pois.FileSize * pattern.SIZE_1MiB * acc.DEFAULT_ELEMS_NUM * 2)

// poisMgt
func (n *Node) poisMgt(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()
	var err error
	var chainSt bool
	var minerSt string
	for {
		chainSt = n.GetChainState()
		if !chainSt {
			return
		}

		minerSt = n.GetMinerState()
		if minerSt != pattern.MINER_STATE_POSITIVE &&
			minerSt != pattern.MINER_STATE_FROZEN {
			return
		}
		err = n.certifiedSpace()
		if err != nil {
			n.Space("err", err.Error())
			time.Sleep(time.Minute)
		}
		time.Sleep(pattern.BlockInterval)
	}
}

func (n *Node) InitPois(firstflag bool, front, rear, freeSpace, count int64, key_n, key_g big.Int) error {
	var err error
	if n.Pois == nil {
		expendersInfo := n.ExpendersInfo
		n.Pois = &Pois{
			ExpendersInfo: expendersInfo,
		}
	}

	if len(key_n.Bytes()) != len(pattern.PoISKey_N{}) {
		return errors.New("invalid key_n length")
	}

	if len(key_g.Bytes()) != len(pattern.PoISKey_G{}) {
		return errors.New("invalid key_g length")
	}

	n.Pois.RsaKey = &acc.RsaKey{
		N: key_n,
		G: key_g,
	}
	n.Pois.front = front
	n.Pois.rear = rear
	cfg := pois.Config{
		AccPath:        n.DataDir.PoisDir,
		IdleFilePath:   n.DataDir.SpaceDir,
		ChallAccPath:   n.DataDir.AccDir,
		MaxProofThread: n.GetCpuCores(),
	}

	// k,n,d and key are params that needs to be negotiated with the verifier in advance.
	// minerID is storage node's account ID, and space is the amount of physical space available(MiB)
	n.Prover, err = pois.NewProver(
		int64(n.ExpendersInfo.K),
		int64(n.ExpendersInfo.N),
		int64(n.ExpendersInfo.D),
		n.GetSignatureAccPulickey(),
		freeSpace,
		count,
	)
	if err != nil {
		return err
	}

	if firstflag {
		//Please initialize prover for the first time
		err = n.Prover.Init(*n.Pois.RsaKey, cfg)
		if err != nil {
			return err
		}
	} else {
		// If it is downtime recovery, call the recovery method.front and rear are read from minner info on chain
		err = n.Prover.Recovery(*n.Pois.RsaKey, front, rear, cfg)
		if err != nil {
			if strings.Contains(err.Error(), "read element data") {
				err = n.Prover.CheckAndRestoreSubAccFiles(front, rear)
				if err != nil {
					return err
				}
				err = n.Prover.Recovery(*n.Pois.RsaKey, front, rear, cfg)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}
	n.Prover.AccManager.GetSnapshot()
	return nil
}

func (n *Node) genIdlefile(ch chan<- bool) {
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

	decSpace, validSpace, usedSpace, lockSpace := n.GetMinerSpaceInfo()
	if (validSpace + usedSpace + lockSpace) >= decSpace {
		n.Space("info", "The declared space has been authenticated")
		time.Sleep(time.Minute * 10)
		return
	}

	configSpace := n.GetUseSpace() * pattern.SIZE_1GiB
	if configSpace < minSpace {
		n.Space("err", "The configured space is less than the minimum space requirement")
		time.Sleep(time.Minute * 10)
		return
	}

	if (validSpace + usedSpace + lockSpace) > (configSpace - minSpace) {
		n.Space("info", "The space for authentication has reached the configured space size")
		time.Sleep(time.Hour)
		return
	}

	dirfreeSpace, err := utils.GetDirFreeSpace(n.Workspace())
	if err != nil {
		n.Space("err", fmt.Sprintf("[GetDirFreeSpace] %v", err))
		time.Sleep(time.Minute)
		return
	}

	if dirfreeSpace < minSpace {
		n.Space("err", fmt.Sprintf("The disk space is less than %dG", minSpace/pattern.SIZE_1GiB))
		time.Sleep(time.Minute * 10)
		return
	}

	n.Space("info", "Start generating idle files")
	n.SetGenIdleFlag(true)
	err = n.Prover.GenerateIdleFileSet()
	n.SetGenIdleFlag(false)
	if err != nil {
		if strings.Contains(err.Error(), "not enough space") {
			out.Err("Your workspace is out of capacity")
			n.Space("err", "workspace is out of capacity")
		} else {
			n.Space("err", fmt.Sprintf("[GenerateIdleFileSet] %v", err))
		}
		time.Sleep(time.Minute * 10)
		return
	}
	n.Space("info", "generate a idle file")
}

func (n *Node) certifiedSpace() error {
	defer func() {
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	for {
		for {
			if n.Prover.CommitDataIsReady() {
				break
			}
			n.SetAuthIdleFlag(false)
			time.Sleep(pattern.BlockInterval)
		}
		n.SetAuthIdleFlag(true)
		minerInfo, err := n.QueryStorageMiner(n.GetSignatureAccPulickey())
		if err != nil {
			n.Space("err", fmt.Sprintf("[QueryStorageMiner] %v", err))
			time.Sleep(pattern.BlockInterval)
			continue
		}
		var ok bool
		var spaceProofInfo pattern.SpaceProofInfo
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

		teeEndPoints := n.GetPriorityTeeList()
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
				chall_pb, err = n.RequestMinerCommitGenChall(
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

		if err == nil && commitProofs == nil && accProof == nil {
			n.Prover.AccRollback(false)
			return errors.New("other programs are updating the data of the prover object")
		}

		if spaceProofInfo.Front == types.U64(n.Prover.GetFront()) {
			n.MinerPoisInfo.Front = int64(spaceProofInfo.Front)
			n.MinerPoisInfo.Rear = int64(spaceProofInfo.Rear)
			n.MinerPoisInfo.Acc = []byte(string(spaceProofInfo.Accumulator[:]))
			n.MinerPoisInfo.StatusTeeSign = []byte(string(minerInfo.TeeSignature[:]))
		} else {
			minerInfo, err = n.QueryStorageMiner(n.GetSignatureAccPulickey())
			if err != nil {
				n.Space("err", fmt.Sprintf("[QueryStorageMiner] %v", err))
				time.Sleep(pattern.BlockInterval)
				return err
			}
			if minerInfo.SpaceProofInfo.HasValue() {
				ok, spaceProofInfo = minerInfo.SpaceProofInfo.Unwrap()
				if !ok {
					return errors.New("minerInfo.SpaceProofInfo.Unwrap() failed")
				}
				n.MinerPoisInfo.Front = int64(spaceProofInfo.Front)
				n.MinerPoisInfo.Rear = int64(spaceProofInfo.Rear)
				n.MinerPoisInfo.Acc = []byte(string(spaceProofInfo.Accumulator[:]))
				n.MinerPoisInfo.StatusTeeSign = []byte(string(minerInfo.TeeSignature[:]))
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
			verifyCommitOrDeletionProof, err = n.RequestVerifyCommitProof(
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

		if len(verifyCommitOrDeletionProof.StatusTeeSign) != pattern.TeeSignatureLen ||
			len(verifyCommitOrDeletionProof.SignatureWithTeeController) != len(pattern.TeeSignature{}) {
			n.Prover.AccRollback(false)
			return errors.Wrapf(err, "[verifyCommitOrDeletionProof.Sign length err]")
		}

		var idleSignInfo pattern.SpaceProofInfo
		var sign pattern.TeeSignature
		for i := 0; i < pattern.TeeSignatureLen; i++ {
			sign[i] = types.U8(verifyCommitOrDeletionProof.StatusTeeSign[i])
		}
		var signWithAcc pattern.TeeSignature
		for i := 0; i < pattern.TeeSignatureLen; i++ {
			signWithAcc[i] = types.U8(verifyCommitOrDeletionProof.SignatureWithTeeController[i])
		}
		if len(verifyCommitOrDeletionProof.PoisStatus.Acc) != len(pattern.Accumulator{}) {
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
		g_byte := n.Pois.RsaKey.G.Bytes()
		n_byte := n.Pois.RsaKey.N.Bytes()
		for i := 0; i < len(g_byte); i++ {
			idleSignInfo.PoisKey.G[i] = types.U8(g_byte[i])
		}
		for i := 0; i < len(n_byte); i++ {
			idleSignInfo.PoisKey.N[i] = types.U8(n_byte[i])
		}

		n.Space("info", "Submit idle space")
		txhash, err := n.CertIdleSpace(idleSignInfo, signWithAcc, sign, usedTeeWorkAccount)
		if err != nil || txhash == "" {
			n.Space("err", fmt.Sprintf("[%s] [CertIdleSpace]: %s", txhash, err))
			time.Sleep(pattern.BlockInterval)
			time.Sleep(pattern.BlockInterval)
			minerInfo, err := n.QueryStorageMiner(n.GetSignatureAccPulickey())
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
		n.Space("info", "update pois status")
		n.MinerPoisInfo.Front = verifyCommitOrDeletionProof.PoisStatus.Front
		n.MinerPoisInfo.Rear = verifyCommitOrDeletionProof.PoisStatus.Rear
		n.MinerPoisInfo.Acc = verifyCommitOrDeletionProof.PoisStatus.Acc
		n.MinerPoisInfo.StatusTeeSign = verifyCommitOrDeletionProof.StatusTeeSign
	}
}
