/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/CESSProject/cess_pois/acc"
	"github.com/CESSProject/cess_pois/pois"
)

var minSpace = uint64(pois.FileSize * chain.SIZE_1MiB * acc.DEFAULT_ELEMS_NUM * 2)

func NewPoisProver(expendersInfo chain.ExpendersInfo, freeSpace, count int64, signAccPulickey []byte) (*pois.Prover, error) {
	// k,n,d and key are params that needs to be negotiated with the verifier in advance.
	// minerID is storage node's account ID, and space is the amount of physical space available(MiB)
	prover, err := pois.NewProver(
		int64(expendersInfo.K),
		int64(expendersInfo.N),
		int64(expendersInfo.D),
		signAccPulickey,
		freeSpace,
		count,
	)
	return prover, err
}
