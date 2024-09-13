/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package runstatus

const (
	St_Normal uint8 = iota
	St_Warning
	St_Error
)

type Runstatus interface {
	Rpcst
	Processst
	Challengest
	Minerst
	Idlest
}

type runstatus struct {
	*RpcSt
	*ProcessSt
	*ChallengeSt
	*MinerSt
	*IdleSt
}

var _ Runstatus = (*runstatus)(nil)

func NewRunstatus() Runstatus {
	return &runstatus{
		RpcSt:       NewRpcSt(),
		ProcessSt:   NewProcessSt(),
		ChallengeSt: NewChallengeSt(),
		MinerSt:     NewMinerSt(),
		IdleSt:      NewIdleSt(),
	}
}
