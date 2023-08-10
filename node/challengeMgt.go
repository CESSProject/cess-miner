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
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/proof"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
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
	Start              int32   `json:"start"`
	ChainFront         int64   `json:"chainFront"`
	ChainRear          int64   `json:"chainRear"`
	IdleResult         bool    `json:"idleResult"`
	IdleProof          []byte  `json:"idleProof"`
	Acc                []byte  `json:"acc"`
	TotalSignature     []byte  `json:"totalSignature"`
	ChallRandom        []int64 `json:"challRandom"`
	FileBlockProofInfo []fileBlockProofInfo
	BlocksProof        []*pb.BlocksProof
}

type serviceProofInfo struct {
	Names              []string `json:"names"`
	Us                 []string `json:"us"`
	Mus                []string `json:"mus"`
	ServiceBloomFilter []uint64 `json:"serviceBloomFilter"`
	TeePeerId          []byte   `json:"teePeerId"`
	Signature          []byte   `json:"signature"`
	Sigma              string   `json:"sigma"`
	Start              uint32   `json:"start"`
	ServiceResult      bool     `json:"serviceResult"`
}

func (n *Node) poisChallenge() error {
	defer func() {
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	challenge, err := n.QueryChallenge_V2()
	if err != nil {
		if err.Error() != pattern.ERR_Empty {
			return errors.Wrapf(err, "[QueryChallenge]")
		}
		return nil
	}

	var haveChallenge bool
	var minerChalInfo pattern.MinerSnapShot_V2
	for _, v := range challenge.MinerSnapshotList {
		if sutils.CompareSlice(n.GetSignatureAccPulickey(), v.Miner[:]) {
			haveChallenge = true
			minerChalInfo = v
			break
		}
	}

	latestBlock, err := n.QueryBlockHeight("")
	if err != nil {
		return errors.Wrapf(err, "[QueryBlockHeight]")
	}

	challExpiration, err := n.QueryChallengeExpiration()
	if err != nil {
		return errors.Wrapf(err, "[QueryChallengeExpiration]")
	}

	if challExpiration <= latestBlock {
		haveChallenge = false
	}

	if !haveChallenge {
		n.Chal("info", "no challenge")
		return nil
	}

	n.Chal("info", fmt.Sprintf("Have a new Challenge: %v", challenge.NetSnapShot.Start))

	var idleProofRecord idleProofInfo
	//keypair, _ := signature.KeyringPairFromSecret("tray fine poem nothing glimpse insane carbon empty grief dismiss bird nurse", 0)
	keypair, err := signature.KeyringPairFromSecret(os.Args[3], 0)
	if err != nil {
		n.Chal("err", fmt.Sprintf("err tee mnemonic: %v", os.Args[3]))
		return nil
	}
	buf, err := os.ReadFile(filepath.Join(n.Workspace(), "idleproof"))
	if err == nil {
		err = json.Unmarshal(buf, &idleProofRecord)
		if err == nil {
			n.Chal("info", fmt.Sprintf("local idleproof file challenge: %v", idleProofRecord.Start))
			if idleProofRecord.Start != int32(challenge.NetSnapShot.Start) {
				os.Remove(filepath.Join(n.Workspace(), "idleproof"))
			} else {
				idleProofInfos, err := n.QueryUnverifiedIdleProof(keypair.PublicKey)
				if err != nil {
					if err.Error() != pattern.ERR_Empty {
						return errors.Wrapf(err, "[QueryUnverifiedIdleProof]")
					}
					var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
					for i := 0; i < len(idleProofRecord.IdleProof); i++ {
						idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
					}
					_, err = n.SubmitIdleProof(idleProve)
					if err != nil {
						return errors.Wrapf(err, "[SubmitIdleProof]")
					}
				} else {
					var have bool
					for i := 0; i < len(idleProofInfos); i++ {
						if sutils.CompareSlice(idleProofInfos[i].MinerSnapShot.Miner[:], n.GetSignatureAccPulickey()) {
							have = true
							n.Chal("info", "already submit idle proof")
							break
						}
					}
					if have {
						if idleProofRecord.TotalSignature != nil {
							var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
							for i := 0; i < len(idleProofRecord.IdleProof); i++ {
								idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
							}
							var teeSignature pattern.TeeSignature
							if len(idleProofRecord.TotalSignature) != len(teeSignature) {
								return errors.New("invalid spaceProofVerifyTotal signature")
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
								keypair.PublicKey,
							)
							if err != nil {
								return errors.Wrapf(err, "[SubmitIdleProofResult]")
							}
							n.Chal("info", fmt.Sprintf("submit idle proof result suc: %s", txHash))
							return nil
						}
						if idleProofRecord.BlocksProof != nil {
							spaceProofVerifyTotal, err := n.PoisRequestVerifySpaceTotal(os.Args[2], n.GetSignatureAccPulickey(), idleProofRecord.BlocksProof, idleProofRecord.ChainFront, idleProofRecord.ChainRear, idleProofRecord.Acc, idleProofRecord.ChallRandom, time.Duration(time.Minute*3))
							if err != nil {
								return errors.Wrapf(err, "[PoisRequestVerifySpaceTotal]")
							}
							idleProofRecord.TotalSignature = spaceProofVerifyTotal.Signature
							idleProofRecord.IdleResult = spaceProofVerifyTotal.IdleResult
							buf, err = json.Marshal(&idleProofRecord)
							if err != nil {
								n.Chal("err", err.Error())
							} else {
								err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "idleproof"))
								if err != nil {
									n.Chal("err", err.Error())
								}
							}
							var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
							for i := 0; i < len(idleProofRecord.IdleProof); i++ {
								idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
							}
							var teeSignature pattern.TeeSignature
							if len(idleProofRecord.TotalSignature) != len(teeSignature) {
								return errors.New("invalid spaceProofVerifyTotal signature")
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
								keypair.PublicKey,
							)
							if err != nil {
								return errors.Wrapf(err, "[SubmitIdleProofResult]")
							}
							n.Chal("info", fmt.Sprintf("SubmitIdleProofResult: %s", txHash))
							return nil
						}
					} else {
						if minerChalInfo.IdleSubmitted {
							n.Chal("info", "idle proof already submitted and verified")
							return nil
						}
						var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
						for i := 0; i < len(idleProofRecord.IdleProof); i++ {
							idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
						}
						_, err = n.SubmitIdleProof(idleProve)
						if err != nil {
							return errors.Wrapf(err, "[SubmitIdleProof]")
						}
					}
				}
				//TODO: wait assigned tee
				var blocksProof = make([]*pb.BlocksProof, 0)
				for i := 0; i < len(idleProofRecord.FileBlockProofInfo); i++ {
					spaceProofVerify, err := n.PoisSpaceProofVerifySingleBlock(
						os.Args[2],
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
						return errors.Wrapf(err, "[PoisSpaceProofVerifySingleBlock]")
					}
					var block = &pb.BlocksProof{
						ProofHashAndLeftRight: &pb.ProofHashAndLeftRight{
							SpaceProofHash: idleProofRecord.FileBlockProofInfo[i].ProofHashSign,
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
					n.Chal("err", err.Error())
				} else {
					err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "idleproof"))
					if err != nil {
						n.Chal("err", err.Error())
					}
				}

				spaceProofVerifyTotal, err := n.PoisRequestVerifySpaceTotal(os.Args[2], n.GetSignatureAccPulickey(), blocksProof, idleProofRecord.ChainFront, idleProofRecord.ChainRear, idleProofRecord.Acc, idleProofRecord.ChallRandom, time.Duration(time.Minute*3))
				if err != nil {
					return errors.Wrapf(err, "[PoisRequestVerifySpaceTotal]")
				}

				var teeSignature pattern.TeeSignature
				if len(spaceProofVerifyTotal.Signature) != len(teeSignature) {
					return errors.New("invalid spaceProofVerifyTotal signature")
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
					keypair.PublicKey,
				)
				if err != nil {
					return errors.Wrapf(err, "[SubmitIdleProofResult]")
				}
				n.Chal("info", fmt.Sprintf("submit idle proof result suc: %s", txHash))
				return nil
			}
		} else {
			os.Remove(filepath.Join(n.Workspace(), "idleproof"))
		}
	}
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
		n.Chal("err", "[SetChallengeState]")
		return err
	}

	var challRandom = make([]int64, len(challenge.NetSnapShot.SpaceChallengeParam))
	for i := 0; i < len(challRandom); i++ {
		challRandom[i] = int64(challenge.NetSnapShot.SpaceChallengeParam[i])
	}

	idleProofRecord.ChallRandom = challRandom

	var rear int64
	var blocksProof = make([]*pb.BlocksProof, 0)

	n.Chal("info", "start calc challenge...")
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
		n.Chal("info", fmt.Sprintf("front: %d  rear: %d", front, rear))
		spaceProof, err := n.Prover.ProveSpace(challRandom, int64(front), rear)
		if err != nil {
			return errors.Wrapf(err, "[ProveSpace]")
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
			return errors.Wrapf(err, "[proto.Marshal]")
		}

		h := sha256.New()
		_, err = h.Write(b)
		if err != nil {
			return errors.Wrapf(err, "[h.Write]")
		}
		proofHash := h.Sum(nil)

		fileBlockProofInfoEle.ProofHashSignOrigin = proofHash
		idleproof = append(idleproof, proofHash...)
		sign, err := n.Sign(proofHash)
		if err != nil {
			return errors.Wrapf(err, "[n.Sign]")
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
		return errors.Wrapf(err, "[h.Write]")
	}
	idleProofRecord.IdleProof = h.Sum(nil)

	var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
	for i := 0; i < len(idleProofRecord.IdleProof); i++ {
		idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
	}

	//
	txhash, err := n.SubmitIdleProof(idleProve)
	if err != nil {
		n.Chal("err", fmt.Sprintf("[SubmitIdleProof] %v", err))
		buf, err := json.Marshal(&idleProofRecord)
		if err != nil {
			n.Chal("err", err.Error())
		} else {
			err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "idleproof"))
			if err != nil {
				n.Chal("err", err.Error())
			}
		}
		return errors.Wrapf(err, "[SubmitIdleProof]")
	}

	n.Chal("info", fmt.Sprintf("SubmitIdleProof: %s", txhash))

	buf, err = json.Marshal(&idleProofRecord)
	if err != nil {
		n.Chal("err", err.Error())
	} else {
		err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "idleproof"))
		if err != nil {
			n.Chal("err", err.Error())
		}
	}

	// TODO: wait for assigned tee

	for i := 0; i < len(idleProofRecord.FileBlockProofInfo); i++ {
		spaceProofVerify, err := n.PoisSpaceProofVerifySingleBlock(
			os.Args[2],
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
			return errors.Wrapf(err, "[PoisSpaceProofVerifySingleBlock]")
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
		n.Chal("err", err.Error())
	} else {
		err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "idleproof"))
		if err != nil {
			n.Chal("err", err.Error())
		}
	}

	spaceProofVerifyTotal, err := n.PoisRequestVerifySpaceTotal(os.Args[2], n.GetSignatureAccPulickey(), blocksProof, int64(minerChalInfo.SpaceProofInfo.Front), int64(minerChalInfo.SpaceProofInfo.Rear), acc, challRandom, time.Duration(time.Minute*3))
	if err != nil {
		return errors.Wrapf(err, "[PoisRequestVerifySpaceTotal]")
	}

	if !spaceProofVerifyTotal.IdleResult {
		n.Chal("err", "spaceProofVerifyTotal.IdleResult is false")
	}

	var teeSignature pattern.TeeSignature
	if len(spaceProofVerifyTotal.Signature) != len(teeSignature) {
		return errors.New("invalid spaceProofVerifyTotal signature")
	}

	for i := 0; i < len(spaceProofVerifyTotal.Signature); i++ {
		teeSignature[i] = types.U8(spaceProofVerifyTotal.Signature[i])
	}

	idleProofRecord.TotalSignature = spaceProofVerifyTotal.Signature
	idleProofRecord.IdleResult = spaceProofVerifyTotal.IdleResult
	buf, err = json.Marshal(&idleProofRecord)
	if err != nil {
		n.Chal("err", err.Error())
	} else {
		err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "idleproof"))
		if err != nil {
			n.Chal("err", err.Error())
		}
	}

	txHash, err := n.SubmitIdleProofResult(
		idleProve,
		types.U64(idleProofRecord.ChainFront),
		types.U64(idleProofRecord.ChainRear),
		minerChalInfo.SpaceProofInfo.Accumulator,
		types.Bool(spaceProofVerifyTotal.IdleResult),
		teeSignature,
		keypair.PublicKey,
	)
	if err != nil {
		return errors.Wrapf(err, "[SubmitIdleProofResult]")
	}

	n.Chal("info", fmt.Sprintf("submit idle proof result suc: %s", txHash))
	return nil
}

func (n *Node) poisServiceChallenge() error {
	defer func() {
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	challenge, err := n.QueryChallenge_V2()
	if err != nil {
		if err.Error() != pattern.ERR_Empty {
			return errors.Wrapf(err, "[QueryChallenge]")
		}
		return nil
	}

	var haveChallenge bool
	var minerChalInfo pattern.MinerSnapShot_V2
	for _, v := range challenge.MinerSnapshotList {
		if sutils.CompareSlice(n.GetSignatureAccPulickey(), v.Miner[:]) {
			haveChallenge = true
			minerChalInfo = v
			break
		}
	}

	latestBlock, err := n.QueryBlockHeight("")
	if err != nil {
		return errors.Wrapf(err, "[QueryBlockHeight]")
	}

	challExpiration, err := n.QueryChallengeExpiration()
	if err != nil {
		return errors.Wrapf(err, "[QueryChallengeExpiration]")
	}

	if challExpiration <= latestBlock {
		haveChallenge = false
	}

	if !haveChallenge {
		n.Chal("info", "no challenge")
		return nil
	}

	var qslice = make([]proof.QElement, len(challenge.NetSnapShot.RandomIndexList))
	for k, v := range challenge.NetSnapShot.RandomIndexList {
		qslice[k].I = int64(v)
		var b = make([]byte, len(challenge.NetSnapShot.RandomList[k]))
		for i := 0; i < len(challenge.NetSnapShot.RandomList[k]); i++ {
			b[i] = byte(challenge.NetSnapShot.RandomList[k][i])
		}
		qslice[k].V = new(big.Int).SetBytes(b).String()
	}

	n.Chal("info", fmt.Sprintf("Have a new service Challenge: %v", challenge.NetSnapShot.Start))

	var serviceProofRecord serviceProofInfo

	// keypair, _ := signature.KeyringPairFromSecret("tray fine poem nothing glimpse insane carbon empty grief dismiss bird nurse", 0)
	keypair, err := signature.KeyringPairFromSecret(os.Args[3], 0)
	if err != nil {
		n.Chal("err", fmt.Sprintf("err tee mnemonic: %v", os.Args[3]))
		return nil
	}
	buf, err := os.ReadFile(filepath.Join(n.Workspace(), "serviceproof"))
	if err == nil {
		err = json.Unmarshal(buf, &serviceProofRecord)
		if err == nil {
			n.Chal("info", fmt.Sprintf("local service proof file challenge: %v", serviceProofRecord.Start))
			if serviceProofRecord.Start != uint32(challenge.NetSnapShot.Start) {
				os.Remove(filepath.Join(n.Workspace(), "serviceproof"))
			} else {
				if !minerChalInfo.ServiceSubmitted {
					var serviceProve = make([]types.U8, len(serviceProofRecord.Sigma))
					for i := 0; i < len(serviceProofRecord.Sigma); i++ {
						serviceProve[i] = types.U8(serviceProofRecord.Sigma[i])
					}
					_, err = n.SubmitServiceProof(serviceProve)
					if err != nil {
						return errors.Wrapf(err, "[SubmitServiceProof]")
					}
				} else {
					if serviceProofRecord.ServiceBloomFilter != nil &&
						serviceProofRecord.TeePeerId != nil &&
						serviceProofRecord.Signature != nil {
						var signature pattern.TeeSignature
						if len(pattern.TeeSignature{}) != len(serviceProofRecord.Signature) {
							return errors.New("invalid batchVerify.Signature")
						}
						for i := 0; i < len(serviceProofRecord.Signature); i++ {
							signature[i] = types.U8(serviceProofRecord.Signature[i])
						}

						var bloomFilter pattern.BloomFilter
						if len(pattern.BloomFilter{}) != len(serviceProofRecord.ServiceBloomFilter) {
							return errors.New("invalid batchVerify.ServiceBloomFilter")
						}
						for i := 0; i < len(serviceProofRecord.ServiceBloomFilter); i++ {
							bloomFilter[i] = types.U64(serviceProofRecord.ServiceBloomFilter[i])
						}

						txhash, err := n.SubmitServiceProofResult(types.Bool(serviceProofRecord.ServiceResult), signature, bloomFilter, keypair.PublicKey)
						if err != nil {
							return errors.Wrapf(err, "[SubmitServiceProofResult]")
						}
						n.Chal("info", fmt.Sprintf("submit service aggr proof result suc: %s", txhash))
					}
				}
				//TODO: wait assigned tee
				peeridSign, err := n.Sign(n.GetPeerPublickey())
				if err != nil {
					return errors.Wrapf(err, "[Sign peerid]")
				}

				var randomIndexList = make([]uint32, len(challenge.NetSnapShot.RandomIndexList))
				for i := 0; i < len(challenge.NetSnapShot.RandomIndexList); i++ {
					randomIndexList[i] = uint32(challenge.NetSnapShot.RandomIndexList[i])
				}
				var randomList = make([][]byte, len(challenge.NetSnapShot.RandomList))
				for i := 0; i < len(challenge.NetSnapShot.RandomList); i++ {
					randomList[i] = make([]byte, len(challenge.NetSnapShot.RandomList[i]))
					for j := 0; j < len(challenge.NetSnapShot.RandomList[i]); j++ {
						randomList[i][j] = byte(challenge.NetSnapShot.RandomList[i][j])
					}
				}

				var qslice_pb = &pb.RequestBatchVerify_Qslice{
					RandomIndexList: randomIndexList,
					RandomList:      randomList,
				}

				batchVerify, err := n.PoisServiceRequestBatchVerify(
					os.Args[2],
					serviceProofRecord.Names,
					serviceProofRecord.Us,
					serviceProofRecord.Mus,
					serviceProofRecord.Sigma,
					n.GetPeerPublickey(),
					n.GetSignatureAccPulickey(),
					peeridSign,
					qslice_pb,
					time.Duration(time.Minute*10),
				)
				if err != nil {
					return errors.Wrapf(err, "[PoisServiceRequestBatchVerify]")
				}
				serviceProofRecord.ServiceResult = batchVerify.BatchVerifyResult
				serviceProofRecord.ServiceBloomFilter = batchVerify.ServiceBloomFilter
				serviceProofRecord.TeePeerId = batchVerify.TeePeerId
				serviceProofRecord.Signature = batchVerify.Signature
				buf, err = json.Marshal(&serviceProofRecord)
				if err != nil {
					n.Chal("err", err.Error())
				}
				err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "serviceproof"))
				if err != nil {
					n.Chal("err", err.Error())
				}

				var signature pattern.TeeSignature
				if len(pattern.TeeSignature{}) != len(batchVerify.Signature) {
					return errors.New("invalid batchVerify.Signature")
				}
				for i := 0; i < len(batchVerify.Signature); i++ {
					signature[i] = types.U8(batchVerify.Signature[i])
				}

				var bloomFilter pattern.BloomFilter
				if len(pattern.BloomFilter{}) != len(batchVerify.ServiceBloomFilter) {
					return errors.New("invalid batchVerify.ServiceBloomFilter")
				}
				for i := 0; i < len(batchVerify.ServiceBloomFilter); i++ {
					bloomFilter[i] = types.U64(batchVerify.ServiceBloomFilter[i])
				}

				txhash, err := n.SubmitServiceProofResult(types.Bool(batchVerify.BatchVerifyResult), signature, bloomFilter, keypair.PublicKey)
				if err != nil {
					return errors.Wrapf(err, "[SubmitServiceProofResult]")
				}
				n.Chal("info", fmt.Sprintf("submit service aggr proof result suc: %s", txhash))
				return nil
			}
		} else {
			os.Remove(filepath.Join(n.Workspace(), "serviceproof"))
		}
	}

	serviceProofRecord = serviceProofInfo{}
	serviceProofRecord.Start = uint32(challenge.NetSnapShot.Start)
	serviceRoothashDir, err := utils.Dirs(n.GetDirs().FileDir)
	if err != nil {
		return errors.Wrapf(err, "[Dirs]")
	}

	var sigma string

	var proveResponse proof.GenProofResponse
	serviceProofRecord.Names = make([]string, 0)
	serviceProofRecord.Us = make([]string, 0)
	serviceProofRecord.Mus = make([]string, 0)

	timeout := time.NewTicker(time.Duration(time.Minute))
	defer timeout.Stop()

	for i := int(0); i < len(serviceRoothashDir); i++ {
		files, err := utils.DirFiles(serviceRoothashDir[i], 0)
		if err != nil {
			n.Chal("err", err.Error())
			continue
		}

		for j := 0; j < len(files); j++ {
			serviceTagPath := filepath.Join(n.GetDirs().ServiceTagDir, filepath.Base(files[j])+".tag")
			buf, err = os.ReadFile(serviceTagPath)
			if err != nil {
				n.Chal("err", fmt.Sprintf("Servicetag not found: %v", serviceTagPath))
				continue
			}
			var tag pb.Tag
			err = json.Unmarshal(buf, &tag)
			if err != nil {
				n.Chal("err", fmt.Sprintf("Unmarshal %v err: %v", serviceTagPath, err))
				continue
			}
			matrix, _, err := proof.SplitByN(files[j], int64(len(tag.T.Phi)))
			if err != nil {
				n.Chal("err", fmt.Sprintf("SplitByN %v err: %v", serviceTagPath, err))
				continue
			}

			proveResponseCh := n.key.GenProof(qslice, nil, tag.T.Phi, matrix)
			timeout.Reset(time.Minute)
			select {
			case proveResponse = <-proveResponseCh:
			case <-timeout.C:
				proveResponse.StatueMsg.StatusCode = 0
			}

			if proveResponse.StatueMsg.StatusCode != proof.Success {
				n.Chal("err", fmt.Sprintf("GenProof  err: %d", proveResponse.StatueMsg.StatusCode))
				continue
			}

			sigmaTemp, ok := n.key.AggrAppendProof(sigma, qslice, tag.T.Phi)
			if !ok {
				continue
			}
			sigma = sigmaTemp
			serviceProofRecord.Names = append(serviceProofRecord.Names, tag.T.Name)
			serviceProofRecord.Us = append(serviceProofRecord.Us, tag.T.U)
			serviceProofRecord.Mus = append(serviceProofRecord.Mus, proveResponse.MU)
		}
	}
	serviceProofRecord.Sigma = sigma
	buf, err = json.Marshal(&serviceProofRecord)
	if err != nil {
		return err
	}
	err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "serviceproof"))
	if err != nil {
		n.Chal("err", err.Error())
	}

	var serviceProof = make([]types.U8, len(sigma))
	for i := 0; i < len(sigma); i++ {
		serviceProof[i] = types.U8(sigma[i])
	}
	n.Chal("info", fmt.Sprintf("sigma: %v", sigma))
	n.Chal("info", fmt.Sprintf("serviceProof: %v", serviceProof))
	txhash, err := n.SubmitServiceProof(serviceProof)
	if err != nil {
		return errors.Wrapf(err, "[SubmitServiceProof]")
	}
	n.Chal("info", fmt.Sprintf("submit service aggr proof suc: %s", txhash))

	peeridSign, err := n.Sign(n.GetPeerPublickey())
	if err != nil {
		return errors.Wrapf(err, "[Sign peerid]")
	}

	var randomIndexList = make([]uint32, len(challenge.NetSnapShot.RandomIndexList))
	for i := 0; i < len(challenge.NetSnapShot.RandomIndexList); i++ {
		randomIndexList[i] = uint32(challenge.NetSnapShot.RandomIndexList[i])
	}

	var randomList = make([][]byte, len(challenge.NetSnapShot.RandomList))
	for i := 0; i < len(challenge.NetSnapShot.RandomList); i++ {
		randomList[i] = make([]byte, len(challenge.NetSnapShot.RandomList[i]))
		for j := 0; j < len(challenge.NetSnapShot.RandomList[i]); j++ {
			randomList[i][j] = byte(challenge.NetSnapShot.RandomList[i][j])
		}
	}

	var qslice_pb = &pb.RequestBatchVerify_Qslice{
		RandomIndexList: randomIndexList,
		RandomList:      randomList,
	}
	n.Chal("info", fmt.Sprintf("randomIndexList: %v", randomIndexList))
	n.Chal("info", fmt.Sprintf("randomList: %v", randomList))
	batchVerify, err := n.PoisServiceRequestBatchVerify(
		os.Args[2],
		serviceProofRecord.Names,
		serviceProofRecord.Us,
		serviceProofRecord.Mus,
		sigma,
		n.GetPeerPublickey(),
		n.GetSignatureAccPulickey(),
		peeridSign,
		qslice_pb,
		time.Duration(time.Minute*10),
	)
	if err != nil {
		return errors.Wrapf(err, "[PoisServiceRequestBatchVerify]")
	}
	n.Chal("info", fmt.Sprintf("BatchVerifyResult:%v", batchVerify.BatchVerifyResult))

	serviceProofRecord.ServiceResult = batchVerify.BatchVerifyResult
	serviceProofRecord.ServiceBloomFilter = batchVerify.ServiceBloomFilter
	serviceProofRecord.TeePeerId = batchVerify.TeePeerId
	serviceProofRecord.Signature = batchVerify.Signature
	buf, err = json.Marshal(&serviceProofRecord)
	if err != nil {
		n.Chal("err", err.Error())
	}
	err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "serviceproof"))
	if err != nil {
		n.Chal("err", err.Error())
	}

	var signature pattern.TeeSignature
	if len(pattern.TeeSignature{}) != len(batchVerify.Signature) {
		return errors.New("invalid batchVerify.Signature")
	}
	for i := 0; i < len(batchVerify.Signature); i++ {
		signature[i] = types.U8(batchVerify.Signature[i])
	}

	var bloomFilter pattern.BloomFilter
	if len(pattern.BloomFilter{}) != len(batchVerify.ServiceBloomFilter) {
		return errors.New("invalid batchVerify.ServiceBloomFilter")
	}
	for i := 0; i < len(batchVerify.ServiceBloomFilter); i++ {
		bloomFilter[i] = types.U64(batchVerify.ServiceBloomFilter[i])
	}

	txhash, err = n.SubmitServiceProofResult(types.Bool(batchVerify.BatchVerifyResult), signature, bloomFilter, keypair.PublicKey)
	if err != nil {
		return errors.Wrapf(err, "[SubmitServiceProofResult]")
	}
	n.Chal("info", fmt.Sprintf("submit service aggr proof result suc: %s", txhash))
	return nil
}

// challengeMgr
// func (n *Node) challengeMgt(ch chan<- bool) {
// 	defer func() {
// 		ch <- true
// 		if err := recover(); err != nil {
// 			n.Pnc(utils.RecoverError(err))
// 		}
// 	}()

// 	var err error
// 	var recordErr string

// 	n.Chal("info", ">>>>> start challengeMgt <<<<<")

// 	tick := time.NewTicker(time.Minute)
// 	defer tick.Stop()

// 	for {
// 		select {
// 		case <-tick.C:
// 			if n.GetChainState() {
// 				err = n.pChallenge()
// 				if err != nil {
// 					if recordErr != err.Error() {
// 						n.Chal("err", err.Error())
// 						recordErr = err.Error()
// 					}
// 				}
// 			} else {
// 				if recordErr != pattern.ERR_RPC_CONNECTION.Error() {
// 					n.Chal("err", pattern.ERR_RPC_CONNECTION.Error())
// 					recordErr = pattern.ERR_RPC_CONNECTION.Error()
// 				}
// 			}
// 		}
// 	}
// }

func (n *Node) pChallenge() error {
	var err error
	var haveChallenge bool
	var challenge pattern.ChallengeSnapshot

	challenge, err = n.QueryChallengeSt()
	if err != nil {
		return errors.Wrapf(err, "[QueryChallengeSnapshot]")
	}

	for _, v := range challenge.MinerSnapshot {
		if n.GetSignatureAcc() == v.Miner {
			haveChallenge = true
			break
		}
	}

	latestBlock, err := n.QueryBlockHeight("")
	if err != nil {
		return errors.Wrapf(err, "[QueryBlockHeight]")
	}

	challExpiration, err := n.QueryChallengeExpiration()
	if err != nil {
		return errors.Wrapf(err, "[QueryChallengeExpiration]")
	}

	if challExpiration <= latestBlock {
		haveChallenge = false
	}

	var b []byte
	var tempInt int
	var peerid peer.ID

	if !haveChallenge {
		b, err = n.Get([]byte(Cach_AggrProof_Transfered))
		if err != nil {
			if err == cache.NotFound {
				err = n.transferProof(challenge)
				if err != nil {
					return errors.Wrapf(err, "[transferProof]")
				}
				return nil
			}
		}

		temp := strings.Split(string(b), "_")
		if len(temp) <= 1 {
			n.Delete([]byte(Cach_AggrProof_Transfered))
			err = n.transferProof(challenge)
			if err != nil {
				return errors.Wrapf(err, "[transferProof]")
			}
			return nil
		}

		peerid, _, err = n.queryProofAssignedTee()
		if err != nil {
			return errors.Wrapf(err, "[queryProofAssignedTee]")
		}

		tempInt, err = strconv.Atoi(temp[1])
		if err != nil {
			n.Delete([]byte(Cach_AggrProof_Transfered))
			err = n.transferProof(challenge)
			if err != nil {
				return errors.Wrapf(err, "[transferProof]")
			}
			return nil
		}

		if uint32(tempInt) != challenge.NetSnapshot.Start || peerid.Pretty() != temp[0] {
			err = n.transferProof(challenge)
			if err != nil {
				return errors.Wrapf(err, "[transferProof]")
			}
		}
		return nil
	}

	n.Delete([]byte(Cach_AggrProof_Transfered))

	n.Chal("info", fmt.Sprintf("Start processing challenges: %v", challenge.NetSnapshot.Start))

	var qslice = make([]proof.QElement, len(challenge.NetSnapshot.Random_index_list))
	for k, v := range challenge.NetSnapshot.Random_index_list {
		qslice[k].I = int64(v)
		qslice[k].V = new(big.Int).SetBytes(challenge.NetSnapshot.Random[k]).String()
	}

	err = n.saveRandom(challenge)
	if err != nil {
		n.Chal("err", fmt.Sprintf("Save challenge random err: %v", err))
	}

	n.Chal("info", "Save challenge random suc")

	var idleSiama string
	var serviceSigma string

	b, err = n.Get([]byte(Cach_IdleChallengeBlock))
	if err != nil {
		idleSiama, err = n.idleAggrProof(qslice, challenge.NetSnapshot.Start)
		if err != nil {
			return errors.Wrapf(err, "[idleAggrProof]")
		}
		n.Put([]byte(Cach_prefix_idleSiama), []byte(idleSiama))
		n.Put([]byte(Cach_IdleChallengeBlock), []byte(fmt.Sprintf("%d", challenge.NetSnapshot.Start)))
		n.Chal("info", fmt.Sprintf("Idle data aggregation proof: %s", idleSiama))
	} else {
		tempInt, err = strconv.Atoi(string(b))
		if err != nil {
			n.Delete([]byte(Cach_IdleChallengeBlock))
			idleSiama, err = n.idleAggrProof(qslice, challenge.NetSnapshot.Start)
			if err != nil {
				return errors.Wrapf(err, "[idleAggrProof]")
			}
			n.Put([]byte(Cach_prefix_idleSiama), []byte(idleSiama))
			n.Put([]byte(Cach_IdleChallengeBlock), []byte(fmt.Sprintf("%d", challenge.NetSnapshot.Start)))
			n.Chal("info", fmt.Sprintf("Idle data aggregation proof: %s", idleSiama))
		} else {
			if uint32(tempInt) != challenge.NetSnapshot.Start {
				idleSiama, err = n.idleAggrProof(qslice, challenge.NetSnapshot.Start)
				if err != nil {
					return errors.Wrapf(err, "[idleAggrProof]")
				}
				n.Put([]byte(Cach_prefix_idleSiama), []byte(idleSiama))
				n.Put([]byte(Cach_IdleChallengeBlock), []byte(fmt.Sprintf("%d", challenge.NetSnapshot.Start)))
				n.Chal("info", fmt.Sprintf("Idle data aggregation proof: %s", idleSiama))
			} else {
				b, err = n.Get([]byte(Cach_prefix_idleSiama))
				if err != nil {
					idleSiama, err = n.idleAggrProof(qslice, challenge.NetSnapshot.Start)
					if err != nil {
						return errors.Wrapf(err, "[idleAggrProof]")
					}
					n.Put([]byte(Cach_prefix_idleSiama), []byte(idleSiama))
					n.Put([]byte(Cach_IdleChallengeBlock), []byte(fmt.Sprintf("%d", challenge.NetSnapshot.Start)))
					n.Chal("info", fmt.Sprintf("Idle data aggregation proof: %s", idleSiama))
				} else {
					idleSiama = string(b)
				}
			}
		}
	}

	b, err = n.Get([]byte(Cach_ServiceChallengeBlock))
	if err != nil {
		serviceSigma, err = n.serviceAggrProof(qslice, challenge.NetSnapshot.Start)
		if err != nil {
			return errors.Wrapf(err, "[serviceAggrProof]")
		}
		n.Put([]byte(Cach_prefix_serviceSiama), []byte(serviceSigma))
		n.Put([]byte(Cach_ServiceChallengeBlock), []byte(fmt.Sprintf("%d", challenge.NetSnapshot.Start)))
		n.Chal("info", fmt.Sprintf("Service data aggregation proof: %s", serviceSigma))
	} else {
		tempInt, err = strconv.Atoi(string(b))
		if err != nil {
			n.Delete([]byte(Cach_ServiceChallengeBlock))
			serviceSigma, err = n.serviceAggrProof(qslice, challenge.NetSnapshot.Start)
			if err != nil {
				return errors.Wrapf(err, "[serviceAggrProof]")
			}
			n.Put([]byte(Cach_prefix_serviceSiama), []byte(serviceSigma))
			n.Put([]byte(Cach_ServiceChallengeBlock), []byte(fmt.Sprintf("%d", challenge.NetSnapshot.Start)))
			n.Chal("info", fmt.Sprintf("Service data aggregation proof: %s", serviceSigma))
		} else {
			if uint32(tempInt) != challenge.NetSnapshot.Start {
				serviceSigma, err = n.serviceAggrProof(qslice, challenge.NetSnapshot.Start)
				if err != nil {
					return errors.Wrapf(err, "[serviceAggrProof]")
				}
				n.Put([]byte(Cach_prefix_serviceSiama), []byte(serviceSigma))
				n.Put([]byte(Cach_ServiceChallengeBlock), []byte(fmt.Sprintf("%d", challenge.NetSnapshot.Start)))
				n.Chal("info", fmt.Sprintf("Service data aggregation proof: %s", serviceSigma))
			} else {
				b, err = n.Get([]byte(Cach_prefix_serviceSiama))
				if err != nil {
					serviceSigma, err = n.idleAggrProof(qslice, challenge.NetSnapshot.Start)
					if err != nil {
						return errors.Wrapf(err, "[serviceAggrProof]")
					}
					n.Put([]byte(Cach_prefix_serviceSiama), []byte(serviceSigma))
					n.Put([]byte(Cach_ServiceChallengeBlock), []byte(fmt.Sprintf("%d", challenge.NetSnapshot.Start)))
					n.Chal("info", fmt.Sprintf("Service data aggregation proof: %s", serviceSigma))
				} else {
					serviceSigma = string(b)
				}
			}
		}
	}

	if idleSiama == "" && serviceSigma == "" {
		return errors.New("Both proofs are empty")
	}

	txhash, err := n.ReportProof(idleSiama, serviceSigma)
	if err != nil {
		return errors.Wrapf(err, "[ReportProof]")
	}

	n.Chal("info", fmt.Sprintf("Reported challenge results: %v", txhash))

	time.Sleep(pattern.BlockInterval * 3)

	err = n.transferProof(challenge)
	if err != nil {
		return errors.Wrapf(err, "[transferProof]")
	}
	return nil
}

func (n *Node) transferProof(challenge pattern.ChallengeSnapshot) error {
	idleProofFileHash, err := sutils.CalcPathSHA256Bytes(n.GetDirs().IproofFile)
	if err != nil {
		return errors.Wrapf(err, "[CalcPathSHA256Bytes]")
	}
	serviceProofFileHash, err := sutils.CalcPathSHA256Bytes(n.GetDirs().SproofFile)
	if err != nil {
		return errors.Wrapf(err, "[CalcPathSHA256Bytes]")
	}
	peerid, code, err := n.proofAssignedInfo(idleProofFileHash, serviceProofFileHash, challenge.NetSnapshot.Random_index_list, challenge.NetSnapshot.Random)
	if err != nil || code != 0 {
		return errors.Wrapf(err, "[proofAsigmentInfo]")
	}
	err = n.Put([]byte(Cach_AggrProof_Transfered), []byte(fmt.Sprintf("%s_%v", peerid, challenge.NetSnapshot.Start)))
	if err != nil {
		return errors.Wrapf(err, "[PutCache]")
	}
	return nil
}

func (n *Node) proofAssignedInfo(ihash, shash []byte, randomIndexList []uint32, random [][]byte) (string, uint32, error) {
	var err error
	var code uint32
	var teeAsigned []byte
	var peerid peer.ID
	peerid, teeAsigned, err = n.queryProofAssignedTee()
	if err != nil {
		return "", code, errors.Wrapf(err, "[queryProofAssignedTee]")
	}

	if teeAsigned == nil {
		return "", code, errors.New("proof not assigned")
	}

	var qslice = make([]*pb.Qslice, len(randomIndexList))
	for k, v := range randomIndexList {
		qslice[k] = new(pb.Qslice)
		qslice[k].I = uint64(v)
		qslice[k].V = random[k]
	}

	sign, err := n.Sign(n.GetPeerPublickey())
	if err != nil {
		return "", code, errors.Wrapf(err, "[Sign]")
	}

	addr, ok := n.GetPeer(peerid.Pretty())
	if !ok {
		addr, err = n.DHTFindPeer(peerid.Pretty())
		if err != nil {
			return "", code, fmt.Errorf("No verification proof tee found: %s", peerid.Pretty())
		}
	}

	err = n.Connect(n.GetCtxQueryFromCtxCancel(), addr)
	if err != nil {
		return "", code, fmt.Errorf("Failed to connect to verification proof tee: %s", peerid.Pretty())
	}

	code, err = n.AggrProofReq(peerid, ihash, shash, qslice, n.GetStakingPublickey(), sign)
	if err != nil || code != 0 {
		return "", code, errors.New(fmt.Sprintf("AggrProofReq to %s err: %v, code: %d", peerid.Pretty(), err, code))

	}
	n.Chal("info", fmt.Sprintf("Aggr proof response suc: %s", peerid.Pretty()))

	idleProofFileHashs, _ := sutils.CalcPathSHA256(n.GetDirs().IproofFile)
	serviceProofFileHashs, _ := sutils.CalcPathSHA256(n.GetDirs().SproofFile)

	code, err = n.FileReq(peerid, idleProofFileHashs, pb.FileType_IdleMu, n.GetDirs().IproofFile)
	if err != nil || code != 0 {
		return "", code, errors.New(fmt.Sprintf("FileReq FileType_IdleMu err: %v,code: %d", err, code))
	}
	n.Chal("info", fmt.Sprintf("Aggr proof idle file response suc: %s", peerid.Pretty()))

	code, err = n.FileReq(peerid, serviceProofFileHashs, pb.FileType_CustomMu, n.GetDirs().SproofFile)
	if err != nil || code != 0 {
		return peerid.Pretty(), code, errors.New(fmt.Sprintf("FileReq FileType_IdleMu err: %v,code: %d", err, code))
	}

	n.Chal("info", fmt.Sprintf("Aggr proof service file response suc: %s", peerid.Pretty()))
	return peerid.Pretty(), 0, nil
}

func (n *Node) idleAggrProof(qslice []proof.QElement, start uint32) (string, error) {
	idleRoothashs, err := n.QueryPrefixKeyListByHeigh(Cach_prefix_idle, start)
	if err != nil {
		return "", err
	}

	var buf []byte
	var actualCount int
	var pf ProofFileType
	var proveResponse proof.GenProofResponse
	var sigma string
	var tag pb.Tag

	pf.Names = make([]string, len(idleRoothashs))
	pf.Us = make([]string, len(idleRoothashs))
	pf.Mus = make([]string, len(idleRoothashs))

	timeout := time.NewTicker(time.Duration(time.Minute))
	defer timeout.Stop()

	for i := int(0); i < len(idleRoothashs); i++ {
		idleTagPath := filepath.Join(n.GetDirs().IdleTagDir, idleRoothashs[i]+".tag")
		buf, err = os.ReadFile(idleTagPath)
		if err != nil {
			n.Chal("err", fmt.Sprintf("Idletag not found: %v", idleTagPath))
			continue
		}

		err = json.Unmarshal(buf, &tag)
		if err != nil {
			n.Chal("err", fmt.Sprintf("Unmarshal err: %v", err))
			continue
		}

		matrix, _, err := proof.SplitByN(filepath.Join(n.GetDirs().IdleDataDir, idleRoothashs[i]), int64(len(tag.T.Phi)))
		if err != nil {
			n.Delete([]byte(Cach_prefix_idle + idleRoothashs[i]))
			os.Remove(idleTagPath)
			n.Chal("err", fmt.Sprintf("SplitByN err: %v", err))
			continue
		}

		proveResponseCh := n.key.GenProof(qslice, nil, tag.T.Phi, matrix)
		timeout.Reset(time.Minute)
		select {
		case proveResponse = <-proveResponseCh:
		case <-timeout.C:
			proveResponse.StatueMsg.StatusCode = 0
		}

		if proveResponse.StatueMsg.StatusCode != proof.Success {
			continue
		}

		sigmaTemp, ok := n.key.AggrAppendProof(sigma, qslice, tag.T.Phi)
		if !ok {
			continue
		}
		sigma = sigmaTemp
		pf.Names[actualCount] = tag.T.Name
		pf.Us[actualCount] = tag.T.U
		pf.Mus[actualCount] = proveResponse.MU
		actualCount++
	}

	pf.Names = pf.Names[:actualCount]
	pf.Us = pf.Us[:actualCount]
	pf.Mus = pf.Mus[:actualCount]
	pf.Sigma = sigma

	//
	buf, err = json.Marshal(&pf)
	if err != nil {
		return "", err
	}
	f, err := os.OpenFile(n.GetDirs().IproofFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return "", err
	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	_, err = f.Write(buf)
	if err != nil {
		return "", err
	}

	err = f.Sync()
	if err != nil {
		return "", err
	}

	f.Close()
	f = nil

	return sigma, nil
}

func (n *Node) serviceAggrProof(qslice []proof.QElement, start uint32) (string, error) {
	serviceRoothashs, err := n.QueryPrefixKeyListByHeigh(Cach_prefix_metadata, start)
	if err != nil {
		return "", err
	}

	var buf []byte
	var sigma string
	var pf ProofFileType
	var proveResponse proof.GenProofResponse
	pf.Names = make([]string, 0)
	pf.Us = make([]string, 0)
	pf.Mus = make([]string, 0)

	timeout := time.NewTicker(time.Duration(time.Minute))
	defer timeout.Stop()

	for i := int(0); i < len(serviceRoothashs); i++ {
		files, err := utils.DirFiles(filepath.Join(n.GetDirs().FileDir, serviceRoothashs[i]), 0)
		if err != nil {
			continue
		}

		for j := 0; j < len(files); j++ {
			serviceTagPath := filepath.Join(n.GetDirs().ServiceTagDir, filepath.Base(files[j])+".tag")
			buf, err = os.ReadFile(serviceTagPath)
			if err != nil {
				n.Chal("err", fmt.Sprintf("Servicetag not found: %v", serviceTagPath))
				continue
			}
			var tag pb.Tag
			err = json.Unmarshal(buf, &tag)
			if err != nil {
				n.Chal("err", fmt.Sprintf("Unmarshal %v err: %v", serviceTagPath, err))
				continue
			}
			matrix, _, err := proof.SplitByN(files[j], int64(len(tag.T.Phi)))
			if err != nil {
				n.Chal("err", fmt.Sprintf("SplitByN %v err: %v", serviceTagPath, err))
				continue
			}

			proveResponseCh := n.key.GenProof(qslice, nil, tag.T.Phi, matrix)
			timeout.Reset(time.Minute)
			select {
			case proveResponse = <-proveResponseCh:
			case <-timeout.C:
				proveResponse.StatueMsg.StatusCode = 0
			}

			if proveResponse.StatueMsg.StatusCode != proof.Success {
				fmt.Println("GenProof  err: ", proveResponse.StatueMsg.StatusCode)
				continue
			}

			sigmaTemp, ok := n.key.AggrAppendProof(sigma, qslice, tag.T.Phi)
			if !ok {
				continue
			}
			sigma = sigmaTemp
			pf.Names = append(pf.Names, tag.T.Name)
			pf.Us = append(pf.Us, tag.T.U)
			pf.Mus = append(pf.Mus, proveResponse.MU)
		}
	}
	pf.Sigma = sigma
	buf, err = json.Marshal(&pf)
	if err != nil {
		return "", err
	}
	f, err := os.OpenFile(n.GetDirs().SproofFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return "", err
	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	_, err = f.Write(buf)
	if err != nil {
		return "", err
	}
	err = f.Sync()
	if err != nil {
		return "", err
	}
	f.Close()
	f = nil

	return sigma, nil
}

func (n *Node) saveRandom(challenge pattern.ChallengeSnapshot) error {
	randfilePath := filepath.Join(n.GetDirs().ProofDir, fmt.Sprintf("random.%d", challenge.NetSnapshot.Start))
	fstat, err := os.Stat(randfilePath)
	if err == nil && fstat.Size() > 0 {
		return nil
	}
	var rd RandomList
	rd.Index = challenge.NetSnapshot.Random_index_list
	rd.Random = challenge.NetSnapshot.Random
	buff, err := json.Marshal(&rd)
	if err != nil {
		return errors.Wrapf(err, "[json.Marshal]")
	}

	f, err := os.OpenFile(randfilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "[OpenFile]")
	}
	defer f.Close()
	_, err = f.Write(buff)
	if err != nil {
		return errors.Wrapf(err, "[Write]")
	}
	return f.Sync()
}

func (n *Node) queryProofAssignedTee() (peer.ID, []byte, error) {
	var err error

	tees := n.GetAllTeeWorkAccount()

	for _, v := range tees {
		puk, err := sutils.ParsingPublickey(v)
		if err != nil {
			continue
		}
		proof, err := n.QueryTeeAssignedProof(puk)
		if err != nil {
			continue
		}

		for i := 0; i < len(proof); i++ {
			if sutils.CompareSlice(proof[i].SnapShot.Miner[:], n.GetStakingPublickey()) {
				teepeerid, ok := n.GetTeeWork(v)
				if !ok {
					continue
				}
				peerid, err := peer.Decode(base58.Encode(teepeerid))
				if err != nil {
					return "", nil, errors.Wrapf(err, "[peer.Decode]")
				}
				return peerid, puk, nil
			}
		}
	}
	return peer.ID(""), nil, err
}
