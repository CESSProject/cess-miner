/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"runtime"
	"time"

	sprocess "github.com/shirou/gopsutil/process"
)

type Node struct {
	// sdk.SDK
	// confile.Confile
	// logger.Logger
	// cache.Cache
	// TeeRecord
	// MinerRecord
	// RunningStater
	// *core.PeerNode
	// *gin.Engine
	// *proof.RSAKeyPair
	// *pb.MinerPoisInfo
	// *DataDir
	// *Pois
}

// New is used to build a node instance
func New() *Node {
	return &Node{}
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
