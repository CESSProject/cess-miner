/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package runstatus

import (
	"sync"
)

type Idlest interface {
	SetGeneratingIdle(st bool)
	SetCertifyingIdle(st bool)

	GetGeneratingIdle() bool
	GetCertifyingIdle() bool
}

type IdleSt struct {
	lock           *sync.RWMutex
	generatingIdle bool
	certifyingIdle bool
}

func NewIdleSt() *IdleSt {
	return &IdleSt{
		lock: new(sync.RWMutex),
	}
}

func (p *IdleSt) SetGeneratingIdle(st bool) {
	p.lock.Lock()
	p.generatingIdle = st
	p.lock.Unlock()
}

func (p *IdleSt) GetGeneratingIdle() bool {
	p.lock.RLock()
	value := p.generatingIdle
	p.lock.RUnlock()
	return value
}

func (p *IdleSt) SetCertifyingIdle(st bool) {
	p.lock.Lock()
	p.certifyingIdle = st
	p.lock.Unlock()
}

func (p *IdleSt) GetCertifyingIdle() bool {
	p.lock.RLock()
	value := p.certifyingIdle
	p.lock.RUnlock()
	return value
}
