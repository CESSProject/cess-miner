/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package runstatus

import (
	"sync"
)

type Minerst interface {
	SetSignAcc(acc string)
	SetStakingAcc(acc string)
	SetEarningsAcc(acc string)
	SetState(st string)
	// declaration_space, idle_space, service_space, locked_space
	SetSpaceInfo(decSpace, validSpace, usedSpace, lockedSpace uint64)
	SetRegister(register bool)

	GetSignAcc() string
	GetStakingAcc() string
	GetEarningsAcc() string
	GetState() string
	// declaration_space, idle_space, service_space, locked_space
	GetMinerSpaceInfo() (uint64, uint64, uint64, uint64)
	GetRegister() bool
}

type MinerSt struct {
	lock           *sync.RWMutex
	signAcc        string
	stakingAcc     string
	earningsAcc    string
	state          string
	decSpace       uint64
	validSpace     uint64
	usedSpace      uint64
	lockedSpace    uint64
	calcTagFlag    bool
	reportFileFlag bool
	genIdleFlag    bool
	authIdleFlag   bool
	register       bool
}

func NewMinerSt() *MinerSt {
	return &MinerSt{
		lock: new(sync.RWMutex),
	}
}

func (m *MinerSt) SetSignAcc(acc string) {
	m.lock.Lock()
	m.signAcc = acc
	m.lock.Unlock()
}

func (m *MinerSt) GetSignAcc() string {
	m.lock.RLock()
	value := m.signAcc
	m.lock.RUnlock()
	return value
}

func (m *MinerSt) SetStakingAcc(acc string) {
	m.lock.Lock()
	m.stakingAcc = acc
	m.lock.Unlock()
}

func (m *MinerSt) GetStakingAcc() string {
	m.lock.RLock()
	value := m.stakingAcc
	m.lock.RUnlock()
	return value
}

func (m *MinerSt) SetEarningsAcc(acc string) {
	m.lock.Lock()
	m.earningsAcc = acc
	m.lock.Unlock()
}

func (m *MinerSt) GetEarningsAcc() string {
	m.lock.RLock()
	value := m.earningsAcc
	m.lock.RUnlock()
	return value
}

func (m *MinerSt) SetState(st string) {
	m.lock.Lock()
	m.state = st
	m.lock.Unlock()
}

func (m *MinerSt) GetState() string {
	m.lock.RLock()
	value := m.state
	m.lock.RUnlock()
	return value
}

func (m *MinerSt) SetSpaceInfo(decSpace, validSpace, usedSpace, lockedSpace uint64) {
	m.lock.Lock()
	m.decSpace = decSpace
	m.validSpace = validSpace
	m.usedSpace = usedSpace
	m.lockedSpace = lockedSpace
	m.lock.Unlock()
}

func (m *MinerSt) GetMinerSpaceInfo() (uint64, uint64, uint64, uint64) {
	m.lock.RLock()
	decSpace := m.decSpace
	validSpace := m.validSpace
	usedSpace := m.usedSpace
	lockedSpace := m.lockedSpace
	m.lock.RUnlock()
	return decSpace, validSpace, usedSpace, lockedSpace
}

func (m *MinerSt) SetRegister(register bool) {
	m.lock.Lock()
	m.register = register
	m.lock.Unlock()
}

func (m *MinerSt) GetRegister() bool {
	m.lock.RLock()
	value := m.register
	m.lock.RUnlock()
	return value
}
