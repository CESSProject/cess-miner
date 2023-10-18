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

	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/cess_pois/acc"
	"github.com/CESSProject/cess_pois/pois"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
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

	for {
		err := n.pois()
		if err != nil {
			n.Space("err", err.Error())
		}
		time.Sleep(pattern.BlockInterval)
	}
}

func (n *Node) InitPois(front, rear, freeSpace, count int64, key_n, key_g big.Int) error {
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
		MaxProofThread: n.GetCpuCore(),
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

	if n.front == 0 && n.rear == 0 {
		//Please initialize prover for the first time
		err = n.Prover.Init(*n.Pois.RsaKey, cfg)
		if err != nil {
			return err
		}
	} else {
		// If it is downtime recovery, call the recovery method.front and rear are read from minner info on chain
		err = n.Prover.Recovery(*n.Pois.RsaKey, front, rear, cfg)
		if err != nil {
			return err
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
	err = n.Prover.GenerateIdleFileSet()
	if err != nil {
		n.Space("err", fmt.Sprintf("[GenerateIdleFileSet] %v", err))
		return
	}
	n.Space("info", "generate a idle file")
}

func (n *Node) pois() error {
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
			time.Sleep(time.Minute)
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
		var workTeePeerID peer.ID
		var commitGenChall = &pb.RequestMinerCommitGenChall{
			MinerId:  n.GetSignatureAccPulickey(),
			Commit:   commit_pb,
			PoisInfo: n.MinerPoisInfo,
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
		n.Space("info", fmt.Sprintf("len(Acc): %v, acc: %v", len(n.MinerPoisInfo.Acc), n.MinerPoisInfo.Acc))
		teePeerIds := n.GetAllTeeWorkPeerIdString()
		n.Space("info", fmt.Sprintf("All tees: %v", teePeerIds))
		for i := 0; i < len(teePeerIds); i++ {
			if teePeerIds[i] != "12D3KooWAdyc4qPWFHsxMtXvSrm7CXNFhUmKPQdoXuKQXki69qBo" {
				continue
			}
			n.Space("info", fmt.Sprintf("Will use tee: %v", teePeerIds[i]))
			addrInfo, ok := n.GetPeer(teePeerIds[i])
			if !ok {
				n.Space("err", fmt.Sprintf("Not found tee: %s", teePeerIds[i]))
				continue
			}
			err = n.Connect(n.GetCtxQueryFromCtxCancel(), addrInfo)
			if err != nil {
				n.Space("err", fmt.Sprintf("Connect %s err: %v", teePeerIds[i], err))
				continue
			}
			chall_pb, err = n.PoisMinerCommitGenChallP2P(addrInfo.ID, commitGenChall, time.Duration(time.Minute*5))
			if err != nil {
				n.Space("err", fmt.Sprintf("[PoisMinerCommitGenChallP2P] %v", err))
				continue
			}
			workTeePeerID = addrInfo.ID
			break
		}

		if workTeePeerID.Pretty() == "" {
			n.Prover.CommitRollback()
			return errors.New("no worked tee")
		}

		var chals = make([][]int64, len(chall_pb.Rows))
		for i := 0; i < len(chall_pb.Rows); i++ {
			chals[i] = chall_pb.Rows[i].Values
		}

		n.Space("info", fmt.Sprintf("Commit idle file commits to %s", workTeePeerID.Pretty()))

		commitProofs, accProof, err := n.Prover.ProveCommitAndAcc(chals)
		if err != nil {
			n.Prover.AccRollback(false)
			return errors.Wrapf(err, "[ProveCommitAndAcc]")
		}

		if err == nil && commitProofs == nil && accProof == nil {
			n.Prover.AccRollback(false)
			return errors.New("other programs are updating the data of the prover object")
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

		for {
			if tryCount >= 100 {
				n.Prover.AccRollback(false)
				return errors.Wrapf(err, "[PoisVerifyCommitProofP2P]")
			}
			verifyCommitOrDeletionProof, err = n.PoisVerifyCommitProofP2P(
				workTeePeerID,
				requestVerifyCommitAndAccProof,
				time.Duration(time.Minute*10),
			)
			if err != nil {
				if strings.Contains(err.Error(), "busy") {
					tryCount++
					time.Sleep(time.Minute)
					continue
				}
				n.Prover.AccRollback(false)
				return errors.Wrapf(err, "[PoisVerifyCommitProofP2P]")
			}
			break
		}

		n.MinerPoisInfo.StatusTeeSign = verifyCommitOrDeletionProof.SignatureAbove

		// If the challenge is failure, need to roll back the prover to the previous status,
		// this method will return whether the rollback is successful, and its parameter is also whether it is a delete operation be rolled back.

		if len(verifyCommitOrDeletionProof.SignatureAbove) != len(pattern.TeeSignature{}) {
			n.Prover.AccRollback(false)
			return errors.Wrapf(err, "[verifyCommitOrDeletionProof.SignatureAbove length err]")
		}

		var idleSignInfo pattern.SpaceProofInfo
		var sign pattern.TeeSignature
		for i := 0; i < len(verifyCommitOrDeletionProof.SignatureAbove); i++ {
			sign[i] = types.U8(verifyCommitOrDeletionProof.SignatureAbove[i])
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
		txhash, err := n.CertIdleSpace(idleSignInfo, sign)
		if err != nil {
			n.Prover.AccRollback(false)
			return errors.Wrapf(err, "[CertIdleSpace]")
		}

		n.Space("info", fmt.Sprintf("Certified space transactions: %s", txhash))

		// If the challenge is successful, update the prover status, fileNum is challenged files number,
		// the second parameter represents whether it is a delete operation, and the commit proofs should belong to the joining files, so it is false
		err = n.Prover.UpdateStatus(acc.DEFAULT_ELEMS_NUM, false)
		if err != nil {
			return errors.Wrapf(err, "[UpdateStatus]")
		}

		n.MinerPoisInfo.Front = n.Prover.GetFront()
		n.MinerPoisInfo.Rear = n.Prover.GetRear()
		n.MinerPoisInfo.Acc = n.Prover.GetAccValue()

		n.Space("info", "update pois status")
	}
}
