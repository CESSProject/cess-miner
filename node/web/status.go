/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package web

import (
	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/node/common"
	"github.com/CESSProject/cess-miner/node/runstatus"
	"github.com/gin-gonic/gin"
)

type StatusHandler struct {
	runstatus.Runstatus
}

func NewStatusHandler(rs runstatus.Runstatus) *StatusHandler {
	return &StatusHandler{Runstatus: rs}
}

func (s *StatusHandler) RegisterRoutes(server *gin.Engine) {
	filegroup := server.Group("/status")
	filegroup.GET("", s.getStatus)
}

type StatusData struct {
	PID      int    `json:"pid"`
	Version  string `json:"version"`
	Cores    int    `json:"cores"`
	Endpoint string `json:"endpoint"`

	CurrentRpc       string `json:"current_rpc"`
	CurrentRpcStatus bool   `json:"current_rpc_status"`
	IsConnectingRpc  bool   `json:"is_connecting_rpc"`
	StartTime        string `json:"start_time"`

	State            string `json:"state"`
	SignatureAcc     string `json:"signature_acc"`
	StakingAcc       string `json:"staking_acc"`
	EarningsAcc      string `json:"earnings_acc"`
	DeclarationSpace uint64 `json:"declaration_space"`
	IdleSpace        uint64 `json:"idle_space"`
	ServiceSpace     uint64 `json:"service_space"`
	LockSpace        uint64 `json:"lock_space"`

	IdleChallenging    bool `json:"idle_challenging"`
	ServiceChallenging bool `json:"service_challenging"`

	GeneratingIdle bool `json:"generating_idle"`
	CertifyingIdle bool `json:"certifying_idle"`

	CheckingIdle bool `json:"checking_idle"`
}

func (s *StatusHandler) getStatus(c *gin.Context) {

	declaration_space, idle_space, service_space, locked_space := s.GetMinerSpaceInfo()

	var data = StatusData{
		PID:      s.GetPID(),
		Version:  configs.Version,
		Cores:    s.GetCpucores(),
		Endpoint: s.GetComAddr(),

		CurrentRpc:       s.GetCurrentRpc(),
		CurrentRpcStatus: s.GetCurrentRpcst(),
		IsConnectingRpc:  s.GetRpcConnecting(),
		StartTime:        s.GetStartTime(),

		State:        s.GetState(),
		SignatureAcc: s.GetSignAcc(),
		StakingAcc:   s.GetStakingAcc(),
		EarningsAcc:  s.GetEarningsAcc(),

		DeclarationSpace: declaration_space,
		IdleSpace:        idle_space,
		ServiceSpace:     service_space,
		LockSpace:        locked_space,

		IdleChallenging:    s.GetIdleChallenging(),
		ServiceChallenging: s.GetServiceChallenging(),

		GeneratingIdle: s.GetGeneratingIdle(),
		CertifyingIdle: s.GetCertifyingIdle(),

		CheckingIdle: s.GetCheckPois(),
	}

	c.JSON(200, common.RespType{
		Code: 200,
		Msg:  common.OK,
		Data: data,
	})
}
