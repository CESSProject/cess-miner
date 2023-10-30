/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"math/big"
	"time"

	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/CESSProject/cess_pois/acc"
	"github.com/CESSProject/cess_pois/pois"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58"
)

func (n *Node) replaceIdle(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	for {
		if n.GetChainState() {
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

			var verifyCommitOrDeletionProof *pb.ResponseVerifyCommitOrDeletionProof
			var workTeePeerID peer.ID
			tees := n.GetAllTeeWorkPeerId()
			utils.RandSlice(tees)
			for _, t := range tees {
				teePeerId := base58.Encode(t)
				addr, ok := n.GetPeer(teePeerId)
				if !ok {
					n.Replace("err", fmt.Sprintf("Not found tee: %s", teePeerId))
					continue
				}
				err = n.Connect(n.GetCtxQueryFromCtxCancel(), addr)
				if err != nil {
					n.Replace("err", fmt.Sprintf("Connect %s err: %v", addr.ID.Pretty(), err))
					continue
				}
				verifyCommitOrDeletionProof, err = n.PoisRequestVerifyDeletionProofP2P(addr.ID, delProof.Roots, witChain, delProof.AccPath, n.GetSignatureAccPulickey(), n.Pois.RsaKey.N.Bytes(), n.Pois.RsaKey.G.Bytes(), time.Duration(time.Minute*10))
				if err != nil {
					n.Replace("err", fmt.Sprintf("[PoisRequestVerifyDeletionProofP2P] %v", err))
					continue
				}
				workTeePeerID = addr.ID
				break
			}

			if workTeePeerID.String() == "" {
				n.AccRollback(true)
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

			err = n.Prover.UpdateStatus(int64(num), true)
			if err != nil {
				n.Replace("err", fmt.Sprintf("[UpdateStatus] %v", err))
			}

			n.Replace("info", fmt.Sprintf("Replace files suc: %v", txhash))

			challenge, err := n.QueryChallenge_V2()
			if err != nil {
				if err.Error() != pattern.ERR_Empty {
					n.Replace("err", err.Error())
					return
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
				n.Replace("err", err.Error())
			}
			n.Replace("info", fmt.Sprintf("Successfully replaced %d idle files", num))
		}
		time.Sleep(pattern.BlockInterval)
	}
}
