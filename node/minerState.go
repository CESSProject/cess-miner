/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"sync"

	"github.com/CESSProject/cess-go-sdk/core/pattern"
)

type MinerStater interface {
	// set
	SaveMinerState(state string) error
	SaveMinerSpaceInfo(decSpace, validSpace, usedSpace, lockedSpace uint64)

	//get
	GetMinerState() string
	GetMinerSpaceInfo() (uint64, uint64, uint64, uint64)
}

type MinerState struct {
	lock        *sync.RWMutex
	state       string
	decSpace    uint64
	validSpace  uint64
	usedSpace   uint64
	lockedSpace uint64
}

var _ MinerStater = (*MinerState)(nil)

func NewMinerState() *MinerState {
	return &MinerState{
		lock: new(sync.RWMutex),
	}
}

func (m *MinerState) SaveMinerState(state string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	switch state {
	case pattern.MINER_STATE_POSITIVE:
		m.state = pattern.MINER_STATE_POSITIVE

	case pattern.MINER_STATE_FROZEN:
		m.state = pattern.MINER_STATE_FROZEN

	case pattern.MINER_STATE_LOCK:
		m.state = pattern.MINER_STATE_LOCK

	case pattern.MINER_STATE_EXIT:
		m.state = pattern.MINER_STATE_EXIT

	case pattern.MINER_STATE_OFFLINE:
		m.state = pattern.MINER_STATE_OFFLINE

	default:
		return fmt.Errorf("invalid miner state: %s", state)
	}

	return nil
}

func (m *MinerState) SaveMinerSpaceInfo(decSpace, validSpace, usedSpace, lockedSpace uint64) {
	m.lock.Lock()
	m.decSpace = decSpace
	m.validSpace = validSpace
	m.usedSpace = usedSpace
	m.lockedSpace = lockedSpace
	m.lock.Unlock()
}

func (m *MinerState) GetMinerState() string {
	m.lock.RLock()
	result := m.state
	m.lock.RUnlock()
	return result
}

func (m *MinerState) GetMinerSpaceInfo() (uint64, uint64, uint64, uint64) {
	m.lock.RLock()
	decSpace := m.decSpace
	validSpace := m.validSpace
	usedSpace := m.usedSpace
	lockedSpace := m.lockedSpace
	m.lock.RUnlock()
	return decSpace, validSpace, usedSpace, lockedSpace
}
