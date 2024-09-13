/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package runstatus

import (
	"sync"
)

type Processst interface {
	SetPID(pid int)
	SetCpucores(cores int)
	SetComAddr(addr string)

	GetPID() int
	GetCpucores() int
	GetComAddr() string
}

type ProcessSt struct {
	lock     *sync.RWMutex
	cpucores int
	pid      int
	addr     string
}

func NewProcessSt() *ProcessSt {
	return &ProcessSt{
		lock: new(sync.RWMutex),
	}
}

func (p *ProcessSt) SetPID(pid int) {
	p.lock.Lock()
	p.pid = pid
	p.lock.Unlock()
}

func (p *ProcessSt) GetPID() int {
	p.lock.RLock()
	value := p.pid
	p.lock.RUnlock()
	return value
}

func (p *ProcessSt) SetCpucores(cores int) {
	p.lock.Lock()
	p.cpucores = cores
	p.lock.Unlock()
}

func (p *ProcessSt) GetCpucores() int {
	p.lock.RLock()
	value := p.cpucores
	p.lock.RUnlock()
	return value
}

func (p *ProcessSt) SetComAddr(addr string) {
	p.lock.Lock()
	p.addr = addr
	p.lock.Unlock()
}

func (p *ProcessSt) GetComAddr() string {
	p.lock.RLock()
	value := p.addr
	p.lock.RUnlock()
	return value
}
