/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
)

func (n *Node) challengeMgt(idleChallTaskCh, serviceChallTaskCh chan bool) {
	var idleProofSubmited bool = true
	var serviceProofSubmited bool = true
	var idleChallResult bool
	var serviceChallResult bool
	var challSuc bool = true
	challenge, err := n.QueryChallenge_V2()
	if err != nil {
		if err.Error() != pattern.ERR_Empty {
			n.Ichal("err", fmt.Sprintf("[QueryChallenge] %v", err))
			n.Schal("err", fmt.Sprintf("[QueryChallenge] %v", err))
		}
		return
	}
	n.Ichal("info", fmt.Sprintf("challenge.start: %v", challenge.NetSnapShot.Start))
	n.Schal("info", fmt.Sprintf("challenge.start: %v", challenge.NetSnapShot.Start))
	latestBlock, err := n.QueryBlockHeight("")
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[QueryBlockHeight] %v", err))
		n.Schal("err", fmt.Sprintf("[QueryBlockHeight] %v", err))
		return
	}
	n.Ichal("info", fmt.Sprintf("latestBlock: %v", latestBlock))
	n.Schal("info", fmt.Sprintf("latestBlock: %v", latestBlock))
	challVerifyExpiration, err := n.QueryChallengeVerifyExpiration()
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[QueryChallengeExpiration] %v", err))
		n.Schal("err", fmt.Sprintf("[QueryChallengeExpiration] %v", err))
		return
	}
	n.Ichal("info", fmt.Sprintf("challVerifyExpiration: %v", challVerifyExpiration))
	n.Schal("info", fmt.Sprintf("challVerifyExpiration: %v", challVerifyExpiration))
	for _, v := range challenge.MinerSnapshotList {
		if sutils.CompareSlice(n.GetSignatureAccPulickey(), v.Miner[:]) {
			challSuc = false
			idleProofSubmited = bool(v.IdleSubmitted)
			serviceProofSubmited = bool(v.ServiceSubmitted)
			n.Ichal("info", fmt.Sprintf("IdleSubmitted: %v", v.IdleSubmitted))
			n.Schal("info", fmt.Sprintf("ServiceSubmitted: %v", v.ServiceSubmitted))
			n.Ichal("info", fmt.Sprintf("IdleLife: %v", v.IdleLife))
			n.Schal("info", fmt.Sprintf("ServiceLife: %v", v.ServiceLife))
			if !v.IdleSubmitted {
				if len(idleChallTaskCh) > 0 {
					_ = <-idleChallTaskCh
					go n.idleChallenge(
						idleChallTaskCh,
						idleProofSubmited,
						latestBlock,
						uint32(v.IdleLife+challenge.NetSnapShot.Start),
						challVerifyExpiration,
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
						uint32(v.ServiceLife+challenge.NetSnapShot.Start),
						challVerifyExpiration,
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
						challSuc = false
						idleChallResult = true
						n.Ichal("info", fmt.Sprintf("IdleLife2: %v", idleProofInfos[i].MinerSnapShot.IdleLife))
						if len(idleChallTaskCh) > 0 {
							_ = <-idleChallTaskCh
							go n.idleChallenge(
								idleChallTaskCh,
								idleProofSubmited,
								latestBlock,
								uint32(idleProofInfos[i].MinerSnapShot.IdleLife+challenge.NetSnapShot.Start),
								challVerifyExpiration,
								uint32(challenge.NetSnapShot.Start),
								int64(idleProofInfos[i].MinerSnapShot.SpaceProofInfo.Front),
								int64(idleProofInfos[i].MinerSnapShot.SpaceProofInfo.Rear),
								challenge.NetSnapShot.SpaceChallengeParam,
								idleProofInfos[i].MinerSnapShot.SpaceProofInfo.Accumulator,
							)
						}
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
						challSuc = false
						serviceChallResult = true
						n.Schal("info", fmt.Sprintf("ServiceLife2: %v", serviceProofInfos[i].MinerSnapShot.IdleLife))
						if len(serviceChallTaskCh) > 0 {
							_ = <-serviceChallTaskCh
							go n.serviceChallenge(
								serviceChallTaskCh,
								serviceProofSubmited,
								latestBlock,
								uint32(serviceProofInfos[i].MinerSnapShot.ServiceLife+challenge.NetSnapShot.Start),
								challVerifyExpiration,
								uint32(challenge.NetSnapShot.Start),
								challenge.NetSnapShot.RandomIndexList,
								challenge.NetSnapShot.RandomList,
							)
						}
						break
					}
				}
			}
		}
	}
	if challSuc {
		if challVerifyExpiration > latestBlock {
			n.Ichal("info", fmt.Sprintf("challenge complete and sleep %ds", ((challVerifyExpiration-latestBlock)*4)))
			n.Schal("info", fmt.Sprintf("challenge complete and sleep %ds", ((challVerifyExpiration-latestBlock)*4)))
			n.chalTick.Reset(time.Second * time.Duration((challVerifyExpiration-latestBlock)*4))
		} else {
			n.Ichal("info", "challenge complete")
			n.Schal("info", "challenge complete")
			n.chalTick.Reset(time.Second * time.Duration(6+rand.Intn(30)))
		}
	} else {
		n.Ichal("info", "challenge go on")
		n.Schal("info", "challenge go on")
		n.chalTick.Reset(time.Second * time.Duration(6+rand.Intn(30)))
	}
}
