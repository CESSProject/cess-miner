/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"

	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/proof"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/cess-go-sdk/core/sdk"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/pb"
)

func ChallengeMgt(
	cli sdk.SDK,
	l logger.Logger,
	ws *Workspace,
	r *RunningState,
	teeRecord *TeeRecord,
	peernode *core.PeerNode,
	m *pb.MinerPoisInfo,
	rsa *proof.RSAKeyPair,
	p *Pois,
	cace cache.Cache,
	idleChallTaskCh chan bool,
	serviceChallTaskCh chan bool,
) {
	haveChall, challenge, err := cli.QueryChallengeInfo(cli.GetSignatureAccPulickey())
	if err != nil {
		if err.Error() != pattern.ERR_Empty {
			l.Ichal("err", fmt.Sprintf("[QueryChallengeInfo] %v", err))
			l.Schal("err", fmt.Sprintf("[QueryChallengeInfo] %v", err))
		}
		return
	}

	if !haveChall {
		return
	}

	latestBlock, err := cli.QueryBlockHeight("")
	if err != nil {
		l.Ichal("err", fmt.Sprintf("[QueryBlockHeight] %v", err))
		l.Schal("err", fmt.Sprintf("[QueryBlockHeight] %v", err))
		return
	}

	if len(idleChallTaskCh) > 0 {
		l.Ichal("info", fmt.Sprintf("challenge start: %v latestBlock: %v", challenge.ChallengeElement.Start, latestBlock))
	}

	if len(serviceChallTaskCh) > 0 {
		l.Schal("info", fmt.Sprintf("challenge start: %v latestBlock: %v", challenge.ChallengeElement.Start, latestBlock))
	}

	if challenge.ProveInfo.IdleProve.HasValue() {
		_, idleProve := challenge.ProveInfo.IdleProve.Unwrap()
		if !idleProve.VerifyResult.HasValue() {
			if uint32(challenge.ChallengeElement.VerifySlip) < latestBlock {
				l.Ichal("err", fmt.Sprintf("idle data challenge verification expired: %v < %v", uint32(challenge.ChallengeElement.VerifySlip), latestBlock))
			} else {
				if len(idleChallTaskCh) > 0 {
					<-idleChallTaskCh
					go idleChallenge(
						cli, r, l, m, rsa, p, teeRecord, peernode, ws,
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
			l.Ichal("err", fmt.Sprintf("idle data challenge has expired: %v < %v", uint32(challenge.ChallengeElement.IdleSlip), latestBlock))
		} else {
			if len(idleChallTaskCh) > 0 {
				<-idleChallTaskCh
				r.SetIdleChallengeFlag(true)
				go idleChallenge(
					cli, r, l, m, rsa, p, teeRecord, peernode, ws,
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
				l.Schal("err", fmt.Sprintf("service data challenge verification expired: %v < %v", uint32(challenge.ChallengeElement.VerifySlip), latestBlock))
			} else {
				if len(serviceChallTaskCh) > 0 {
					<-serviceChallTaskCh
					go serviceChallenge(
						cli, r, l, teeRecord, peernode, ws, cace, rsa,
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
			l.Schal("err", fmt.Sprintf("service challenge has expired: %v < %v", uint32(challenge.ChallengeElement.ServiceSlip), latestBlock))
		} else {
			if len(serviceChallTaskCh) > 0 {
				<-serviceChallTaskCh
				r.SetServiceChallengeFlag(true)
				go serviceChallenge(
					cli, r, l, teeRecord, peernode, ws, cace, rsa,
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
