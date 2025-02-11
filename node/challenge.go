/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"

	"github.com/CESSProject/cess-go-sdk/chain"
)

func (n *Node) ChallengeMgt(idleChallTaskCh chan bool, serviceChallTaskCh chan bool) {
	haveChall, challenge, err := n.QueryChallengeSnapShot(n.GetSignatureAccPulickey(), -1)
	if err != nil {
		if err.Error() != chain.ERR_Empty {
			n.Ichal("err", fmt.Sprintf("[QueryChallengeInfo] %v", err))
			n.Schal("err", fmt.Sprintf("[QueryChallengeInfo] %v", err))
		}
		return
	}

	if !haveChall {
		return
	}

	latestBlock, err := n.QueryBlockNumber("")
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
					<-idleChallTaskCh
					go n.idleChallenge(
						idleChallTaskCh,
						uint32(challenge.ChallengeElement.Start),
						uint32(challenge.ChallengeElement.IdleSlip),
						uint32(challenge.ChallengeElement.VerifySlip),
						int64(challenge.MinerSnapshot.SpaceProofInfo.Front),
						int64(challenge.MinerSnapshot.SpaceProofInfo.Rear),
						challenge.ChallengeElement.SpaceParam,
						challenge.MinerSnapshot.SpaceProofInfo.Accumulator,
						challenge.MinerSnapshot.TeeSig,
					)
				}
			}
		}
	} else {
		if uint32(challenge.ChallengeElement.IdleSlip) < latestBlock {
			n.Ichal("err", fmt.Sprintf("idle data challenge has expired: %v < %v", uint32(challenge.ChallengeElement.IdleSlip), latestBlock))
		} else {
			if len(idleChallTaskCh) > 0 {
				<-idleChallTaskCh
				go n.idleChallenge(
					idleChallTaskCh,
					uint32(challenge.ChallengeElement.Start),
					uint32(challenge.ChallengeElement.IdleSlip),
					uint32(challenge.ChallengeElement.VerifySlip),
					int64(challenge.MinerSnapshot.SpaceProofInfo.Front),
					int64(challenge.MinerSnapshot.SpaceProofInfo.Rear),
					challenge.ChallengeElement.SpaceParam,
					challenge.MinerSnapshot.SpaceProofInfo.Accumulator,
					challenge.MinerSnapshot.TeeSig,
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
					<-serviceChallTaskCh
					go n.serviceChallenge(
						serviceChallTaskCh,
						challenge.ChallengeElement.ServiceParam.Index,
						challenge.ChallengeElement.ServiceParam.Value,
						uint32(challenge.ChallengeElement.Start),
						uint32(challenge.ChallengeElement.ServiceSlip),
						uint32(challenge.ChallengeElement.VerifySlip),
					)
				}
			}
		}
	} else {
		if uint32(challenge.ChallengeElement.ServiceSlip) < latestBlock {
			n.Schal("err", fmt.Sprintf("service challenge has expired: %v < %v", uint32(challenge.ChallengeElement.ServiceSlip), latestBlock))
		} else {
			if len(serviceChallTaskCh) > 0 {
				<-serviceChallTaskCh
				go n.serviceChallenge(
					serviceChallTaskCh,
					challenge.ChallengeElement.ServiceParam.Index,
					challenge.ChallengeElement.ServiceParam.Value,
					uint32(challenge.ChallengeElement.Start),
					uint32(challenge.ChallengeElement.ServiceSlip),
					uint32(challenge.ChallengeElement.VerifySlip),
				)
			}
		}
	}
}
