/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"

	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
)

func (n *Node) challenge(idleChallTaskCh, serviceChallTaskCh chan bool) {
	var idleProofSubmited bool = true
	var serviceProofSubmited bool = true
	var idleChallResult bool
	var serviceChallResult bool
	var idleChallTeeAcc string
	var serviceChallTeeAcc string
	var minerSnapShot pattern.MinerSnapShot_V2

	challenge, err := n.QueryChallenge_V2()
	if err != nil {
		if err.Error() != pattern.ERR_Empty {
			n.Ichal("err", fmt.Sprintf("[QueryChallenge] %v", err))
			n.Schal("err", fmt.Sprintf("[QueryChallenge] %v", err))
		}
		return
	}

	latestBlock, err := n.QueryBlockHeight("")
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[QueryBlockHeight] %v", err))
		n.Schal("err", fmt.Sprintf("[QueryBlockHeight] %v", err))
		return
	}
	challExpiration, err := n.QueryChallengeExpiration()
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[QueryChallengeExpiration] %v", err))
		n.Schal("err", fmt.Sprintf("[QueryChallengeExpiration] %v", err))
		return
	}

	for _, v := range challenge.MinerSnapshotList {
		if sutils.CompareSlice(n.GetSignatureAccPulickey(), v.Miner[:]) {
			idleProofSubmited = bool(v.IdleSubmitted)
			serviceProofSubmited = bool(v.ServiceSubmitted)
			if !v.IdleSubmitted {
				if len(idleChallTaskCh) > 0 {
					_ = <-idleChallTaskCh
					go n.idleChallenge(
						idleChallTaskCh,
						idleProofSubmited,
						latestBlock,
						challExpiration,
						uint32(challenge.NetSnapShot.Start),
						int64(v.SpaceProofInfo.Front),
						int64(v.SpaceProofInfo.Rear),
						challenge.NetSnapShot.SpaceChallengeParam,
						v.SpaceProofInfo.Accumulator,
					)
				}
			}

			if !v.ServiceSubmitted {
				if len(serviceChallTaskCh) > 0 {
					_ = <-serviceChallTaskCh
					go n.serviceChallenge(
						serviceChallTaskCh,
						serviceProofSubmited,
						latestBlock,
						challExpiration,
						uint32(challenge.NetSnapShot.Start),
						challenge.NetSnapShot.RandomIndexList,
						challenge.NetSnapShot.RandomList,
					)
				}
			}
			break
		}
	}

	idleChallResult = false
	serviceChallResult = false
	teeAccounts := n.GetAllTeeWorkAccount()
	for _, v := range teeAccounts {
		if idleChallResult && serviceChallResult {
			break
		}
		publickey, err := sutils.ParsingPublickey(v)
		if err != nil {
			continue
		}
		if !idleChallResult {
			idleProofInfos, err := n.QueryUnverifiedIdleProof(publickey)
			if err == nil {
				for i := 0; i < len(idleProofInfos); i++ {
					if sutils.CompareSlice(idleProofInfos[i].MinerSnapShot.Miner[:], n.GetSignatureAccPulickey()) {
						idleChallResult = true
						idleChallTeeAcc = v
						minerSnapShot = idleProofInfos[i].MinerSnapShot
						break
					}
				}
			}
		}
		if !serviceChallResult {
			serviceProofInfos, err := n.QueryUnverifiedServiceProof(publickey)
			if err == nil {
				for i := 0; i < len(serviceProofInfos); i++ {
					if sutils.CompareSlice(serviceProofInfos[i].MinerSnapShot.Miner[:], n.GetSignatureAccPulickey()) {
						serviceChallResult = true
						serviceChallTeeAcc = v
						minerSnapShot = serviceProofInfos[i].MinerSnapShot
						break
					}
				}
			}
		}
	}

	if idleChallResult || serviceChallResult {
		latestBlock, err := n.QueryBlockHeight("")
		if err != nil {
			n.Ichal("err", fmt.Sprintf("[QueryBlockHeight] %v", err))
			n.Schal("err", fmt.Sprintf("[QueryBlockHeight] %v", err))
			return
		}

		challVerifyExpiration, err := n.QueryChallengeVerifyExpiration()
		if err != nil {
			n.Ichal("err", fmt.Sprintf("[QueryChallengeExpiration] %v", err))
			n.Schal("err", fmt.Sprintf("[QueryChallengeExpiration] %v", err))
			return
		}

		if idleChallResult {
			if len(idleChallTaskCh) > 0 {
				_ = <-idleChallTaskCh
				go n.poisChallengeResult(idleChallTaskCh, latestBlock, challVerifyExpiration, idleChallTeeAcc, challenge, minerSnapShot)
			}
		}

		if serviceChallResult {
			if len(serviceChallTaskCh) > 0 {
				_ = <-serviceChallTaskCh
				go n.serviceChallengeResult(serviceChallTaskCh, latestBlock, challVerifyExpiration, serviceChallTeeAcc, challenge)
			}
		}
	}
}
