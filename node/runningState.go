/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"strings"
	"sync"
	"time"

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

// init stage
const (
	Stage_Startup uint8 = iota
	Stage_ReadConfig
	Stage_ConnectRpc
	Stage_CreateP2p
	Stage_SyncBlock
	Stage_QueryChain
	Stage_Register
	Stage_BuildDir
	Stage_BuildCache
	Stage_BuildLog
	Stage_Complete
)

type RunningStater interface {
	SetStatus
	GetStatus
}

type SetStatus interface {
	SetInitStage(st uint8, msg string)
	SetTaskPeriod(msg string)
	SetCpuCores(num int)
	SetPID(pid int32)
	SetLastReconnectRpcTime(t string)
	SetCalcTagFlag(flag bool)
	SetReportFileFlag(flag bool)
	SetGenIdleFlag(flag bool)
	SetAuthIdleFlag(flag bool)
	SetIdleChallengeFlag(flag bool)
	SetServiceChallengeFlag(flag bool)
}

type GetStatus interface {
	GetInitStage() [Stage_Complete + 1]string
	GetTaskPeriod() string
	GetCpuCores() int
	GetPID() int32
	GetLastReconnectRpcTime() string
	GetCalcTagFlag() bool
	GetReportFileFlag() bool
	GetGenIdleFlag() bool
	GetAuthIdleFlag() bool
	GetIdleChallengeFlag() bool
	GetServiceChallengeFlag() bool
}

type RunningState struct {
	lock                 *sync.RWMutex
	initStageMsg         [Stage_Complete + 1]string
	taskPeriod           string
	cpuCores             int
	pid                  int32
	lastReconnectRpcTime string
	calcTagFlag          bool
	reportFileFlag       bool
	genIdleFlag          bool
	authIdleFlag         bool
	idleChallengeFlag    bool
	serviceChallengeFlag bool
}

var _ RunningStater = (*RunningState)(nil)

func NewRunningState() *RunningState {
	return &RunningState{
		lock: new(sync.RWMutex),
	}
}

func (s *RunningState) ListenLocal() {
	var port uint32 = 6000
	engine := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET"}
	engine.Use(cors.New(config))
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

// getStatusHandle
func (n *RunningState) getStatusHandle(c *gin.Context) {
	var msg string
	initStage := n.GetInitStage()
	if !strings.Contains(initStage[Stage_Complete], "[ok]") {
		msg += "init stage: \n"
		for i := 0; i < len(initStage); i++ {
			msg += fmt.Sprintf("    %d: %s\n", i, initStage[i])
		}
	}
	msg += fmt.Sprintf("Process ID: %d\n", n.GetPID())

	msg += fmt.Sprintf("Task Stage: %s\n", n.GetTaskPeriod())

	msg += fmt.Sprintf("Miner State: %s\n", n.GetMinerState())

	if n.GetChainState() {
		msg += fmt.Sprintf("RPC Connection: [ok] %v\n", n.GetCurrentRpcAddr())
	} else {
		msg += fmt.Sprintf("RPC Connection: [fail] %v\n", n.GetCurrentRpcAddr())
	}
	msg += fmt.Sprintf("Last reconnection: %v\n", n.GetLastReconnectRpcTime())

	msg += fmt.Sprintf("Calculate Tag: %v\n", n.GetCalcTagFlag())

	msg += fmt.Sprintf("Report file: %v\n", n.GetReportFileFlag())

	msg += fmt.Sprintf("Generate idle: %v\n", n.GetGenIdleFlag())

	msg += fmt.Sprintf("Report idle: %v\n", n.GetAuthIdleFlag())

	msg += fmt.Sprintf("Calc idle challenge: %v\n", n.GetIdleChallengeFlag())

	msg += fmt.Sprintf("Calc service challenge: %v\n", n.GetServiceChallengeFlag())

	msg += fmt.Sprintf("Receiving data: %v\n", n.PeerNode.GetRecvFlag())

	msg += fmt.Sprintf("Cpu usage: %.2f%%\n", getCpuUsage(int32(n.GetPID())))

	msg += fmt.Sprintf("Memory usage: %d", getMemUsage())

	c.Data(200, "application/octet-stream", []byte(msg))
}

func (s *RunningState) SetInitStage(stage uint8, msg string) {
	s.lock.Lock()
	s.initStageMsg[stage] = msg
	s.lock.Unlock()
}

func (s *RunningState) GetInitStage() [Stage_Complete + 1]string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.initStageMsg
}

func (s *RunningState) SetTaskPeriod(msg string) {
	s.lock.Lock()
	s.taskPeriod = msg
	s.lock.Unlock()
}

func (s *RunningState) GetTaskPeriod() string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.taskPeriod
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

func (s *RunningState) GetPID() int32 {
	return s.pid
}

func (s *RunningState) SetPID(pid int32) {
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
