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

type MinerState interface {
	SaveMinerState(state string) error
	GetMinerState() string
}

type MinerStateType struct {
	lock  *sync.RWMutex
	state string
}

var _ MinerState = (*MinerStateType)(nil)

func NewMinerState() MinerState {
	return &MinerStateType{
		lock: new(sync.RWMutex),
	}
}

func (m *MinerStateType) SaveMinerState(state string) error {
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

func (m *MinerStateType) GetMinerState() string {
	m.lock.RLock()
	result := m.state
	m.lock.RUnlock()
	return result
}
