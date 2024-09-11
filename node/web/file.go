/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package web

import "github.com/gin-gonic/gin"

type FileHandler struct {
}

func NewFileHandler() *FileHandler {
	return &FileHandler{}
}

func (f *FileHandler) RegisterRoutes(server *gin.Engine) {
	filegroup := server.Group("/file")
	filegroup.POST("/upload", f.Upload)
	filegroup.POST("/download", f.Download)
}

func (h *FileHandler) Upload(ctx *gin.Context) {

}

func (h *FileHandler) Download(ctx *gin.Context) {

}
