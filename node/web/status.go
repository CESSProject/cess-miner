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
	"strconv"

	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/node/common"
	"github.com/CESSProject/cess-miner/node/runstatus"
	"github.com/CESSProject/cess-miner/node/workspace"
	"github.com/CESSProject/cess-miner/pkg/utils"
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
	server.GET("/status", s.getStatus)
	server.GET("/file/:name", s.getfile)
	server.GET("/file/list", s.getfilelist)
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

func (s *StatusHandler) getfilelist(c *gin.Context) {
	filetype, ok := c.GetQuery("path")
	if !ok {
		filetype = "/"
	}

	files, err := utils.DirFileDetail(filepath.Join(s.wspace.GetRootDir(), filetype), 0)
	if err != nil {
		c.JSON(200, common.RespType{
			Code: 500,
			Msg:  err.Error(),
		})
		return
	}
	c.JSON(200, common.RespType{
		Code: 200,
		Msg:  common.OK,
		Data: files,
	})
}

func (s *StatusHandler) getfile(c *gin.Context) {
	fp := c.Param("path")
	headsize := 0
	tailsize := 0
	var err error

	download, _ := c.GetQuery("download")
	headsizestr, ok := c.GetQuery("head")
	if ok {
		headsize, err = strconv.Atoi(headsizestr)
		if err != nil {
			c.JSON(200, common.RespType{
				Code: 400,
				Msg:  "invalid head number",
				Data: err.Error(),
			})
			return
		}
	}
	tailsizestr, ok := c.GetQuery("tail")
	if ok {
		tailsize, err = strconv.Atoi(tailsizestr)
		if err != nil {
			c.JSON(200, common.RespType{
				Code: 400,
				Msg:  "invalid tail number",
				Data: err.Error(),
			})
			return
		}
	}

	fstat, err := os.Stat(filepath.Join(s.wspace.GetRootDir(), fp))
	if err != nil {
		c.JSON(200, common.RespType{
			Code: 500,
			Msg:  common.ERR_SystemErr,
			Data: err.Error(),
		})
		return
	}
	fd, err := os.Open(filepath.Join(s.wspace.GetRootDir(), fp))
	if err != nil {
		c.JSON(200, common.RespType{
			Code: 500,
			Msg:  common.ERR_SystemErr,
			Data: err.Error(),
		})
		return
	}
	defer fd.Close()

	if headsize > 0 {
		if headsize > int(fstat.Size()) {
			headsize = int(fstat.Size())
		}
		buf := make([]byte, headsize)
		n, err := fd.Read(buf)
		if err != nil {
			c.JSON(200, common.RespType{
				Code: 500,
				Msg:  common.ERR_SystemErr,
				Data: err.Error(),
			})
			return
		}
		c.JSON(200, common.RespType{
			Code: 200,
			Msg:  fmt.Sprintf("size: %d", n),
			Data: string(buf),
		})
		return
	}

	if tailsize > 0 {
		if tailsize >= int(fstat.Size()) {
			tailsize = int(fstat.Size())
			buf := make([]byte, tailsize)
			n, err := fd.Read(buf)
			if err != nil {
				c.JSON(200, common.RespType{
					Code: 500,
					Msg:  common.ERR_SystemErr,
					Data: err.Error(),
				})
				return
			}
			c.JSON(200, common.RespType{
				Code: 200,
				Msg:  fmt.Sprintf("size: %d", n),
				Data: string(buf),
			})
			return
		} else {
			_, err = fd.Seek(fstat.Size()-int64(tailsize), 0)
			if err != nil {
				c.JSON(200, common.RespType{
					Code: 500,
					Msg:  common.ERR_SystemErr,
					Data: err.Error(),
				})
				return
			}
			buf := make([]byte, tailsize)
			n, err := fd.Read(buf)
			if err != nil {
				c.JSON(200, common.RespType{
					Code: 500,
					Msg:  common.ERR_SystemErr,
					Data: err.Error(),
				})
				return
			}
			c.JSON(200, common.RespType{
				Code: 200,
				Msg:  fmt.Sprintf("size: %d", n),
				Data: string(buf),
			})
			return
		}
	}

	if download != "" {
		c.DataFromReader(http.StatusOK, fstat.Size(), "application/octet-stream", fd, nil)
	} else {
		c.DataFromReader(http.StatusOK, fstat.Size(), "text/plain", fd, nil)
	}
	return

}
