/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/CESSProject/cess-miner/pkg/com/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/gin-gonic/gin"
)

type RespType struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

type FileBlockProofInfo struct {
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
	IdleProof           []types.U8            `json:"idleProof"`
	Acc                 []byte                `json:"acc"`
	TotalSignature      []byte                `json:"totalSignature"`
	ChallRandom         []int64               `json:"challRandom"`
	FileBlockProofInfo  []FileBlockProofInfo
	BlocksProof         []*pb.BlocksProof
}

type ServiceProofInfo struct {
	// Names              []string              `json:"names"`
	// Us                 []string              `json:"us"`
	// Mus                []string              `json:"mus"`
	// Usig               [][]byte              `json:"usig"`
	Signature          []byte                `json:"signature"`
	Proof              []types.U8            `json:"proof"`
	BloomFilter        chain.BloomFilter     `json:"bloom_filter"`
	TeeWorkerPublicKey chain.WorkerPublicKey `json:"tee_worker_public_key"`
	Sigma              string                `json:"sigma"`
	Start              uint32                `json:"start"`
	ServiceResult      bool                  `json:"serviceResult"`
	SubmitProof        bool                  `json:"submitProof"`
	SubmitResult       bool                  `json:"submitResult"`
}

type RandomList struct {
	Index  []uint32 `json:"index"`
	Random [][]byte `json:"random"`
}

func ReturnJSON(c *gin.Context, code int, msg string, data any) {
	c.JSON(200, RespType{
		Code: code,
		Msg:  msg,
		Data: data,
	})
}
