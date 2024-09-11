/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package web

import "github.com/gin-gonic/gin"

type Handler struct {
	*FileHandler
	*StatusHandler
}

func NewHandler() *Handler {
	return &Handler{
		FileHandler:   NewFileHandler(),
		StatusHandler: NewStatusHandler(),
	}
}

func (h *Handler) RegisterRoutes(server *gin.Engine) {
	h.FileHandler.RegisterRoutes(server)
	h.StatusHandler.RegisterRoutes(server)
}
