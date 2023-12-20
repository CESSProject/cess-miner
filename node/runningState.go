/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"sync"
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

type RunningRecord interface {
	SetStatus
	GetStatus
}

type SetStatus interface {
	SetInitStage(st uint8, msg string)
	SetTaskPeriod(msg string)
	SetCpuCores(num int)
	SetReconnectRpc(value bool)
	SetCalcTagFlag(flag bool)
	SetReportFileFlag(flag bool)
	SetGenIdleFlag(flag bool)
	SetAuthIdleFlag(flag bool)
}

type GetStatus interface {
	GetInitStage() [Stage_Complete + 1]string
	GetTaskPeriod() string
	GetCpuCores() int
	GetReconnectRpc() bool
	GetCalcTagFlag() bool
	GetReportFileFlag() bool
	GetGenIdleFlag() bool
	GetAuthIdleFlag() bool
}

type RunningRecordType struct {
	lock           *sync.RWMutex
	initStageMsg   [Stage_Complete + 1]string
	taskPeriod     string
	cpuCores       int
	workStage      uint8
	reconnectRpc   bool
	calcTagFlag    bool
	reportFileFlag bool
	genIdleFlag    bool
	authIdleFlag   bool
}

var _ RunningRecord = (*RunningRecordType)(nil)

func NewRunningRecord() RunningRecord {
	return &RunningRecordType{
		lock: new(sync.RWMutex),
	}
}

func (s *RunningRecordType) SetInitStage(stage uint8, msg string) {
	s.lock.Lock()
	s.initStageMsg[stage] = msg
	s.lock.Unlock()
}

func (s *RunningRecordType) GetInitStage() [Stage_Complete + 1]string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.initStageMsg
}

func (s *RunningRecordType) SetTaskPeriod(msg string) {
	s.lock.Lock()
	s.taskPeriod = msg
	s.lock.Unlock()
}

func (s *RunningRecordType) GetTaskPeriod() string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.taskPeriod
}

func (s *RunningRecordType) SetReconnectRpc(value bool) {
	s.lock.Lock()
	s.reconnectRpc = value
	s.lock.Unlock()
}

func (s *RunningRecordType) GetReconnectRpc() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.reconnectRpc
}

func (s *RunningRecordType) SetCpuCores(num int) {
	s.lock.Lock()
	s.cpuCores = num
	s.lock.Unlock()
}

func (s *RunningRecordType) GetCpuCores() int {
	return s.cpuCores
}

func (s *RunningRecordType) SetCalcTagFlag(flag bool) {
	s.lock.Lock()
	s.calcTagFlag = flag
	s.lock.Unlock()
}

func (s *RunningRecordType) GetCalcTagFlag() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.calcTagFlag
}

func (s *RunningRecordType) SetReportFileFlag(flag bool) {
	s.lock.Lock()
	s.reportFileFlag = flag
	s.lock.Unlock()
}

func (s *RunningRecordType) GetReportFileFlag() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.reportFileFlag
}

func (s *RunningRecordType) SetGenIdleFlag(flag bool) {
	s.lock.Lock()
	s.genIdleFlag = flag
	s.lock.Unlock()
}

func (s *RunningRecordType) GetGenIdleFlag() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.genIdleFlag
}

func (s *RunningRecordType) SetAuthIdleFlag(flag bool) {
	s.lock.Lock()
	s.authIdleFlag = flag
	s.lock.Unlock()
}

func (s *RunningRecordType) GetAuthIdleFlag() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.authIdleFlag
}
