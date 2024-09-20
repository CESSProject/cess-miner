/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/pkg/logger"
	"github.com/CESSProject/cess-miner/pkg/utils"
	"github.com/CESSProject/cess_pois/acc"
	"github.com/CESSProject/cess_pois/pois"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

func ReplaceIdle(cli *chain.ChainClient, l logger.Logger, p *Pois, m *pb.MinerPoisInfo, teeRecord *TeeRecord, peernode *core.PeerNode, ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			l.Pnc(utils.RecoverError(err))
		}
	}()

	replaceSize, err := cli.QueryPendingReplacements(cli.GetSignatureAccPulickey(), -1)
	if err != nil {
		if err.Error() != chain.ERR_Empty {
			l.Replace("err", err.Error())
		}
		return
	}

	if replaceSize.CmpAbs(big.NewInt(0)) <= 0 {
		return
	}

	if !replaceSize.IsUint64() {
		l.Replace("err", "replaceSize is not uint64")
		return
	}

	l.Replace("info", fmt.Sprintf("replace size: %v", replaceSize.Uint64()))
	num := uint64(replaceSize.Uint64() / 1024 / 1024 / uint64(pois.FileSize))
	if num == 0 {
		l.Replace("info", "no files to replace")
		return
	}

	if int64(num) > int64((int64(acc.DEFAULT_ELEMS_NUM) - p.GetFront()%int64(acc.DEFAULT_ELEMS_NUM))) {
		num = uint64((int64(acc.DEFAULT_ELEMS_NUM) - p.GetFront()%int64(acc.DEFAULT_ELEMS_NUM)))
	}

	l.Replace("info", fmt.Sprintf("Will replace %d idle files", num))

	delProof, err := p.Prover.ProveDeletion(int64(num))
	if err != nil {
		l.Replace("err", err.Error())
		p.Prover.AccRollback(true)
		return
	}

	if delProof == nil {
		l.Replace("err", "delProof is nil")
		p.Prover.AccRollback(true)
		return
	}

	if delProof.Roots == nil || delProof.AccPath == nil || delProof.WitChain == nil {
		l.Replace("err", "delProof have nil field")
		p.Prover.AccRollback(true)
		return
	}

	minerInfo, err := cli.QueryMinerItems(cli.GetSignatureAccPulickey(), -1)
	if err != nil {
		l.Replace("err", fmt.Sprintf("[QueryStorageMiner] %v", err))
		p.Prover.AccRollback(true)
		return
	}
	if minerInfo.SpaceProofInfo.HasValue() {
		_, spaceProofInfo := minerInfo.SpaceProofInfo.Unwrap()
		if spaceProofInfo.Front > types.U64(p.Prover.GetFront()) {
			err = p.Prover.SyncChainPoisStatus(int64(spaceProofInfo.Front), int64(spaceProofInfo.Rear))
			if err != nil {
				l.Replace("err", err.Error())
				p.Prover.AccRollback(true)
				return
			}
		}
		m.Front = int64(spaceProofInfo.Front)
		m.Rear = int64(spaceProofInfo.Rear)
		m.Acc = []byte(string(spaceProofInfo.Accumulator[:]))
		m.StatusTeeSign = []byte(string(minerInfo.TeeSig[:]))
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
		MinerId:  cli.GetSignatureAccPulickey(),
		PoisInfo: m,
	}
	buf, err := proto.Marshal(requestVerifyDeletionProof)
	if err != nil {
		p.Prover.AccRollback(true)
		l.Replace("err", fmt.Sprintf("[Marshal-2] %v", err))
		return
	}
	signData, err := cli.Sign(buf)
	if err != nil {
		p.Prover.AccRollback(true)
		l.Replace("err", fmt.Sprintf("[Sign-2] %v", err))
		return
	}
	requestVerifyDeletionProof.MinerSign = signData
	var verifyCommitOrDeletionProof *pb.ResponseVerifyCommitOrDeletionProof
	var usedTeeEndPoint string
	var usedTeeWorkAccount string
	var timeout time.Duration
	var timeoutStep time.Duration = 3
	var dialOptions []grpc.DialOption
	teeEndPoints := teeRecord.GetAllMarkerTeeEndpoint()
	for _, t := range teeEndPoints {
		timeout = time.Duration(time.Minute * timeoutStep)
		l.Replace("info", fmt.Sprintf("Will use tee: %v", t))
		if !strings.Contains(t, "443") {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		} else {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
		}
		for try := 2; try <= 6; try += 2 {
			verifyCommitOrDeletionProof, err = peernode.RequestVerifyDeletionProof(
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
				l.Replace("err", fmt.Sprintf("[RequestVerifyDeletionProof] %v", err))
				break
			}
			usedTeeEndPoint = t
			usedTeeWorkAccount, err = teeRecord.GetTeeWorkAccount(usedTeeEndPoint)
			if err != nil {
				l.Space("err", fmt.Sprintf("[GetTeeWorkAccount(%s)] %v", usedTeeEndPoint, err))
			}
			break
		}
		if usedTeeEndPoint != "" && usedTeeWorkAccount != "" {
			break
		}
	}

	if usedTeeEndPoint == "" || usedTeeWorkAccount == "" {
		p.AccRollback(true)
		l.Replace("err", "No available tee")
		return
	}

	var idleSignInfo chain.SpaceProofInfo
	minerAcc, _ := types.NewAccountID(cli.GetSignatureAccPulickey())
	idleSignInfo.Miner = *minerAcc
	idleSignInfo.Front = types.U64(verifyCommitOrDeletionProof.PoisStatus.Front)
	idleSignInfo.Rear = types.U64(verifyCommitOrDeletionProof.PoisStatus.Rear)

	if len(verifyCommitOrDeletionProof.StatusTeeSign) != chain.TeeSigLen {
		p.AccRollback(true)
		l.Replace("err", "invalid tee sign length")
		return
	}

	for i := 0; i < len(verifyCommitOrDeletionProof.PoisStatus.Acc); i++ {
		idleSignInfo.Accumulator[i] = types.U8(verifyCommitOrDeletionProof.PoisStatus.Acc[i])
	}
	g_byte := p.RsaKey.G.Bytes()
	n_byte := p.RsaKey.N.Bytes()
	for i := 0; i < len(g_byte); i++ {
		idleSignInfo.PoisKey.G[i] = types.U8(g_byte[i])
	}
	for i := 0; i < len(n_byte); i++ {
		idleSignInfo.PoisKey.N[i] = types.U8(n_byte[i])
	}

	var sign chain.TeeSig
	for i := 0; i < chain.TeeSigLen; i++ {
		sign[i] = types.U8(verifyCommitOrDeletionProof.StatusTeeSign[i])
	}
	var signWithAcc chain.TeeSig
	for i := 0; i < chain.TeeSigLen; i++ {
		signWithAcc[i] = types.U8(verifyCommitOrDeletionProof.SignatureWithTeeController[i])
	}
	//
	var teeSignBytes = make(types.Bytes, len(sign))
	for j := 0; j < len(sign); j++ {
		teeSignBytes[j] = byte(sign[j])
	}
	var signWithAccBytes = make(types.Bytes, len(signWithAcc))
	for j := 0; j < len(sign); j++ {
		signWithAccBytes[j] = byte(signWithAcc[j])
	}
	wpuk, err := chain.BytesToWorkPublickey([]byte(usedTeeWorkAccount))
	if err != nil {
		p.AccRollback(true)
		l.Replace("err", err.Error())
		return
	}
	txhash, err := cli.ReplaceIdleSpace(idleSignInfo, signWithAccBytes, teeSignBytes, wpuk)
	if err != nil || txhash == "" {
		p.AccRollback(true)
		l.Replace("err", err.Error())
		return
	}

	l.Replace("info", fmt.Sprintf("Replace files suc: %v", txhash))

	err = p.Prover.UpdateStatus(int64(num), true)
	if err != nil {
		l.Replace("err", err.Error())
	}

	l.Replace("info", fmt.Sprintf("new acc value: %s", hex.EncodeToString(p.Prover.GetAccValue())))

	ok, challenge, err := cli.QueryChallengeSnapShot(cli.GetSignatureAccPulickey(), -1)
	if err != nil {
		if err.Error() != chain.ERR_Empty {
			l.Replace("err", err.Error())
			return
		}
	}

	if ok {
		err = p.Prover.SetChallengeState(*p.RsaKey, []byte(string(challenge.MinerSnapshot.SpaceProofInfo.Accumulator[:])), int64(challenge.MinerSnapshot.SpaceProofInfo.Front), int64(challenge.MinerSnapshot.SpaceProofInfo.Rear))
		if err != nil {
			l.Replace("err", err.Error())
			return
		}
	}

	err = p.Prover.DeleteFiles()
	if err != nil {
		l.Replace("err", err.Error())
	}
	l.Replace("info", fmt.Sprintf("Successfully replaced %d idle files", num))
}
