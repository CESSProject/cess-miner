/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package web

import (
	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/CESSProject/cess-miner/node/runstatus"
	"github.com/CESSProject/cess-miner/node/workspace"
	"github.com/CESSProject/cess-miner/pkg/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	*FragmentHandler
	*StatusHandler
}

func NewHandler(cli chain.Chainer, ws workspace.Workspace, rs runstatus.Runstatus, lg logger.Logger) *Handler {
	return &Handler{
		FragmentHandler: NewFragmentHandler(cli, ws, lg),
		StatusHandler:   NewStatusHandler(rs),
	}
}

func (h *Handler) RegisterRoutes(server *gin.Engine) {
	h.FragmentHandler.RegisterRoutes(server)
	h.StatusHandler.RegisterRoutes(server)
}
