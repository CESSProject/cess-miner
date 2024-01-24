/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"

	"github.com/CESSProject/cess-go-sdk/core/pattern"
)

func (n *Node) challengeMgt(idleChallTaskCh, serviceChallTaskCh chan bool) {
	chainSt := n.GetChainState()
	if !chainSt {
		return
	}

	minerSt := n.GetMinerState()
	if minerSt != pattern.MINER_STATE_POSITIVE &&
		minerSt != pattern.MINER_STATE_FROZEN {
		return
	}

	haveChall, challenge, err := n.QueryChallengeInfo(n.GetSignatureAccPulickey())
	if err != nil {
		if err.Error() != pattern.ERR_Empty {
			n.Ichal("err", fmt.Sprintf("[QueryChallengeInfo] %v", err))
			n.Schal("err", fmt.Sprintf("[QueryChallengeInfo] %v", err))
		}
		return
	}

	if !haveChall {
		return
	}

	latestBlock, err := n.QueryBlockHeight("")
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[QueryBlockHeight] %v", err))
		n.Schal("err", fmt.Sprintf("[QueryBlockHeight] %v", err))
		return
	}

	if len(idleChallTaskCh) > 0 {
		n.Ichal("info", fmt.Sprintf("challenge start: %v latestBlock: %v", challenge.ChallengeElement.Start, latestBlock))
	}

	if len(serviceChallTaskCh) > 0 {
		n.Schal("info", fmt.Sprintf("challenge start: %v latestBlock: %v", challenge.ChallengeElement.Start, latestBlock))
	}

	if challenge.ProveInfo.IdleProve.HasValue() {
		_, idleProve := challenge.ProveInfo.IdleProve.Unwrap()
		if !idleProve.VerifyResult.HasValue() {
			if uint32(challenge.ChallengeElement.VerifySlip) < latestBlock {
				n.Ichal("err", fmt.Sprintf("idle data challenge verification expired: %v < %v", uint32(challenge.ChallengeElement.VerifySlip), latestBlock))
			} else {
				if len(idleChallTaskCh) > 0 {
					_ = <-idleChallTaskCh
					go n.idleChallenge(
						idleChallTaskCh,
						true,
						latestBlock,
						uint32(challenge.ChallengeElement.VerifySlip),
						uint32(challenge.ChallengeElement.Start),
						int64(challenge.MinerSnapshot.SpaceProofInfo.Front),
						int64(challenge.MinerSnapshot.SpaceProofInfo.Rear),
						challenge.ChallengeElement.SpaceParam,
						challenge.MinerSnapshot.SpaceProofInfo.Accumulator,
						challenge.MinerSnapshot.TeeSig,
						idleProve.TeePubkey,
					)
				}
			}
		}
	} else {
		if uint32(challenge.ChallengeElement.IdleSlip) < latestBlock {
			n.Ichal("err", fmt.Sprintf("idle data challenge has expired: %v < %v", uint32(challenge.ChallengeElement.IdleSlip), latestBlock))
		} else {
			if len(idleChallTaskCh) > 0 {
				_ = <-idleChallTaskCh
				go n.idleChallenge(
					idleChallTaskCh,
					false,
					latestBlock,
					uint32(challenge.ChallengeElement.VerifySlip),
					uint32(challenge.ChallengeElement.Start),
					int64(challenge.MinerSnapshot.SpaceProofInfo.Front),
					int64(challenge.MinerSnapshot.SpaceProofInfo.Rear),
					challenge.ChallengeElement.SpaceParam,
					challenge.MinerSnapshot.SpaceProofInfo.Accumulator,
					challenge.MinerSnapshot.TeeSig,
					pattern.WorkerPublicKey{},
				)
			}
		}
	}

	if challenge.ProveInfo.ServiceProve.HasValue() {
		_, serviceProve := challenge.ProveInfo.ServiceProve.Unwrap()
		if !serviceProve.VerifyResult.HasValue() {
			if uint32(challenge.ChallengeElement.VerifySlip) < latestBlock {
				n.Schal("err", fmt.Sprintf("service data challenge verification expired: %v < %v", uint32(challenge.ChallengeElement.VerifySlip), latestBlock))
			} else {
				if len(serviceChallTaskCh) > 0 {
					_ = <-serviceChallTaskCh
					go n.serviceChallenge(
						serviceChallTaskCh,
						true,
						latestBlock,
						uint32(challenge.ChallengeElement.VerifySlip),
						uint32(challenge.ChallengeElement.Start),
						challenge.ChallengeElement.ServiceParam.Index,
						challenge.ChallengeElement.ServiceParam.Value,
						serviceProve.TeePubkey,
					)
				}
			}
		}
	} else {
		if uint32(challenge.ChallengeElement.ServiceSlip) < latestBlock {
			n.Schal("err", fmt.Sprintf("service challenge has expired: %v < %v", uint32(challenge.ChallengeElement.ServiceSlip), latestBlock))
		} else {
			if len(serviceChallTaskCh) > 0 {
				_ = <-serviceChallTaskCh
				go n.serviceChallenge(
					serviceChallTaskCh,
					false,
					latestBlock,
					uint32(challenge.ChallengeElement.VerifySlip),
					uint32(challenge.ChallengeElement.Start),
					challenge.ChallengeElement.ServiceParam.Index,
					challenge.ChallengeElement.ServiceParam.Value,
					pattern.WorkerPublicKey{},
				)
			}
		}
	}
}
