/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"

	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/CESSProject/cess-miner/pkg/cache"
	"github.com/CESSProject/cess-miner/pkg/confile"
	"github.com/CESSProject/cess-miner/pkg/logger"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/pb"
)

func ChallengeMgt(
	cli *chain.ChainClient,
	l logger.Logger,
	ws *Workspace,
	r *RunningState,
	teeRecord *TeeRecord,
	peernode *core.PeerNode,
	m *pb.MinerPoisInfo,
	rsa *RSAKeyPair,
	p *Pois,
	cfg *confile.Confile,
	cace cache.Cache,
	idleChallTaskCh chan bool,
	serviceChallTaskCh chan bool,
) {
	haveChall, challenge, err := cli.QueryChallengeSnapShot(cli.GetSignatureAccPulickey(), -1)
	if err != nil {
		if err.Error() != chain.ERR_Empty {
			l.Ichal("err", fmt.Sprintf("[QueryChallengeInfo] %v", err))
			l.Schal("err", fmt.Sprintf("[QueryChallengeInfo] %v", err))
		}
		return
	}

	if !haveChall {
		return
	}

	latestBlock, err := cli.QueryBlockNumber("")
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
					chain.WorkerPublicKey{},
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
						cli, r, l, teeRecord, peernode, ws, cace, rsa, cfg,
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
					cli, r, l, teeRecord, peernode, ws, cace, rsa, cfg,
					serviceChallTaskCh,
					false,
					latestBlock,
					uint32(challenge.ChallengeElement.VerifySlip),
					uint32(challenge.ChallengeElement.Start),
					challenge.ChallengeElement.ServiceParam.Index,
					challenge.ChallengeElement.ServiceParam.Value,
					chain.WorkerPublicKey{},
				)
			}
		}
	}
}
