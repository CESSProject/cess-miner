/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package runstatus

import (
	"sync"
)

type Challengest interface {
	SetLastChallenge(blocknumber uint32)
	SetIdleChallenging(st bool)
	SetServiceChallenging(st bool)

	GetLastChallenge() uint32
	GetIdleChallenging() bool
	GetServiceChallenging() bool
}

type ChallengeSt struct {
	lock               *sync.RWMutex
	lastChallenge      uint32
	idleChallenging    bool
	serviceChallenging bool
}

func NewChallengeSt() *ChallengeSt {
	return &ChallengeSt{
		lock: new(sync.RWMutex),
	}
}

func (c *ChallengeSt) SetLastChallenge(blocknumber uint32) {
	c.lock.Lock()
	c.lastChallenge = blocknumber
	c.lock.Unlock()
}

func (c *ChallengeSt) GetLastChallenge() uint32 {
	c.lock.RLock()
	value := c.lastChallenge
	c.lock.RUnlock()
	return value
}

func (c *ChallengeSt) SetIdleChallenging(st bool) {
	c.lock.Lock()
	c.idleChallenging = st
	c.lock.Unlock()
}

func (c *ChallengeSt) GetIdleChallenging() bool {
	c.lock.RLock()
	value := c.idleChallenging
	c.lock.RUnlock()
	return value
}

func (c *ChallengeSt) SetServiceChallenging(st bool) {
	c.lock.Lock()
	c.serviceChallenging = st
	c.lock.Unlock()
}

func (c *ChallengeSt) GetServiceChallenging() bool {
	c.lock.RLock()
	value := c.serviceChallenging
	c.lock.RUnlock()
	return value
}
