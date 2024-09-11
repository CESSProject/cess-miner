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
	PID   int `json:"pid"`
	Cores int `json:"cores"`

	CurrentRpc        string `json:"current_rpc"`
	CurrentRpcSt      bool   `json:"current_rpc_st"`
	IsConnecting      bool   `json:"is_connecting"`
	LastConnectedTime string `json:"last_connected_time"`

	State        string `json:"state"`
	SignatureAcc string `json:"signature_acc"`
	StakingAcc   string `json:"staking_acc"`
	EarningsAcc  string `json:"earnings_acc"`
}

func (s *StatusHandler) getStatus(c *gin.Context) {

	var data = StatusData{
		PID:   s.GetPID(),
		Cores: s.GetCpucores(),

		CurrentRpc:        s.GetCurrentRpc(),
		CurrentRpcSt:      s.GetCurrentRpcst(),
		IsConnecting:      s.GetRpcConnecting(),
		LastConnectedTime: s.GetLastConnectedTime(),

		State:        s.GetState(),
		SignatureAcc: s.GetSignAcc(),
		StakingAcc:   s.GetStakingAcc(),
		EarningsAcc:  s.GetEarningsAcc(),
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
