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

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
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
	Start                 uint32  `json:"start"`
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

func (n *Node) idleChallenge(
	ch chan<- bool,
	idleProofSubmited bool,
	latestBlock uint32,
	challExpiration uint32,
	challVerifyExpiration uint32,
	challStart uint32,
	minerChallFront int64,
	minerChallRear int64,
	spaceChallengeParam pattern.SpaceChallengeParam,
	minerAccumulator pattern.Accumulator,
) {
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
		if !idleProofSubmited {
			n.Ichal("err", "Proof of idle files not submitted")
			return
		}
	}

	if challVerifyExpiration <= latestBlock {
		return
	}

	err := n.checkIdleProofRecord(
		idleProofSubmited,
		challStart,
		minerChallFront,
		minerChallRear,
		minerAccumulator,
	)
	if err == nil {
		return
	}

	n.Ichal("info", fmt.Sprintf("Idle file chain challenge: %v", challStart))

	var idleProofRecord idleProofInfo
	idleProofRecord.Start = challStart
	idleProofRecord.ChainFront = minerChallFront
	idleProofRecord.ChainRear = minerChallRear

	var acc = make([]byte, len(pattern.Accumulator{}))
	for i := 0; i < len(acc); i++ {
		acc[i] = byte(minerAccumulator[i])
	}

	idleProofRecord.Acc = acc

	err = n.Prover.SetChallengeState(*n.Pois.RsaKey, acc, minerChallFront, minerChallRear)
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[SetChallengeState] %v", err))
		return
	}

	var challRandom = make([]int64, len(spaceChallengeParam))
	for i := 0; i < len(challRandom); i++ {
		challRandom[i] = int64(spaceChallengeParam[i])
	}

	idleProofRecord.ChallRandom = challRandom

	var rear int64
	var blocksProof = make([]*pb.BlocksProof, 0)

	n.Ichal("info", "start calc challenge...")
	idleProofRecord.FileBlockProofInfo = make([]fileBlockProofInfo, 0)
	var idleproof []byte

	for front := (minerChallFront + 1); front <= (minerChallRear + 1); {
		var fileBlockProofInfoEle fileBlockProofInfo
		if (front + poisSignalBlockNum) > (minerChallRear + 1) {
			rear = int64(minerChallRear + 1)
		} else {
			rear = int64(front + poisSignalBlockNum)
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
		if rear >= (minerChallRear + 1) {
			break
		}
		front += poisSignalBlockNum
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

	n.saveidleProofRecord(idleProofRecord)

	//
	txhash, err := n.SubmitIdleProof(idleProve)
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[SubmitIdleProof] %v", err))
		return
	}
	n.Ichal("info", fmt.Sprintf("SubmitIdleProof: %s", txhash))

	time.Sleep(pattern.BlockInterval * 2)

	teeAccounts := n.GetAllTeeWorkAccount()
	var teePeerIdPubkey []byte
	found := false
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
			minerChallFront,
			minerChallRear,
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
	n.saveidleProofRecord(idleProofRecord)

	spaceProofVerifyTotal, err := n.PoisRequestVerifySpaceTotalP2P(teeAddrInfo.ID, n.GetSignatureAccPulickey(), blocksProof, minerChallFront, minerChallRear, acc, challRandom, time.Duration(time.Minute*3))
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
	n.saveidleProofRecord(idleProofRecord)

	txHash, err := n.SubmitIdleProofResult(
		idleProve,
		types.U64(idleProofRecord.ChainFront),
		types.U64(idleProofRecord.ChainRear),
		minerAccumulator,
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

func (n *Node) checkIdleProofRecord(
	idleProofSubmited bool,
	challStart uint32,
	minerChallFront int64,
	minerChallRear int64,
	minerAccumulator pattern.Accumulator,
) error {
	var idleProofRecord idleProofInfo
	buf, err := os.ReadFile(filepath.Join(n.Workspace(), configs.IdleProofFile))
	if err != nil {
		return err
	}

	err = json.Unmarshal(buf, &idleProofRecord)
	if err != nil {
		return err
	}

	if idleProofRecord.Start != challStart {
		os.Remove(filepath.Join(n.Workspace(), configs.IdleProofFile))
		return errors.New("Local service file challenge record is outdated")
	}

	n.Ichal("info", fmt.Sprintf("local idle proof file challenge: %v", idleProofRecord.Start))
	if !idleProofSubmited {
		return errors.New("Idle proof not submited")
	}

	found := false
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
				n.Ichal("info", fmt.Sprintf("Allocated tee account: %v", v))
				break
			}
		}
	}
	if !found {
		n.Ichal("err", "Not found allocated tee for idle proof")
		return nil
	}
	var acc = make([]byte, len(pattern.Accumulator{}))
	for i := 0; i < len(acc); i++ {
		acc[i] = byte(minerAccumulator[i])
	}

	for {
		if idleProofRecord.TotalSignature != nil {
			var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
			for i := 0; i < len(idleProofRecord.IdleProof); i++ {
				idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
			}
			var teeSignature pattern.TeeSignature
			if len(idleProofRecord.TotalSignature) != len(teeSignature) {
				n.Ichal("err", "invalid spaceProofVerifyTotal signature")
				break
			}
			for i := 0; i < len(idleProofRecord.TotalSignature); i++ {
				teeSignature[i] = types.U8(idleProofRecord.TotalSignature[i])
			}
			txHash, err := n.SubmitIdleProofResult(
				idleProve,
				types.U64(minerChallFront),
				types.U64(minerChallRear),
				minerAccumulator,
				types.Bool(idleProofRecord.IdleResult),
				teeSignature,
				idleProofRecord.AllocatedTeeAccountId,
			)
			if err != nil {
				n.Ichal("err", fmt.Sprintf("[SubmitIdleProofResult] hash: %s, err: %v", txHash, err))
				break
			}
			n.Ichal("info", fmt.Sprintf("submit idle proof result suc: %s", txHash))
			return nil
		}
		break
	}

	teePeerIdPubkey, _ := n.GetTeeWork(idleProofRecord.AllocatedTeeAccount)
	peerid_str := base58.Encode(teePeerIdPubkey)
	n.Ichal("info", fmt.Sprintf("Allocated tee peer id: %v", peerid_str))
	teeAddrInfo, ok := n.GetPeer(peerid_str)
	if !ok {
		n.Ichal("err", fmt.Sprintf("Not discovered the tee peer: %s", peerid_str))
		return nil
	}
	err = n.Connect(n.GetCtxQueryFromCtxCancel(), teeAddrInfo)
	if err != nil {
		n.Ichal("err", fmt.Sprintf("Connect tee peer err: %v", err))
	}

	for {
		if idleProofRecord.BlocksProof != nil {
			spaceProofVerifyTotal, err := n.PoisRequestVerifySpaceTotalP2P(
				teeAddrInfo.ID,
				n.GetSignatureAccPulickey(),
				idleProofRecord.BlocksProof,
				minerChallFront,
				minerChallRear,
				acc,
				idleProofRecord.ChallRandom,
				time.Duration(time.Minute*10),
			)
			if err != nil {
				n.Ichal("err", fmt.Sprintf("[PoisRequestVerifySpaceTotalP2P] %v", err))
				break
			}
			idleProofRecord.TotalSignature = spaceProofVerifyTotal.Signature
			idleProofRecord.IdleResult = spaceProofVerifyTotal.IdleResult

			var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
			for i := 0; i < len(idleProofRecord.IdleProof); i++ {
				idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
			}
			var teeSignature pattern.TeeSignature
			if len(idleProofRecord.TotalSignature) != len(teeSignature) {
				n.Ichal("err", "invalid spaceProofVerifyTotal signature")
				break
			}

			for i := 0; i < len(idleProofRecord.TotalSignature); i++ {
				teeSignature[i] = types.U8(idleProofRecord.TotalSignature[i])
			}
			n.saveidleProofRecord(idleProofRecord)
			txHash, err := n.SubmitIdleProofResult(
				idleProve,
				types.U64(minerChallFront),
				types.U64(minerChallRear),
				minerAccumulator,
				types.Bool(idleProofRecord.IdleResult),
				teeSignature,
				idleProofRecord.AllocatedTeeAccountId,
			)
			if err != nil {
				n.Ichal("err", fmt.Sprintf("[SubmitIdleProofResult] hash: %s, err: %v", txHash, err))
				break
			}
			n.Ichal("info", fmt.Sprintf("SubmitIdleProofResult: %s", txHash))
			return nil
		}
		break
	}

	var blocksProof = make([]*pb.BlocksProof, 0)
	for i := 0; i < len(idleProofRecord.FileBlockProofInfo); i++ {
		spaceProofVerify, err := n.PoisSpaceProofVerifySingleBlockP2P(
			teeAddrInfo.ID,
			n.GetSignatureAccPulickey(),
			idleProofRecord.ChallRandom,
			n.Pois.RsaKey.N.Bytes(),
			n.Pois.RsaKey.G.Bytes(),
			acc,
			minerChallFront,
			minerChallRear,
			idleProofRecord.FileBlockProofInfo[i].SpaceProof,
			idleProofRecord.FileBlockProofInfo[i].ProofHashSign,
			time.Duration(time.Minute*3),
		)
		if err != nil {
			n.Ichal("err", fmt.Sprintf("[PoisSpaceProofVerifySingleBlockP2P] %v", err))
			return nil
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
	n.saveidleProofRecord(idleProofRecord)

	spaceProofVerifyTotal, err := n.PoisRequestVerifySpaceTotalP2P(
		teeAddrInfo.ID,
		n.GetSignatureAccPulickey(),
		blocksProof,
		minerChallFront,
		minerChallRear,
		acc,
		idleProofRecord.ChallRandom,
		time.Duration(time.Minute*10),
	)
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[PoisRequestVerifySpaceTotalP2P] %v", err))
		return nil
	}

	var teeSignature pattern.TeeSignature
	if len(spaceProofVerifyTotal.Signature) != len(teeSignature) {
		n.Ichal("err", "invalid spaceProofVerifyTotal signature")
		return nil
	}

	for i := 0; i < len(spaceProofVerifyTotal.Signature); i++ {
		teeSignature[i] = types.U8(spaceProofVerifyTotal.Signature[i])
	}

	var idleProve = make([]types.U8, len(idleProofRecord.IdleProof))
	for i := 0; i < len(idleProofRecord.IdleProof); i++ {
		idleProve[i] = types.U8(idleProofRecord.IdleProof[i])
	}
	idleProofRecord.TotalSignature = spaceProofVerifyTotal.Signature
	idleProofRecord.IdleResult = spaceProofVerifyTotal.IdleResult
	n.saveidleProofRecord(idleProofRecord)
	txHash, err := n.SubmitIdleProofResult(
		idleProve,
		types.U64(minerChallFront),
		types.U64(minerChallRear),
		minerAccumulator,
		types.Bool(spaceProofVerifyTotal.IdleResult),
		teeSignature,
		idleProofRecord.AllocatedTeeAccountId,
	)
	if err != nil {
		n.Ichal("err", fmt.Sprintf("[SubmitIdleProofResult] hash: %s, err: %v", txHash, err))
		return nil
	}
	n.Ichal("info", fmt.Sprintf("submit idle proof result suc: %s", txHash))
	return nil
}

func (n *Node) saveidleProofRecord(idleProofRecord idleProofInfo) {
	buf, err := json.Marshal(&idleProofRecord)
	if err == nil {
		err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), configs.IdleProofFile))
		if err != nil {
			n.Schal("err", err.Error())
		}
	}
}
