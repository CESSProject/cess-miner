/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package web

import (
	"github.com/gin-gonic/gin"
)

type StatusHandler struct {
}

func NewStatusHandler() *StatusHandler {
	return &StatusHandler{}
}

func (s *StatusHandler) RegisterRoutes(server *gin.Engine) {
	filegroup := server.Group("/status")
	filegroup.GET("", s.GetStatusHandle)
}

// getStatusHandle
func (s *StatusHandler) GetStatusHandle(c *gin.Context) {
	// var msg string

	// msg += fmt.Sprintf("Process ID: %d\n", n.GetPID())

	// msg += fmt.Sprintf("Miner Signature Account: %s\n", n.GetMinerSignatureAcc())

	// msg += fmt.Sprintf("Miner State: %s\n", n.GetMinerState())

	// if n.GetChainStatus() {
	// 	msg += fmt.Sprintf("RPC Connection: [ok] %v\n", n.GetCurrentRpc())
	// } else {
	// 	msg += fmt.Sprintf("RPC Connection: [fail] %v\n", n.GetCurrentRpc())
	// }
	// msg += fmt.Sprintf("Last reconnection: %v\n", n.GetLastReconnectRpcTime())

	// msg += fmt.Sprintf("Calculate Tag: %v\n", n.GetCalcTagFlag())

	// msg += fmt.Sprintf("Report file: %v\n", n.GetReportFileFlag())

	// msg += fmt.Sprintf("Generate idle: %v\n", n.GetGenIdleFlag())

	// msg += fmt.Sprintf("Report idle: %v\n", n.GetAuthIdleFlag())

	// msg += fmt.Sprintf("Calc idle challenge: %v\n", n.GetIdleChallengeFlag())

	// msg += fmt.Sprintf("Calc service challenge: %v\n", n.GetServiceChallengeFlag())

	// msg += fmt.Sprintf("Receiving data: %v\n", n.GetReceiveFlag())

	// msg += fmt.Sprintf("Cpu usage: %.2f%%\n", getCpuUsage(int32(n.GetPID())))

	// msg += fmt.Sprintf("Memory usage: %d", getMemUsage())

	// c.Data(200, "application/octet-stream", []byte(msg))
}
