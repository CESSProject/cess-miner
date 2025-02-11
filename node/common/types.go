/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/AstaFrode/go-substrate-rpc-client/v4/types"
	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/CESSProject/cess-miner/pkg/com/pb"
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
	CanSubmintProof     bool                  `json:"submintProof"`
	CanSubmintResult    bool                  `json:"submintResult"`
	AllocatedTeeWorkpuk chain.WorkerPublicKey `json:"allocatedTeeWorkpuk"`
	IdleProof           []types.U8            `json:"idleProof"`
	Acc                 []byte                `json:"acc"`
	TotalSignature      []byte                `json:"totalSignature"`
	ChallRandom         []int64               `json:"challRandom"`
	FileBlockProofInfo  []FileBlockProofInfo
	BlocksProof         []*pb.BlocksProof
}

type ServiceProofInfo struct {
	SignatureHex    string                `json:"signature_hex"`
	Proof           []types.U8            `json:"proof"`
	BloomFilter     chain.BloomFilter     `json:"bloom_filter"`
	TeePublicKey    chain.WorkerPublicKey `json:"tee_public_key"`
	Start           uint32                `json:"start"`
	Result          bool                  `json:"result"`
	CanSubmitProof  bool                  `json:"can_submit_proof"`
	CanSubmitResult bool                  `json:"can_submit_result"`
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
