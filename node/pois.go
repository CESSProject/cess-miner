/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"math/big"
	"os"
	"runtime"
	"time"

	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/CESSProject/cess_pois/acc"
	"github.com/CESSProject/cess_pois/pois"
	"github.com/CESSProject/p2p-go/out"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

type Pois struct {
	*pois.Prover
	*acc.RsaKey
	pattern.ExpendersInfo
	teePeerid string
	front     int64
	rear      int64
}

// spaceMgt is a subtask for managing spaces
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
			out.Err(err.Error())
		}
		time.Sleep(time.Second * 20)
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
		AccPath:        n.GetDirs().ProofDir,
		IdleFilePath:   n.GetDirs().IdleDataDir,
		MaxProofThread: runtime.NumCPU(),
	}

	// k,n,d and key are params that needs to be negotiated with the verifier in advance.
	// minerID is storage node's account ID, and space is the amount of physical space available(MiB)
	n.Prover, err = pois.NewProver(
		int64(n.ExpendersInfo.K),
		int64(n.ExpendersInfo.N),
		int64(n.ExpendersInfo.D),
		n.GetSignatureAccPulickey(),
		1024*1024*4, //freeSpace,
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

func (n *Node) pois() error {
	fmt.Println("start calc idle file...")
	// Generate Idle Files
	err := n.Prover.GenerateIdleFileSet()
	if err != nil {
		return errors.Wrapf(err, "[GenerateIdleFileSet]")
	}
	fmt.Println("calc idle file suc")

	fmt.Println("start get idle file commit...")
	//n.Prover.CommitDataIsReady()
	commits, err := n.Prover.GetIdleFileSetCommits()
	if err != nil {
		n.Prover.CommitRollback()
		return errors.Wrapf(err, "[GetIdleFileSetCommits]")
	}
	fmt.Println("get idle file commit suc")

	//fmt.Println("commit.FileIndexs:", commits.FileIndexs)
	//fmt.Println("commit.Roots:", commits.Roots)
	// TODO: send commits to tee and receive chall
	var commit_pb = &pb.Commits{
		FileIndexs: commits.FileIndexs,
		Roots:      commits.Roots,
	}

	fmt.Println("start send commit to tee...")
	chall_pb, err := n.PoisMinerCommitGenChall(os.Args[2], n.GetSignatureAccPulickey(), commit_pb, time.Duration(time.Second*10))
	if err != nil {
		n.Prover.CommitRollback()
		out.Err(fmt.Sprintf("[PoisMinerCommitGenChall] %v", err))
		return errors.Wrapf(err, "[PoisMinerCommitGenChall]")
	}
	fmt.Println("send commit to tee suc...")

	var chals = make([][]int64, len(chall_pb.Rows))
	for i := 0; i < len(chall_pb.Rows); i++ {
		chals[i] = chall_pb.Rows[i].Values
	}

	verifier := pois.NewVerifier(8, 16*1024, 64)
	verifier.RegisterProverNode(n.Prover.ID, *n.RsaKey, n.AccManager.GetSnapshot().Accs.Value, n.front, n.rear)
	if ok := verifier.ReceiveCommits(n.Prover.ID, commits); !ok {
		fmt.Println("self reveive commits error")
	}
	//fmt.Println("chals:", chals)
	fmt.Println("start calc tee's challenge...")
	commitProofs, accProof, err := n.Prover.ProveCommitAndAcc(chals)
	if err != nil {
		n.Prover.AccRollback(false)
		return errors.Wrapf(err, "[ProveCommitAndAcc]")
	}
	if err == nil && commitProofs == nil && accProof == nil {
		n.Prover.AccRollback(false)
		// If the results are all nil, it means that other programs are updating the data of the prover object.
		return errors.New("other programs are updating the data of the prover object")
	}

	fmt.Println("start self verify......")
	if err := verifier.VerifyCommitProofs(n.Prover.ID, chals, commitProofs); err != nil {
		fmt.Println("verify commit proofs error", err)
	}

	if err := verifier.VerifyAcc(n.Prover.ID, chals, accProof); err != nil {
		fmt.Println("verify acc proofs error", err)
	}

	fmt.Println("calc tee's challenge suc")

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

	//fmt.Println("accProof.AccPath:", accProof.AccPath)
	//fmt.Println("accProof.WitChains:", *accProof.WitChains)

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
	// TODO: send commitProofs and accProof to verifier and then wait for the response
	fmt.Println("start verify commit and acc proof...")
	verifyCommitOrDeletionProof, err := n.PoisVerifyCommitProof(os.Args[2], n.GetSignatureAccPulickey(), commitProofGroup_pb, accProof_pb, n.Pois.RsaKey.N.Bytes(), n.Pois.RsaKey.G.Bytes(), time.Duration(time.Minute*5))
	if err != nil {
		n.Prover.AccRollback(false)
		return errors.Wrapf(err, "[PoisVerifyCommitProof]")
	}
	fmt.Println("verify commit and acc proof suc")

	// If the challenge is failure, need to roll back the prover to the previous status,
	// this method will return whether the rollback is successful, and its parameter is also whether it is a delete operation be rolled back.

	if len(verifyCommitOrDeletionProof.SignatureAbove) != len(pattern.TeeSignature{}) {
		n.Prover.AccRollback(false)
		out.Err(fmt.Sprintf("[verifyCommitOrDeletionProof.SignatureAbove length err] %v", len(verifyCommitOrDeletionProof.SignatureAbove)))
		return errors.Wrapf(err, "[verifyCommitOrDeletionProof.SignatureAbove length err]")
	}

	//
	var idleSignInfo pattern.SpaceProofInfo
	var sign pattern.TeeSignature
	for i := 0; i < len(verifyCommitOrDeletionProof.SignatureAbove); i++ {
		sign[i] = types.U8(verifyCommitOrDeletionProof.SignatureAbove[i])
	}
	if len(verifyCommitOrDeletionProof.PoisStatus.Acc) != len(pattern.Accumulator{}) {
		n.Prover.AccRollback(false)
		out.Err(fmt.Sprintf("[verifyCommitOrDeletionProof.PoisStatus.Acc length err] %v", len(verifyCommitOrDeletionProof.PoisStatus.Acc)))
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

	fmt.Println("start CertIdleSpace...")
	txhash, err := n.CertIdleSpace(idleSignInfo, sign)
	if err != nil {
		n.Prover.AccRollback(false)
		return errors.Wrapf(err, "[CertIdleSpace]")
	}
	fmt.Println("CertIdleSpace suc...")
	fmt.Println("CertIdleSpace txhash:", txhash)

	// If the challenge is successful, update the prover status, fileNum is challenged files number,
	// the second parameter represents whether it is a delete operation, and the commit proofs should belong to the joining files, so it is false
	fmt.Println("start UpdateStatus...")
	err = n.Prover.UpdateStatus(256, false)
	if err != nil {
		return errors.Wrapf(err, "[UpdateStatus]")
	}
	fmt.Println("UpdateStatus suc")

	// challenge, err := n.QueryChallenge_V2()
	// if err != nil {
	// 	if err.Error() != pattern.ERR_Empty {
	// 		return errors.Wrapf(err, "[QueryChallenge]")
	// 	}
	// 	n.Prover.SetChallengeState(false)
	// 	return nil
	// }

	// for _, v := range challenge.MinerSnapshotList {
	// 	if sutils.CompareSlice(n.GetSignatureAccPulickey(), v.Miner[:]) {
	// 		if int64(v.SpaceProofInfo.Front) != n.Prover.GetFront() || int64(v.SpaceProofInfo.Rear) != n.Prover.GetRear() {
	// 			n.Prover.SetChallengeState(true)
	// 		} else {
	// 			n.Prover.SetChallengeState(false)
	// 		}
	// 		return nil
	// 	}
	// }
	// n.Prover.SetChallengeState(false)

	return nil
}

func (n *Node) replaceIdle() {
	replaceSize, err := n.QueryPendingReplacements_V2(n.GetStakingPublickey())
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

	num := uint64(replaceSize.Uint64() / 1024 / 1024 / uint64(pois.FileSize))
	n.Replace("info", fmt.Sprintf("Will replace %d idle files", num))

	chProof, ch := n.Prover.ProveDeletion(int64(num))
	var delProof *pois.DeletionProof
	var timeOut = time.NewTicker(time.Minute)
	err = nil
	select {
	case err = <-ch:
		break
	case delProof = <-chProof:
		break
	case <-timeOut.C:
		err = errors.New("timeout")
	}

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
	// TODO: send delProof to tee
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
	verifyCommitOrDeletionProof, err := n.PoisRequestVerifyDeletionProof(os.Args[2], delProof.Roots, witChain, delProof.AccPath, n.GetSignatureAccPulickey(), n.Pois.RsaKey.N.Bytes(), n.Pois.RsaKey.G.Bytes(), time.Duration(time.Minute*5))
	if err != nil {
		n.AccRollback(true)
		n.Replace("err", fmt.Sprintf("[PoisRequestVerifyDeletionProof] %v", err))
		return
	}

	var idleSignInfo pattern.SpaceProofInfo
	minerAcc, _ := types.NewAccountID(n.GetSignatureAccPulickey())
	idleSignInfo.Miner = *minerAcc
	idleSignInfo.Front = types.U64(verifyCommitOrDeletionProof.PoisStatus.Front)
	idleSignInfo.Rear = types.U64(verifyCommitOrDeletionProof.PoisStatus.Rear)
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
	for i := 0; i < len(verifyCommitOrDeletionProof.SignatureAbove); i++ {
		sign[i] = types.U8(verifyCommitOrDeletionProof.SignatureAbove[i])
	}

	//
	txhash, err := n.ReplaceIdleSpace(idleSignInfo, sign)
	if err != nil {
		n.AccRollback(true)
		n.Replace("err", err.Error())
		return
	}

	n.Replace("info", fmt.Sprintf("Replace files suc: %v", txhash))

	challenge, err := n.QueryChallenge_V2()
	if err != nil {
		if err.Error() != pattern.ERR_Empty {
			n.Replace("err", err.Error())
			return //errors.Wrapf(err, "[QueryChallenge]")
		}
	}

	for _, v := range challenge.MinerSnapshotList {
		if sutils.CompareSlice(n.GetSignatureAccPulickey(), v.Miner[:]) {
			if int64(v.SpaceProofInfo.Front) != n.Prover.GetFront() || int64(v.SpaceProofInfo.Rear) != n.Prover.GetRear() {
				minerInfo, err := n.QueryStorageMiner_V2(n.GetSignatureAccPulickey())
				if err != nil {
					return
				}
				var acc = make([]byte, len(pattern.Accumulator{}))
				for i := 0; i < len(acc); i++ {
					acc[i] = byte(minerInfo.SpaceProofInfo.Accumulator[i])
				}
				err = n.Prover.SetChallengeState(*n.Pois.RsaKey, acc, int64(minerInfo.SpaceProofInfo.Front), int64(minerInfo.SpaceProofInfo.Rear))
				if err != nil {
					return
				}
			}
		}
	}

	err = n.Prover.DeleteFiles()
	if err != nil {
		return
	}
}
