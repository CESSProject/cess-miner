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

	n.Ichal("info", fmt.Sprintf("challenge.start: %v", challenge.ChallengeElement.Start))
	n.Schal("info", fmt.Sprintf("challenge.start: %v", challenge.ChallengeElement.Start))

	latestBlock, err := n.QueryBlockHeight("")
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[QueryBlockHeight] %v", err))
		n.Schal("err", fmt.Sprintf("[QueryBlockHeight] %v", err))
		return
	}
	n.Ichal("info", fmt.Sprintf("latestBlock: %v", latestBlock))
	n.Schal("info", fmt.Sprintf("latestBlock: %v", latestBlock))

	if len(challenge.ProveInfo.IdleProve.IdleProve) == 0 {
		if uint32(challenge.ChallengeElement.IdleSlip) < latestBlock {
			n.Ichal("err", fmt.Sprintf("idle challenge has expired: %v < %v", uint32(challenge.ChallengeElement.IdleSlip), latestBlock))
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
					challenge.MinerSnapshot.TeeSignature,
				)
			}
		}
	} else {
		if uint32(challenge.ChallengeElement.VerifySlip) < latestBlock {
			n.Ichal("err", fmt.Sprintf("idle challenge verification expired: %v < %v", uint32(challenge.ChallengeElement.VerifySlip), latestBlock))
		} else {
			if !challenge.ProveInfo.IdleProve.VerifyResult {
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
						challenge.MinerSnapshot.TeeSignature,
					)
				}
			}
		}
	}

	if len(challenge.ProveInfo.ServiceProve.ServiceProve) == 0 {
		if uint32(challenge.ChallengeElement.ServiceSlip) < latestBlock {
			n.Ichal("err", fmt.Sprintf("service challenge has expired: %v < %v", uint32(challenge.ChallengeElement.ServiceSlip), latestBlock))
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
				)
			}
		}
	} else {
		if uint32(challenge.ChallengeElement.VerifySlip) < latestBlock {
			n.Ichal("err", fmt.Sprintf("service challenge verification expired: %v < %v", uint32(challenge.ChallengeElement.VerifySlip), latestBlock))
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
				)
			}
		}
	}
}
