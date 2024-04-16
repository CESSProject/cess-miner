/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"crypto/x509"
	"runtime"
	"time"

	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/proof"
	"github.com/CESSProject/cess-go-sdk/core/sdk"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/gin-gonic/gin"
	sprocess "github.com/shirou/gopsutil/process"
)

type Node struct {
	sdk.SDK
	confile.Confile
	logger.Logger
	cache.Cache
	MinerState
	TeeRecord
	PeerRecord
	RunningStater
	*core.PeerNode
	*gin.Engine
	*proof.RSAKeyPair
	*pb.MinerPoisInfo
	*DataDir
	*Pois
}

func (n *Node) GetPodr2Key() *proof.RSAKeyPair {
	return n.RSAKeyPair
}

func (n *Node) SetPublickey(pubkey []byte) error {
	rsaPubkey, err := x509.ParsePKCS1PublicKey(pubkey)
	if err != nil {
		return err
	}

	if n.RSAKeyPair == nil {
		n.RSAKeyPair = proof.NewKey()
	}
	n.RSAKeyPair.Spk = rsaPubkey
	return nil
}

func getCpuUsage(pid int32) float64 {
	p, _ := sprocess.NewProcess(pid)
	cpuPercent, err := p.Percent(time.Second)
	if err != nil {
		return 0
	}
	return cpuPercent / float64(runtime.NumCPU())
}

func getMemUsage() uint64 {
	memSt := &runtime.MemStats{}
	runtime.ReadMemStats(memSt)
	return memSt.HeapSys + memSt.StackSys
}
