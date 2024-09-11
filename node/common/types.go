/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/CESSProject/cess-miner/pkg/com/pb"
)

type RespType struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

type fileBlockProofInfo struct {
	ProofHashSign       []byte         `json:"proofHashSign"`
	ProofHashSignOrigin []byte         `json:"proofHashSignOrigin"`
	SpaceProof          *pb.SpaceProof `json:"spaceProof"`
	FileBlockFront      int64          `json:"fileBlockFront"`
	FileBlockRear       int64          `json:"fileBlockRear"`
}

type IdleProofInfo struct {
	Start               uint32                `json:"start"`
	ChainFront          int64                 `json:"chainFront"`
	ChainRear           int64                 `json:"chainRear"`
	IdleResult          bool                  `json:"idleResult"`
	SubmintProof        bool                  `json:"submintProof"`
	SubmintResult       bool                  `json:"submintResult"`
	AllocatedTeeWorkpuk chain.WorkerPublicKey `json:"allocatedTeeWorkpuk"`
	IdleProof           []byte                `json:"idleProof"`
	Acc                 []byte                `json:"acc"`
	TotalSignature      []byte                `json:"totalSignature"`
	ChallRandom         []int64               `json:"challRandom"`
	FileBlockProofInfo  []fileBlockProofInfo
	BlocksProof         []*pb.BlocksProof
}

type ServiceProofInfo struct {
	Names               []string              `json:"names"`
	Us                  []string              `json:"us"`
	Mus                 []string              `json:"mus"`
	Usig                [][]byte              `json:"usig"`
	ServiceBloomFilter  []uint64              `json:"serviceBloomFilter"`
	Signature           []byte                `json:"signature"`
	AllocatedTeeWorkpuk chain.WorkerPublicKey `json:"allocatedTeeWorkpuk"`
	Sigma               string                `json:"sigma"`
	Start               uint32                `json:"start"`
	ServiceResult       bool                  `json:"serviceResult"`
	SubmitProof         bool                  `json:"submitProof"`
	SubmitResult        bool                  `json:"submitResult"`
}

type RandomList struct {
	Index  []uint32 `json:"index"`
	Random [][]byte `json:"random"`
}
