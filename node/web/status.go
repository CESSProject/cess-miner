/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package web

import (
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
	PID   int    `json:"pid"`
	Cores int    `json:"cores"`
	Addr  string `json:"addr"`

	CurrentRpc        string `json:"current_rpc"`
	CurrentRpcStatus  bool   `json:"current_rpc_status"`
	IsConnectingRpc   bool   `json:"is_connecting_rpc"`
	LastConnectedTime string `json:"last_connected_time"`

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
}

func (s *StatusHandler) getStatus(c *gin.Context) {

	declaration_space, idle_space, service_space, locked_space := s.GetMinerSpaceInfo()

	var data = StatusData{
		PID:   s.GetPID(),
		Cores: s.GetCpucores(),
		Addr:  s.GetComAddr(),

		CurrentRpc:        s.GetCurrentRpc(),
		CurrentRpcStatus:  s.GetCurrentRpcst(),
		IsConnectingRpc:   s.GetRpcConnecting(),
		LastConnectedTime: s.GetLastConnectedTime(),

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
	}

	c.JSON(200, common.RespType{
		Code: 200,
		Msg:  common.OK,
		Data: data,
	})

	// msg += fmt.Sprintf("Calculate Tag: %v\n", n.GetCalcTagFlag())

	// msg += fmt.Sprintf("Report file: %v\n", n.GetReportFileFlag())

	// msg += fmt.Sprintf("Generate idle: %v\n", n.GetGenIdleFlag())

	// msg += fmt.Sprintf("Report idle: %v\n", n.GetAuthIdleFlag())

	// msg += fmt.Sprintf("Calc idle challenge: %v\n", n.GetIdleChallengeFlag())

	// msg += fmt.Sprintf("Calc service challenge: %v\n", n.GetServiceChallengeFlag())

	// msg += fmt.Sprintf("Receiving data: %v\n", n.GetReceiveFlag())

	// msg += fmt.Sprintf("Cpu usage: %.2f%%\n", getCpuUsage(int32(n.GetPID())))

	// msg += fmt.Sprintf("Memory usage: %d", getMemUsage())
}
