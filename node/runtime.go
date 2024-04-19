/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"sync"
	"time"

	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/out"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const (
	St_Normal uint8 = iota
	St_Warning
	St_Error
)

type RunningStater interface {
	SetStatus
	GetStatus
}

type SetStatus interface {
	SetCpuCores(num int)
	SetPID(pid int)
	SetLastReconnectRpcTime(t string)
	SetCalcTagFlag(flag bool)
	SetReportFileFlag(flag bool)
	SetGenIdleFlag(flag bool)
	SetAuthIdleFlag(flag bool)
	SetIdleChallengeFlag(flag bool)
	SetServiceChallengeFlag(flag bool)
	SetChainStatus(status bool)
	SetReceiveFlag(flag bool)
	SetCurrentRpc(rpc string)

	// miner
	SetMinerState(state string) error
	SetMinerSpaceInfo(decSpace, validSpace, usedSpace, lockedSpace uint64)
	SetMinerSignAcc(acc string)
}

type GetStatus interface {
	GetCpuCores() int
	GetPID() int
	GetLastReconnectRpcTime() string
	GetCalcTagFlag() bool
	GetReportFileFlag() bool
	GetGenIdleFlag() bool
	GetAuthIdleFlag() bool
	GetIdleChallengeFlag() bool
	GetServiceChallengeFlag() bool
	GetChainStatus() bool
	GetReceiveFlag() bool
	GetCurrentRpc() string

	// miner
	GetMinerState() string
	GetMinerSpaceInfo() (uint64, uint64, uint64, uint64)
	GetMinerSignatureAcc() string
}

type RunningState struct {
	lock                 *sync.RWMutex
	minerLock            *sync.RWMutex
	minerSignAcc         string
	minerState           string
	currentRpc           string
	lastReconnectRpcTime string
	decSpace             uint64
	validSpace           uint64
	usedSpace            uint64
	lockedSpace          uint64
	cpuCores             int
	pid                  int
	calcTagFlag          bool
	reportFileFlag       bool
	genIdleFlag          bool
	authIdleFlag         bool
	idleChallengeFlag    bool
	serviceChallengeFlag bool
	chainStatus          bool
	receiveFlag          bool
}

var _ RunningStater = (*RunningState)(nil)

func NewRunTime() *RunningState {
	return &RunningState{
		lock:      new(sync.RWMutex),
		minerLock: new(sync.RWMutex),
	}
}

func (s *RunningState) ListenLocal() {
	var port uint32 = 6000
	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET"}
	engine.Use(cors.New(config))
	engine.Use(AllowSpecificRoute("/status"))
	for {
		if !core.FreeLocalPort(port) {
			port++
		} else {
			break
		}
	}
	engine.GET("/status", s.getStatusHandle)
	go engine.Run(fmt.Sprintf("localhost:%d", port))
	time.Sleep(time.Second)
	if !core.FreeLocalPort(port) {
		out.Tip(fmt.Sprintf("Local service started: [GET] localhost:%d/status", port))
	}
}

func AllowSpecificRoute(allowedPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == allowedPath {
			c.Next()
		} else {
			c.Abort()
		}
	}
}

// getStatusHandle
func (n *RunningState) getStatusHandle(c *gin.Context) {
	var msg string

	msg += fmt.Sprintf("Process ID: %d\n", n.GetPID())

	msg += fmt.Sprintf("Miner Signature Account: %s\n", n.GetMinerSignatureAcc())

	msg += fmt.Sprintf("Miner State: %s\n", n.GetMinerState())

	if n.GetChainStatus() {
		msg += fmt.Sprintf("RPC Connection: [ok] %v\n", n.GetCurrentRpc())
	} else {
		msg += fmt.Sprintf("RPC Connection: [fail] %v\n", n.GetCurrentRpc())
	}
	msg += fmt.Sprintf("Last reconnection: %v\n", n.GetLastReconnectRpcTime())

	msg += fmt.Sprintf("Calculate Tag: %v\n", n.GetCalcTagFlag())

	msg += fmt.Sprintf("Report file: %v\n", n.GetReportFileFlag())

	msg += fmt.Sprintf("Generate idle: %v\n", n.GetGenIdleFlag())

	msg += fmt.Sprintf("Report idle: %v\n", n.GetAuthIdleFlag())

	msg += fmt.Sprintf("Calc idle challenge: %v\n", n.GetIdleChallengeFlag())

	msg += fmt.Sprintf("Calc service challenge: %v\n", n.GetServiceChallengeFlag())

	msg += fmt.Sprintf("Receiving data: %v\n", n.GetReceiveFlag())

	msg += fmt.Sprintf("Cpu usage: %.2f%%\n", getCpuUsage(int32(n.GetPID())))

	msg += fmt.Sprintf("Memory usage: %d", getMemUsage())

	c.Data(200, "application/octet-stream", []byte(msg))
}

func (s *RunningState) SetLastReconnectRpcTime(t string) {
	s.lock.Lock()
	s.lastReconnectRpcTime = t
	s.lock.Unlock()
}

func (s *RunningState) GetLastReconnectRpcTime() string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.lastReconnectRpcTime
}

func (s *RunningState) SetCpuCores(num int) {
	s.lock.Lock()
	s.cpuCores = num
	s.lock.Unlock()
}

func (s *RunningState) GetPID() int {
	return s.pid
}

func (s *RunningState) SetPID(pid int) {
	s.lock.Lock()
	s.pid = pid
	s.lock.Unlock()
}

func (s *RunningState) GetCpuCores() int {
	return s.cpuCores
}

func (s *RunningState) SetCalcTagFlag(flag bool) {
	s.lock.Lock()
	s.calcTagFlag = flag
	s.lock.Unlock()
}

func (s *RunningState) GetCalcTagFlag() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.calcTagFlag
}

func (s *RunningState) SetReportFileFlag(flag bool) {
	s.lock.Lock()
	s.reportFileFlag = flag
	s.lock.Unlock()
}

func (s *RunningState) GetReportFileFlag() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.reportFileFlag
}

func (s *RunningState) SetGenIdleFlag(flag bool) {
	s.lock.Lock()
	s.genIdleFlag = flag
	s.lock.Unlock()
}

func (s *RunningState) GetGenIdleFlag() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.genIdleFlag
}

func (s *RunningState) SetAuthIdleFlag(flag bool) {
	s.lock.Lock()
	s.authIdleFlag = flag
	s.lock.Unlock()
}

func (s *RunningState) GetAuthIdleFlag() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.authIdleFlag
}

func (s *RunningState) SetIdleChallengeFlag(flag bool) {
	s.lock.Lock()
	s.idleChallengeFlag = flag
	s.lock.Unlock()
}

func (s *RunningState) GetIdleChallengeFlag() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.idleChallengeFlag
}

func (s *RunningState) SetServiceChallengeFlag(flag bool) {
	s.lock.Lock()
	s.serviceChallengeFlag = flag
	s.lock.Unlock()
}

func (s *RunningState) GetServiceChallengeFlag() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.serviceChallengeFlag
}

func (s *RunningState) SetChainStatus(status bool) {
	s.lock.Lock()
	s.chainStatus = status
	s.lock.Unlock()
}

func (s *RunningState) GetChainStatus() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.chainStatus
}

func (s *RunningState) SetReceiveFlag(flag bool) {
	s.lock.Lock()
	s.receiveFlag = flag
	s.lock.Unlock()
}

func (s *RunningState) GetReceiveFlag() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.receiveFlag
}

func (s *RunningState) SetCurrentRpc(rpc string) {
	s.lock.Lock()
	s.currentRpc = rpc
	s.lock.Unlock()
}

func (s *RunningState) GetCurrentRpc() string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.currentRpc
}

func (m *RunningState) SetMinerState(state string) error {
	m.minerLock.Lock()
	defer m.minerLock.Unlock()
	switch state {
	case pattern.MINER_STATE_POSITIVE:
		m.minerState = pattern.MINER_STATE_POSITIVE

	case pattern.MINER_STATE_FROZEN:
		m.minerState = pattern.MINER_STATE_FROZEN

	case pattern.MINER_STATE_LOCK:
		m.minerState = pattern.MINER_STATE_LOCK

	case pattern.MINER_STATE_EXIT:
		m.minerState = pattern.MINER_STATE_EXIT

	case pattern.MINER_STATE_OFFLINE:
		m.minerState = pattern.MINER_STATE_OFFLINE

	default:
		return fmt.Errorf("invalid miner state: %s", state)
	}

	return nil
}

func (m *RunningState) SetMinerSignAcc(acc string) {
	m.minerLock.Lock()
	m.minerSignAcc = acc
	m.minerLock.Unlock()
}

func (m *RunningState) SetMinerSpaceInfo(decSpace, validSpace, usedSpace, lockedSpace uint64) {
	m.minerLock.Lock()
	m.decSpace = decSpace
	m.validSpace = validSpace
	m.usedSpace = usedSpace
	m.lockedSpace = lockedSpace
	m.minerLock.Unlock()
}

func (s *RunningState) GetMinerSignatureAcc() string {
	s.minerLock.RLock()
	defer s.minerLock.RUnlock()
	return s.minerSignAcc
}

func (s *RunningState) GetMinerState() string {
	s.minerLock.RLock()
	defer s.minerLock.RUnlock()
	return s.minerState
}

func (m *RunningState) GetMinerSpaceInfo() (uint64, uint64, uint64, uint64) {
	m.minerLock.RLock()
	decSpace := m.decSpace
	validSpace := m.validSpace
	usedSpace := m.usedSpace
	lockedSpace := m.lockedSpace
	m.minerLock.RUnlock()
	return decSpace, validSpace, usedSpace, lockedSpace
}
