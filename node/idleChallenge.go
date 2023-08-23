/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/mr-tron/base58"
	"google.golang.org/protobuf/proto"
)

type fileBlockProofInfo struct {
	ProofHashSign       []byte         `json:"proofHashSign"`
	ProofHashSignOrigin []byte         `json:"proofHashSignOrigin"`
	SpaceProof          *pb.SpaceProof `json:"spaceProof"`
	FileBlockFront      int64          `json:"fileBlockFront"`
	FileBlockRear       int64          `json:"fileBlockRear"`
}

type idleProofInfo struct {
	Start                 int32   `json:"start"`
	ChainFront            int64   `json:"chainFront"`
	ChainRear             int64   `json:"chainRear"`
	IdleResult            bool    `json:"idleResult"`
	AllocatedTeeAccount   string  `json:"allocatedTeeAccount"`
	AllocatedTeeAccountId []byte  `json:"allocatedTeeAccountId"`
	IdleProof             []byte  `json:"idleProof"`
	Acc                   []byte  `json:"acc"`
	TotalSignature        []byte  `json:"totalSignature"`
	ChallRandom           []int64 `json:"challRandom"`
	FileBlockProofInfo    []fileBlockProofInfo
	BlocksProof           []*pb.BlocksProof
}

func (n *Node) poisChallenge(ch chan<- bool, latestBlock, challExpiration uint32, challenge pattern.ChallengeInfo_V2, minerChalInfo pattern.MinerSnapShot_V2) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	var haveChallenge = true

	if challExpiration <= latestBlock {
		haveChallenge = false
	}

	if !haveChallenge {
		if !minerChalInfo.IdleSubmitted {
			n.Ichal("err", "Proof of idle files not submitted")
			return
		}
	}

	n.Ichal("info", fmt.Sprintf("Idle file chain challenge: %v", challenge.NetSnapShot.Start))
	var found bool
	var idleProofRecord idleProofInfo
	buf, err := os.ReadFile(filepath.Join(n.Workspace(), "idleproof"))
	if err == nil {
		err = json.Unmarshal(buf, &idleProofRecord)
		if err == nil {
			n.Ichal("info", fmt.Sprintf("local idleproof file challenge: %v", idleProofRecord.Start))
			if idleProofRecord.Start != int32(challenge.NetSnapShot.Start) {
				os.Remove(filepath.Join(n.Workspace(), "idleproof"))
			} else {
				if !minerChalInfo.IdleSubmitted {
					var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
					for i := 0; i < len(idleProofRecord.IdleProof); i++ {
						idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
					}
					_, err = n.SubmitIdleProof(idleProve)
					if err != nil {
						n.Ichal("err", fmt.Sprintf("[SubmitIdleProof] %v", err))
						return
					}
					time.Sleep(pattern.BlockInterval)
					time.Sleep(pattern.BlockInterval)

					found = false
					teeAccounts := n.GetAllTeeWorkAccount()
					for _, v := range teeAccounts {
						if found {
							break
						}
						publickey, _ := sutils.ParsingPublickey(v)
						idleProofInfos, err := n.QueryUnverifiedIdleProof(publickey)
						if err != nil {
							continue
						}

						for i := 0; i < len(idleProofInfos); i++ {
							if sutils.CompareSlice(idleProofInfos[i].MinerSnapShot.Miner[:], n.GetSignatureAccPulickey()) {
								idleProofRecord.AllocatedTeeAccount = v
								idleProofRecord.AllocatedTeeAccountId = publickey
								found = true
								break
							}
						}
					}

					if !found {
						n.Ichal("err", "Not found allocated tee for idle proof")
						return
					}

					buf, err = json.Marshal(&idleProofRecord)
					if err != nil {
						n.Ichal("err", err.Error())
					} else {
						err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "idleproof"))
						if err != nil {
							n.Ichal("err", err.Error())
						}
					}
				} else {
					found = false
					teeAccounts := n.GetAllTeeWorkAccount()
					for _, v := range teeAccounts {
						if found {
							break
						}
						publickey, _ := sutils.ParsingPublickey(v)
						idleProofInfos, err := n.QueryUnverifiedIdleProof(publickey)
						if err != nil {
							continue
						}

						for i := 0; i < len(idleProofInfos); i++ {
							if sutils.CompareSlice(idleProofInfos[i].MinerSnapShot.Miner[:], n.GetSignatureAccPulickey()) {
								idleProofRecord.AllocatedTeeAccount = v
								idleProofRecord.AllocatedTeeAccountId = publickey
								found = true
								break
							}
						}
					}
					if !found {
						n.Ichal("err", "Not found allocated tee for idle proof")
						return
					}
					if idleProofRecord.TotalSignature != nil {
						var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
						for i := 0; i < len(idleProofRecord.IdleProof); i++ {
							idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
						}
						var teeSignature pattern.TeeSignature
						if len(idleProofRecord.TotalSignature) != len(teeSignature) {
							n.Ichal("err", "invalid spaceProofVerifyTotal signature")
							return
						}

						for i := 0; i < len(idleProofRecord.TotalSignature); i++ {
							teeSignature[i] = types.U8(idleProofRecord.TotalSignature[i])
						}

						txHash, err := n.SubmitIdleProofResult(
							idleProve,
							types.U64(idleProofRecord.ChainFront),
							types.U64(idleProofRecord.ChainRear),
							minerChalInfo.SpaceProofInfo.Accumulator,
							types.Bool(idleProofRecord.IdleResult),
							teeSignature,
							idleProofRecord.AllocatedTeeAccountId,
						)
						if err != nil {
							n.Ichal("err", fmt.Sprintf("[SubmitIdleProofResult] hash: %s, err: %v", txHash, err))
							return
						}
						n.Ichal("info", fmt.Sprintf("submit idle proof result suc: %s", txHash))
						return
					}
					if idleProofRecord.BlocksProof != nil {
						teePeerIdPubkey, _ := n.GetTeeWork(idleProofRecord.AllocatedTeeAccount)

						teeAddrInfo, ok := n.GetPeer(base58.Encode(teePeerIdPubkey))
						if !ok {
							n.Ichal("err", fmt.Sprintf("Not found peer: %s", base58.Encode(teePeerIdPubkey)))
							return
						}

						err = n.Connect(n.GetCtxQueryFromCtxCancel(), teeAddrInfo)
						if err != nil {
							n.Ichal("err", fmt.Sprintf("Connect tee peer err: %v", err))
						}
						spaceProofVerifyTotal, err := n.PoisRequestVerifySpaceTotalP2P(teeAddrInfo.ID, n.GetSignatureAccPulickey(), idleProofRecord.BlocksProof, idleProofRecord.ChainFront, idleProofRecord.ChainRear, idleProofRecord.Acc, idleProofRecord.ChallRandom, time.Duration(time.Minute*3))
						if err != nil {
							n.Ichal("err", fmt.Sprintf("[PoisRequestVerifySpaceTotalP2P] %v", err))
							return
						}
						idleProofRecord.TotalSignature = spaceProofVerifyTotal.Signature
						idleProofRecord.IdleResult = spaceProofVerifyTotal.IdleResult
						buf, err = json.Marshal(&idleProofRecord)
						if err != nil {
							n.Ichal("err", err.Error())
						} else {
							err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "idleproof"))
							if err != nil {
								n.Ichal("err", err.Error())
							}
						}
						var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
						for i := 0; i < len(idleProofRecord.IdleProof); i++ {
							idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
						}
						var teeSignature pattern.TeeSignature
						if len(idleProofRecord.TotalSignature) != len(teeSignature) {
							n.Ichal("err", "invalid spaceProofVerifyTotal signature")
							return
						}

						for i := 0; i < len(idleProofRecord.TotalSignature); i++ {
							teeSignature[i] = types.U8(idleProofRecord.TotalSignature[i])
						}

						txHash, err := n.SubmitIdleProofResult(
							idleProve,
							types.U64(idleProofRecord.ChainFront),
							types.U64(idleProofRecord.ChainRear),
							minerChalInfo.SpaceProofInfo.Accumulator,
							types.Bool(idleProofRecord.IdleResult),
							teeSignature,
							idleProofRecord.AllocatedTeeAccountId,
						)
						if err != nil {
							n.Ichal("err", fmt.Sprintf("[SubmitIdleProofResult] hash: %s, err: %v", txHash, err))
							return
						}
						n.Ichal("info", fmt.Sprintf("SubmitIdleProofResult: %s", txHash))
						return
					}
				}
				teePeerIdPubkey, _ := n.GetTeeWork(idleProofRecord.AllocatedTeeAccount)

				teeAddrInfo, ok := n.GetPeer(base58.Encode(teePeerIdPubkey))
				if !ok {
					n.Ichal("err", fmt.Sprintf("Not found peer: %s", base58.Encode(teePeerIdPubkey)))
					return
				}

				err = n.Connect(n.GetCtxQueryFromCtxCancel(), teeAddrInfo)
				if err != nil {
					n.Ichal("err", fmt.Sprintf("Connect tee peer err: %v", err))
				}
				var blocksProof = make([]*pb.BlocksProof, 0)
				for i := 0; i < len(idleProofRecord.FileBlockProofInfo); i++ {
					spaceProofVerify, err := n.PoisSpaceProofVerifySingleBlockP2P(
						teeAddrInfo.ID,
						n.GetSignatureAccPulickey(),
						idleProofRecord.ChallRandom,
						n.Pois.RsaKey.N.Bytes(),
						n.Pois.RsaKey.G.Bytes(),
						idleProofRecord.Acc,
						int64(minerChalInfo.SpaceProofInfo.Front),
						int64(minerChalInfo.SpaceProofInfo.Rear),
						idleProofRecord.FileBlockProofInfo[i].SpaceProof,
						idleProofRecord.FileBlockProofInfo[i].ProofHashSign,
						time.Duration(time.Minute*3),
					)
					if err != nil {
						n.Ichal("err", fmt.Sprintf("[PoisSpaceProofVerifySingleBlockP2P] %v", err))
						return
					}
					var block = &pb.BlocksProof{
						ProofHashAndLeftRight: &pb.ProofHashAndLeftRight{
							SpaceProofHash: idleProofRecord.FileBlockProofInfo[i].ProofHashSignOrigin,
							Left:           idleProofRecord.FileBlockProofInfo[i].FileBlockFront,
							Right:          idleProofRecord.FileBlockProofInfo[i].FileBlockRear,
						},
						Signature: spaceProofVerify.Signature,
					}
					blocksProof = append(blocksProof, block)
				}

				idleProofRecord.BlocksProof = blocksProof
				buf, err = json.Marshal(&idleProofRecord)
				if err != nil {
					n.Ichal("err", err.Error())
				} else {
					err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "idleproof"))
					if err != nil {
						n.Ichal("err", err.Error())
					}
				}

				spaceProofVerifyTotal, err := n.PoisRequestVerifySpaceTotalP2P(teeAddrInfo.ID, n.GetSignatureAccPulickey(), blocksProof, idleProofRecord.ChainFront, idleProofRecord.ChainRear, idleProofRecord.Acc, idleProofRecord.ChallRandom, time.Duration(time.Minute*10))
				if err != nil {
					n.Ichal("err", fmt.Sprintf("[PoisRequestVerifySpaceTotalP2P] %v", err))
					return
				}

				var teeSignature pattern.TeeSignature
				if len(spaceProofVerifyTotal.Signature) != len(teeSignature) {
					n.Ichal("err", "invalid spaceProofVerifyTotal signature")
					return
				}

				for i := 0; i < len(spaceProofVerifyTotal.Signature); i++ {
					teeSignature[i] = types.U8(spaceProofVerifyTotal.Signature[i])
				}

				var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
				for i := 0; i < len(idleProofRecord.IdleProof); i++ {
					idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
				}
				txHash, err := n.SubmitIdleProofResult(
					idleProve,
					types.U64(idleProofRecord.ChainFront),
					types.U64(idleProofRecord.ChainRear),
					minerChalInfo.SpaceProofInfo.Accumulator,
					types.Bool(spaceProofVerifyTotal.IdleResult),
					teeSignature,
					idleProofRecord.AllocatedTeeAccountId,
				)
				if err != nil {
					n.Ichal("err", fmt.Sprintf("[SubmitIdleProofResult] hash: %s, err: %v", txHash, err))
					return
				}
				n.Ichal("info", fmt.Sprintf("submit idle proof result suc: %s", txHash))
				return
			}
		} else {
			os.Remove(filepath.Join(n.Workspace(), "idleproof"))
		}
	}
	n.Ichal("info", fmt.Sprintf("Have a new idle challenge: %v", challenge.NetSnapShot.Start))
	idleProofRecord = idleProofInfo{}

	idleProofRecord.Start = int32(challenge.NetSnapShot.Start)
	idleProofRecord.ChainFront = int64(minerChalInfo.SpaceProofInfo.Front)
	idleProofRecord.ChainRear = int64(minerChalInfo.SpaceProofInfo.Rear)

	var acc = make([]byte, len(pattern.Accumulator{}))
	for i := 0; i < len(acc); i++ {
		acc[i] = byte(minerChalInfo.SpaceProofInfo.Accumulator[i])
	}

	idleProofRecord.Acc = acc

	err = n.Prover.SetChallengeState(*n.Pois.RsaKey, acc, int64(minerChalInfo.SpaceProofInfo.Front), int64(minerChalInfo.SpaceProofInfo.Rear))
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[SetChallengeState] %v", err))
		return
	}

	var challRandom = make([]int64, len(challenge.NetSnapShot.SpaceChallengeParam))
	for i := 0; i < len(challRandom); i++ {
		challRandom[i] = int64(challenge.NetSnapShot.SpaceChallengeParam[i])
	}

	idleProofRecord.ChallRandom = challRandom

	var rear int64
	var blocksProof = make([]*pb.BlocksProof, 0)

	n.Ichal("info", "start calc challenge...")
	idleProofRecord.FileBlockProofInfo = make([]fileBlockProofInfo, 0)
	var idleproof []byte

	for front := (minerChalInfo.SpaceProofInfo.Front + 1); front <= (minerChalInfo.SpaceProofInfo.Rear + 1); {
		var fileBlockProofInfoEle fileBlockProofInfo
		if (front + 256) > (minerChalInfo.SpaceProofInfo.Rear + 1) {
			rear = int64(minerChalInfo.SpaceProofInfo.Rear + 1)
		} else {
			rear = int64(front + 256)
		}
		fileBlockProofInfoEle.FileBlockFront = int64(front)
		fileBlockProofInfoEle.FileBlockRear = rear
		spaceProof, err := n.Prover.ProveSpace(challRandom, int64(front), rear)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("[ProveSpace] %v", err))
			return
		}

		var mhtProofGroup = make([]*pb.MhtProofGroup, len(spaceProof.Proofs))

		for i := 0; i < len(spaceProof.Proofs); i++ {
			mhtProofGroup[i] = &pb.MhtProofGroup{}
			mhtProofGroup[i].Proofs = make([]*pb.MhtProof, len(spaceProof.Proofs[i]))
			for j := 0; j < len(spaceProof.Proofs[i]); j++ {
				mhtProofGroup[i].Proofs[j] = &pb.MhtProof{
					Index: int32(spaceProof.Proofs[i][j].Index),
					Label: spaceProof.Proofs[i][j].Label,
					Paths: spaceProof.Proofs[i][j].Paths,
					Locs:  spaceProof.Proofs[i][j].Locs,
				}
			}
		}

		var witChains = make([]*pb.AccWitnessNode, len(spaceProof.WitChains))

		for i := 0; i < len(spaceProof.WitChains); i++ {
			witChains[i] = &pb.AccWitnessNode{
				Elem: spaceProof.WitChains[i].Elem,
				Wit:  spaceProof.WitChains[i].Wit,
				Acc: &pb.AccWitnessNode{
					Elem: spaceProof.WitChains[i].Acc.Elem,
					Wit:  spaceProof.WitChains[i].Acc.Wit,
					Acc: &pb.AccWitnessNode{
						Elem: spaceProof.WitChains[i].Acc.Acc.Elem,
						Wit:  spaceProof.WitChains[i].Acc.Acc.Wit,
						Acc: &pb.AccWitnessNode{
							Elem: spaceProof.WitChains[i].Acc.Acc.Acc.Elem,
							Wit:  spaceProof.WitChains[i].Acc.Acc.Acc.Wit,
						},
					},
				},
			}
		}

		var proof = &pb.SpaceProof{
			Left:      spaceProof.Left,
			Right:     spaceProof.Right,
			Roots:     spaceProof.Roots,
			Proofs:    mhtProofGroup,
			WitChains: witChains,
		}

		fileBlockProofInfoEle.SpaceProof = proof

		b, err := proto.Marshal(proof)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("[proto.Marshal] %v", err))
			return
		}

		h := sha256.New()
		_, err = h.Write(b)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("[h.Write] %v", err))
			return
		}
		proofHash := h.Sum(nil)

		fileBlockProofInfoEle.ProofHashSignOrigin = proofHash
		idleproof = append(idleproof, proofHash...)
		sign, err := n.Sign(proofHash)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("[n.Sign] %v", err))
			return
		}

		fileBlockProofInfoEle.ProofHashSign = sign
		idleProofRecord.FileBlockProofInfo = append(idleProofRecord.FileBlockProofInfo, fileBlockProofInfoEle)
		if types.U64(rear) >= (minerChalInfo.SpaceProofInfo.Rear + 1) {
			break
		}
		front += 256
	}

	h := sha256.New()
	_, err = h.Write(idleproof)
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[h.Write] %v", err))
		return
	}
	idleProofRecord.IdleProof = h.Sum(nil)

	var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
	for i := 0; i < len(idleProofRecord.IdleProof); i++ {
		idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
	}

	buf, err = json.Marshal(&idleProofRecord)
	if err != nil {
		n.Ichal("err", err.Error())
	} else {
		err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "idleproof"))
		if err != nil {
			n.Ichal("err", err.Error())
		}
	}

	//
	txhash, err := n.SubmitIdleProof(idleProve)
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[SubmitIdleProof] %v", err))
		return
	}
	n.Ichal("info", fmt.Sprintf("SubmitIdleProof: %s", txhash))

	time.Sleep(pattern.BlockInterval)
	time.Sleep(pattern.BlockInterval)

	teeAccounts := n.GetAllTeeWorkAccount()
	var teePeerIdPubkey []byte
	found = false
	for _, v := range teeAccounts {
		if found {
			break
		}
		publickey, _ := sutils.ParsingPublickey(v)
		idleProofInfos, err := n.QueryUnverifiedIdleProof(publickey)
		if err != nil {
			continue
		}

		for i := 0; i < len(idleProofInfos); i++ {
			if sutils.CompareSlice(idleProofInfos[i].MinerSnapShot.Miner[:], n.GetSignatureAccPulickey()) {
				idleProofRecord.AllocatedTeeAccount = v
				idleProofRecord.AllocatedTeeAccountId = publickey
				found = true
				break
			}
		}
	}
	if !found {
		n.Ichal("err", "Not found allocated tee for idle proof")
		return
	}

	teePeerIdPubkey, _ = n.GetTeeWork(idleProofRecord.AllocatedTeeAccount)

	teeAddrInfo, ok := n.GetPeer(base58.Encode(teePeerIdPubkey))
	if !ok {
		n.Ichal("err", fmt.Sprintf("Not found peer: %s", base58.Encode(teePeerIdPubkey)))
		return
	}

	err = n.Connect(n.GetCtxQueryFromCtxCancel(), teeAddrInfo)
	if err != nil {
		n.Ichal("err", fmt.Sprintf("Connect tee peer err: %v", err))
	}

	n.Ichal("info", fmt.Sprintf("PoisSpaceProofVerifySingleBlockP2P to tee: %s", teeAddrInfo.ID.Pretty()))

	for i := 0; i < len(idleProofRecord.FileBlockProofInfo); i++ {
		spaceProofVerify, err := n.PoisSpaceProofVerifySingleBlockP2P(
			teeAddrInfo.ID,
			n.GetSignatureAccPulickey(),
			idleProofRecord.ChallRandom,
			n.Pois.RsaKey.N.Bytes(),
			n.Pois.RsaKey.G.Bytes(),
			idleProofRecord.Acc,
			int64(minerChalInfo.SpaceProofInfo.Front),
			int64(minerChalInfo.SpaceProofInfo.Rear),
			idleProofRecord.FileBlockProofInfo[i].SpaceProof,
			idleProofRecord.FileBlockProofInfo[i].ProofHashSign,
			time.Duration(time.Minute),
		)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("[PoisSpaceProofVerifySingleBlockP2P] %v", err))
			return
		}
		var block = &pb.BlocksProof{
			ProofHashAndLeftRight: &pb.ProofHashAndLeftRight{
				SpaceProofHash: idleProofRecord.FileBlockProofInfo[i].ProofHashSignOrigin,
				Left:           idleProofRecord.FileBlockProofInfo[i].FileBlockFront,
				Right:          idleProofRecord.FileBlockProofInfo[i].FileBlockRear,
			},
			Signature: spaceProofVerify.Signature,
		}
		blocksProof = append(blocksProof, block)
	}

	idleProofRecord.BlocksProof = blocksProof
	buf, err = json.Marshal(&idleProofRecord)
	if err != nil {
		n.Ichal("err", err.Error())
	} else {
		err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "idleproof"))
		if err != nil {
			n.Ichal("err", err.Error())
		}
	}

	spaceProofVerifyTotal, err := n.PoisRequestVerifySpaceTotalP2P(teeAddrInfo.ID, n.GetSignatureAccPulickey(), blocksProof, int64(minerChalInfo.SpaceProofInfo.Front), int64(minerChalInfo.SpaceProofInfo.Rear), acc, challRandom, time.Duration(time.Minute*3))
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[PoisRequestVerifySpaceTotalP2P] %v", err))
		return
	}

	n.Ichal("info", fmt.Sprintf("spaceProofVerifyTotal.IdleResult is %v", spaceProofVerifyTotal.IdleResult))

	var teeSignature pattern.TeeSignature
	if len(spaceProofVerifyTotal.Signature) != len(teeSignature) {
		n.Ichal("err", "invalid spaceProofVerifyTotal signature")
		return
	}

	for i := 0; i < len(spaceProofVerifyTotal.Signature); i++ {
		teeSignature[i] = types.U8(spaceProofVerifyTotal.Signature[i])
	}

	idleProofRecord.TotalSignature = spaceProofVerifyTotal.Signature
	idleProofRecord.IdleResult = spaceProofVerifyTotal.IdleResult
	buf, err = json.Marshal(&idleProofRecord)
	if err != nil {
		n.Ichal("err", err.Error())
	} else {
		err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "idleproof"))
		if err != nil {
			n.Ichal("err", err.Error())
		}
	}

	txHash, err := n.SubmitIdleProofResult(
		idleProve,
		types.U64(idleProofRecord.ChainFront),
		types.U64(idleProofRecord.ChainRear),
		minerChalInfo.SpaceProofInfo.Accumulator,
		types.Bool(spaceProofVerifyTotal.IdleResult),
		teeSignature,
		idleProofRecord.AllocatedTeeAccountId,
	)
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[SubmitIdleProofResult] hash: %s, err: %v", txHash, err))
		return
	}

	n.Ichal("info", fmt.Sprintf("submit idle proof result suc: %s", txHash))
	return
}

func (n *Node) poisChallengeResult(ch chan<- bool, latestBlock, challVerifyExpiration uint32, idleChallTeeAcc string, challenge pattern.ChallengeInfo_V2, minerChalInfo pattern.MinerSnapShot_V2) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	if challVerifyExpiration <= latestBlock {
		return
	}

	var idleProofRecord idleProofInfo
	buf, err := os.ReadFile(filepath.Join(n.Workspace(), "idleproof"))
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[ReadFile(idleproof)] %v", err))
		return
	}

	err = json.Unmarshal(buf, &idleProofRecord)
	if err != nil {
		os.Remove(filepath.Join(n.Workspace(), "idleproof"))
		n.Ichal("err", fmt.Sprintf("[Unmarshal] %v", err))
		return
	}

	n.Ichal("info", fmt.Sprintf("chain challenge: %v, local idleproof file challenge: %v", challenge.NetSnapShot.Start, idleProofRecord.Start))

	if idleProofRecord.Start != int32(challenge.NetSnapShot.Start) {
		os.Remove(filepath.Join(n.Workspace(), "idleproof"))
		return
	}

	allocatedTeeAccountId, err := sutils.ParsingPublickey(idleChallTeeAcc)
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[ParsingPublickey(%s)] %v", idleChallTeeAcc, err))
		return
	}

	if idleProofRecord.TotalSignature != nil {
		var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
		for i := 0; i < len(idleProofRecord.IdleProof); i++ {
			idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
		}
		var teeSignature pattern.TeeSignature
		if len(idleProofRecord.TotalSignature) != len(teeSignature) {
			n.Ichal("err", "invalid spaceProofVerifyTotal signature")
			return
		}

		for i := 0; i < len(idleProofRecord.TotalSignature); i++ {
			teeSignature[i] = types.U8(idleProofRecord.TotalSignature[i])
		}

		txHash, err := n.SubmitIdleProofResult(
			idleProve,
			types.U64(idleProofRecord.ChainFront),
			types.U64(idleProofRecord.ChainRear),
			minerChalInfo.SpaceProofInfo.Accumulator,
			types.Bool(idleProofRecord.IdleResult),
			teeSignature,
			allocatedTeeAccountId,
		)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("[SubmitIdleProofResult] hash: %s, err: %v", txHash, err))
			return
		}
		n.Ichal("info", fmt.Sprintf("submit idle proof result suc: %s", txHash))
		return
	}
	if idleProofRecord.BlocksProof != nil {
		teePeerIdPubkey, _ := n.GetTeeWork(idleChallTeeAcc)

		teeAddrInfo, ok := n.GetPeer(base58.Encode(teePeerIdPubkey))
		if !ok {
			n.Ichal("err", fmt.Sprintf("Not found peer: %s", base58.Encode(teePeerIdPubkey)))
			return
		}

		err = n.Connect(n.GetCtxQueryFromCtxCancel(), teeAddrInfo)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("Connect tee peer err: %v", err))
		}
		spaceProofVerifyTotal, err := n.PoisRequestVerifySpaceTotalP2P(
			teeAddrInfo.ID,
			n.GetSignatureAccPulickey(),
			idleProofRecord.BlocksProof,
			idleProofRecord.ChainFront,
			idleProofRecord.ChainRear,
			idleProofRecord.Acc,
			idleProofRecord.ChallRandom,
			time.Duration(time.Minute*10),
		)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("[PoisRequestVerifySpaceTotalP2P] %v", err))
			return
		}
		idleProofRecord.TotalSignature = spaceProofVerifyTotal.Signature
		idleProofRecord.IdleResult = spaceProofVerifyTotal.IdleResult
		buf, err = json.Marshal(&idleProofRecord)
		if err != nil {
			n.Ichal("err", err.Error())
		} else {
			err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "idleproof"))
			if err != nil {
				n.Ichal("err", err.Error())
			}
		}
		var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
		for i := 0; i < len(idleProofRecord.IdleProof); i++ {
			idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
		}
		var teeSignature pattern.TeeSignature
		if len(idleProofRecord.TotalSignature) != len(teeSignature) {
			n.Ichal("err", "invalid spaceProofVerifyTotal signature")
			return
		}

		for i := 0; i < len(idleProofRecord.TotalSignature); i++ {
			teeSignature[i] = types.U8(idleProofRecord.TotalSignature[i])
		}

		txHash, err := n.SubmitIdleProofResult(
			idleProve,
			types.U64(idleProofRecord.ChainFront),
			types.U64(idleProofRecord.ChainRear),
			minerChalInfo.SpaceProofInfo.Accumulator,
			types.Bool(idleProofRecord.IdleResult),
			teeSignature,
			allocatedTeeAccountId,
		)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("[SubmitIdleProofResult] hash: %s, err: %v", txHash, err))
			return
		}
		n.Ichal("info", fmt.Sprintf("SubmitIdleProofResult: %s", txHash))
		return
	}

	teePeerIdPubkey, _ := n.GetTeeWork(idleChallTeeAcc)

	teeAddrInfo, ok := n.GetPeer(base58.Encode(teePeerIdPubkey))
	if !ok {
		n.Ichal("err", fmt.Sprintf("Not found peer: %s", base58.Encode(teePeerIdPubkey)))
		return
	}

	err = n.Connect(n.GetCtxQueryFromCtxCancel(), teeAddrInfo)
	if err != nil {
		n.Ichal("err", fmt.Sprintf("Connect tee peer err: %v", err))
	}
	var blocksProof = make([]*pb.BlocksProof, 0)
	for i := 0; i < len(idleProofRecord.FileBlockProofInfo); i++ {
		spaceProofVerify, err := n.PoisSpaceProofVerifySingleBlockP2P(
			teeAddrInfo.ID,
			n.GetSignatureAccPulickey(),
			idleProofRecord.ChallRandom,
			n.Pois.RsaKey.N.Bytes(),
			n.Pois.RsaKey.G.Bytes(),
			idleProofRecord.Acc,
			int64(minerChalInfo.SpaceProofInfo.Front),
			int64(minerChalInfo.SpaceProofInfo.Rear),
			idleProofRecord.FileBlockProofInfo[i].SpaceProof,
			idleProofRecord.FileBlockProofInfo[i].ProofHashSign,
			time.Duration(time.Minute*3),
		)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("[PoisSpaceProofVerifySingleBlockP2P] %v", err))
			return
		}
		var block = &pb.BlocksProof{
			ProofHashAndLeftRight: &pb.ProofHashAndLeftRight{
				SpaceProofHash: idleProofRecord.FileBlockProofInfo[i].ProofHashSignOrigin,
				Left:           idleProofRecord.FileBlockProofInfo[i].FileBlockFront,
				Right:          idleProofRecord.FileBlockProofInfo[i].FileBlockRear,
			},
			Signature: spaceProofVerify.Signature,
		}
		blocksProof = append(blocksProof, block)
	}

	idleProofRecord.BlocksProof = blocksProof
	buf, err = json.Marshal(&idleProofRecord)
	if err != nil {
		n.Ichal("err", err.Error())
	} else {
		err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "idleproof"))
		if err != nil {
			n.Ichal("err", err.Error())
		}
	}

	spaceProofVerifyTotal, err := n.PoisRequestVerifySpaceTotalP2P(
		teeAddrInfo.ID,
		n.GetSignatureAccPulickey(),
		blocksProof,
		idleProofRecord.ChainFront,
		idleProofRecord.ChainRear,
		idleProofRecord.Acc,
		idleProofRecord.ChallRandom,
		time.Duration(time.Minute*10),
	)
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[PoisRequestVerifySpaceTotalP2P] %v", err))
		return
	}

	var teeSignature pattern.TeeSignature
	if len(spaceProofVerifyTotal.Signature) != len(teeSignature) {
		n.Ichal("err", "invalid spaceProofVerifyTotal signature")
		return
	}

	for i := 0; i < len(spaceProofVerifyTotal.Signature); i++ {
		teeSignature[i] = types.U8(spaceProofVerifyTotal.Signature[i])
	}

	var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
	for i := 0; i < len(idleProofRecord.IdleProof); i++ {
		idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
	}

	txHash, err := n.SubmitIdleProofResult(
		idleProve,
		types.U64(idleProofRecord.ChainFront),
		types.U64(idleProofRecord.ChainRear),
		minerChalInfo.SpaceProofInfo.Accumulator,
		types.Bool(spaceProofVerifyTotal.IdleResult),
		teeSignature,
		allocatedTeeAccountId,
	)
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[SubmitIdleProofResult] hash: %s, err: %v", txHash, err))
		return
	}
	n.Ichal("info", fmt.Sprintf("submit idle proof result suc: %s", txHash))
	return
}
