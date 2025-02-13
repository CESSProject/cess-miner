/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package web

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/node/common"
	"github.com/CESSProject/cess-miner/node/runstatus"
	"github.com/CESSProject/cess-miner/node/workspace"
	"github.com/gin-gonic/gin"
)

type StatusHandler struct {
	runstatus runstatus.Runstatus
	wspace    workspace.Workspace
}

func NewStatusHandler(rs runstatus.Runstatus, wspace workspace.Workspace) *StatusHandler {
	return &StatusHandler{runstatus: rs, wspace: wspace}
}

func (s *StatusHandler) RegisterRoutes(server *gin.Engine) {
	filegroup := server.Group("/status")
	filegroup.GET("", s.getStatus)
	filegroup.GET("/:file", s.getfile)
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

	declaration_space, idle_space, service_space, locked_space := s.runstatus.GetMinerSpaceInfo()

	var data = StatusData{
		PID:      s.runstatus.GetPID(),
		Version:  configs.Version,
		Cores:    s.runstatus.GetCpucores(),
		Endpoint: s.runstatus.GetComAddr(),

		CurrentRpc:       s.runstatus.GetCurrentRpc(),
		CurrentRpcStatus: s.runstatus.GetCurrentRpcst(),
		IsConnectingRpc:  s.runstatus.GetRpcConnecting(),
		StartTime:        s.runstatus.GetStartTime(),

		State:        s.runstatus.GetState(),
		SignatureAcc: s.runstatus.GetSignAcc(),
		StakingAcc:   s.runstatus.GetStakingAcc(),
		EarningsAcc:  s.runstatus.GetEarningsAcc(),

		DeclarationSpace: declaration_space,
		IdleSpace:        idle_space,
		ServiceSpace:     service_space,
		LockSpace:        locked_space,

		IdleChallenging:    s.runstatus.GetIdleChallenging(),
		ServiceChallenging: s.runstatus.GetServiceChallenging(),

		GeneratingIdle: s.runstatus.GetGeneratingIdle(),
		CertifyingIdle: s.runstatus.GetCertifyingIdle(),

		CheckingIdle: s.runstatus.GetCheckPois(),
	}

	c.JSON(200, common.RespType{
		Code: 200,
		Msg:  common.OK,
		Data: data,
	})
}

func (s *StatusHandler) getfile(c *gin.Context) {
	filename := c.Param("file")

	switch filename {
	case "idle_proof":
		fd, err := os.Open(s.wspace.GetIdleProve())
		if err != nil {
			c.JSON(200, common.RespType{
				Code: 500,
				Msg:  common.ERR_SystemErr,
				Data: err.Error(),
			})
			return
		}
		defer fd.Close()
		fs, _ := fd.Stat()
		c.DataFromReader(http.StatusOK, fs.Size(), "text/plain", fd, nil)
	case "service_proof":
		fd, err := os.Open(s.wspace.GetServiceProve())
		if err != nil {
			c.JSON(200, common.RespType{
				Code: 500,
				Msg:  common.ERR_SystemErr,
				Data: err.Error(),
			})
			return
		}
		defer fd.Close()
		fs, _ := fd.Stat()
		c.DataFromReader(http.StatusOK, fs.Size(), "text/plain", fd, nil)
	case "ichal":
		fd, err := os.Open(filepath.Join(s.wspace.GetLogDir(), "ichal.log"))
		if err != nil {
			c.JSON(200, common.RespType{
				Code: 500,
				Msg:  common.ERR_SystemErr,
				Data: err.Error(),
			})
			return
		}
		defer fd.Close()
		fs, _ := fd.Stat()
		c.DataFromReader(http.StatusOK, fs.Size(), "text/plain", fd, nil)
	case "schal":
		fd, err := os.Open(filepath.Join(s.wspace.GetLogDir(), "schal.log"))
		if err != nil {
			c.JSON(200, common.RespType{
				Code: 500,
				Msg:  common.ERR_SystemErr,
				Data: err.Error(),
			})
			return
		}
		defer fd.Close()
		fs, _ := fd.Stat()
		c.DataFromReader(http.StatusOK, fs.Size(), "text/plain", fd, nil)
	case "pnc":
		fd, err := os.Open(filepath.Join(s.wspace.GetLogDir(), "panic.log"))
		if err != nil {
			c.JSON(200, common.RespType{
				Code: 500,
				Msg:  common.ERR_SystemErr,
				Data: err.Error(),
			})
			return
		}
		defer fd.Close()
		fs, _ := fd.Stat()
		c.DataFromReader(http.StatusOK, fs.Size(), "text/plain", fd, nil)
	case "space":
		fd, err := os.Open(filepath.Join(s.wspace.GetLogDir(), "space.log"))
		if err != nil {
			c.JSON(200, common.RespType{
				Code: 500,
				Msg:  common.ERR_SystemErr,
				Data: err.Error(),
			})
			return
		}
		defer fd.Close()
		fs, _ := fd.Stat()
		c.DataFromReader(http.StatusOK, fs.Size(), "text/plain", fd, nil)
	case "log":
		fd, err := os.Open(filepath.Join(s.wspace.GetLogDir(), "log.log"))
		if err != nil {
			c.JSON(200, common.RespType{
				Code: 500,
				Msg:  common.ERR_SystemErr,
				Data: fmt.Sprintf("[%s] %v", filepath.Join(s.wspace.GetLogDir(), "log.log"), err),
			})
			return
		}
		defer fd.Close()
		fs, _ := fd.Stat()
		c.DataFromReader(http.StatusOK, fs.Size(), "text/plain", fd, nil)
	default:
		c.JSON(200, common.RespType{
			Code: 404,
			Msg:  common.OK,
			Data: "not found",
		})
	}
}
