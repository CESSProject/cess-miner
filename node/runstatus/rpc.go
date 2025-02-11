/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package runstatus

import (
	"sync"
)

type Rpcst interface {
	SetCurrentRpc(rpc string)
	SetStartTime(t string)
	SetCurrentRpcst(st bool)
	SetRpcConnecting(st bool)

	GetCurrentRpc() string
	GetStartTime() string
	GetCurrentRpcst() bool
	GetRpcConnecting() bool
	GetAndSetRpcConnecting() bool
}

type RpcSt struct {
	lock           *sync.RWMutex
	currentRpc     string
	startTime      string
	currentRpcSt   bool
	isitconnecting bool
}

func NewRpcSt() *RpcSt {
	return &RpcSt{
		lock: new(sync.RWMutex),
	}
}

func (r *RpcSt) SetCurrentRpc(rpc string) {
	r.lock.Lock()
	r.currentRpc = rpc
	r.lock.Unlock()
}

func (r *RpcSt) GetCurrentRpc() string {
	r.lock.RLock()
	value := r.currentRpc
	r.lock.RUnlock()
	return value
}

func (r *RpcSt) SetStartTime(t string) {
	r.lock.Lock()
	r.startTime = t
	r.lock.Unlock()
}

func (r *RpcSt) GetStartTime() string {
	r.lock.RLock()
	value := r.startTime
	r.lock.RUnlock()
	return value
}

func (r *RpcSt) SetCurrentRpcst(st bool) {
	r.lock.Lock()
	r.currentRpcSt = st
	r.lock.Unlock()
}

func (r *RpcSt) GetCurrentRpcst() bool {
	r.lock.RLock()
	value := r.currentRpcSt
	r.lock.RUnlock()
	return value
}

func (r *RpcSt) SetRpcConnecting(st bool) {
	r.lock.Lock()
	r.isitconnecting = st
	r.lock.Unlock()
}

func (r *RpcSt) GetRpcConnecting() bool {
	r.lock.RLock()
	value := r.isitconnecting
	r.lock.RUnlock()
	return value
}

func (r *RpcSt) GetAndSetRpcConnecting() bool {
	r.lock.Lock()
	if r.isitconnecting {
		r.lock.Unlock()
		return true
	}
	r.isitconnecting = true
	r.lock.Unlock()
	return false
}
