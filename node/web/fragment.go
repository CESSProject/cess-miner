/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package web

import (
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/CESSProject/cess-go-sdk/config"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/cess-miner/node/common"
	"github.com/CESSProject/cess-miner/node/workspace"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type FragmentHandler struct {
	*chain.ChainClient
	workspace.Workspace
}

func NewFragmentHandler(cli *chain.ChainClient, ws workspace.Workspace) *FragmentHandler {
	return &FragmentHandler{ChainClient: cli, Workspace: ws}
}

func (f *FragmentHandler) RegisterRoutes(server *gin.Engine) {
	filegroup := server.Group("/fragment")
	filegroup.PUT("", f.putfragment)
	filegroup.GET("", f.getfragment)
}

func (f *FragmentHandler) getfragment(c *gin.Context) {
	defer c.Request.Body.Close()
	fid := c.Request.Header.Get("Fid")
	fragment := c.Request.Header.Get("Fragment")

	if fid == "" || fragment == "" {
		c.JSON(http.StatusOK, common.RespType{
			Code: 400,
			Msg:  common.ERR_EmptyHashName,
		})
		return
	}

	if len(fid) != chain.FileHashLen || len(fragment) != chain.FileHashLen {
		c.JSON(http.StatusOK, common.RespType{
			Code: 400,
			Msg:  common.ERR_HashLength,
		})
		return
	}

	fragmentpath, err := f.findFragment(fid, fragment)
	if err != nil {
		c.JSON(http.StatusOK, common.RespType{
			Code: 404,
			Msg:  err.Error(),
		})
		return
	}

	fd, err := os.Open(fragmentpath)
	if err != nil {
		c.JSON(http.StatusOK, common.RespType{
			Code: 500,
			Msg:  common.ERR_SystemErr,
		})
		return
	}
	defer fd.Close()

	finfo, err := fd.Stat()
	if err != nil {
		c.JSON(http.StatusOK, common.RespType{
			Code: 500,
			Msg:  common.ERR_SystemErr,
		})
		return
	}
	c.DataFromReader(http.StatusOK, finfo.Size(), "application/octet-stream", fd, nil)
}

func (f *FragmentHandler) putfragment(c *gin.Context) {
	defer c.Request.Body.Close()
	fid := c.Request.Header.Get("Fid")

	err := os.MkdirAll(filepath.Join(f.GetTmpDir(), fid), 0755)
	if err != nil {
		c.JSON(http.StatusOK, common.RespType{
			Code: 500,
			Msg:  common.ERR_SystemErr,
		})
		return
	}

	fragment, fragmentpath, size, code, err := f.saveFormFile(c, fid)
	if err != nil {
		// n.Logput("err", clientIp+" saveFormFile: "+err.Error())
		c.JSON(http.StatusOK, common.RespType{
			Code: code,
			Msg:  err.Error(),
		})
		return
	}

	if size != config.FragmentSize {
		c.JSON(http.StatusOK, common.RespType{
			Code: 400,
			Msg:  common.ERR_FragmentSize,
		})
		return
	}

	ok, err := f.checkFragment(fid, fragment)
	if err != nil {
		// n.Logput("err", clientIp+" saveFormFile: "+err.Error())
		c.JSON(http.StatusOK, common.RespType{
			Code: 403,
			Msg:  common.ERR_RPCConnection,
		})
		return
	}

	if !ok {
		// n.Logput("err", clientIp+" saveFormFile: "+err.Error())
		c.JSON(http.StatusOK, common.RespType{
			Code: 403,
			Msg:  common.ERR_FragmentNotMatchFid,
		})
		return
	}

	err = os.MkdirAll(filepath.Join(f.GetReportDir(), fid), 0755)
	if err != nil {
		c.JSON(http.StatusOK, common.RespType{
			Code: 500,
			Msg:  common.ERR_SystemErr,
		})
		return
	}

	err = os.Rename(fragmentpath, filepath.Join(f.GetReportDir(), fid, fragment))
	if err != nil {
		c.JSON(http.StatusOK, common.RespType{
			Code: 500,
			Msg:  common.ERR_SystemErr,
		})
		return
	}
	c.JSON(http.StatusOK, common.RespType{
		Code: http.StatusOK,
		Msg:  common.OK,
	})
}

func (f *FragmentHandler) findFragment(fid, fragment string) (string, error) {
	fragmentpath := filepath.Join(f.GetFileDir(), fid, fragment)
	_, err := os.Stat(fragmentpath)
	if err == nil {
		return fragmentpath, nil
	}

	fragmentpath = filepath.Join(f.GetReportDir(), fid, fragment)
	_, err = os.Stat(fragmentpath)
	if err == nil {
		return fragmentpath, nil
	}

	return "", errors.New(common.ERR_NotFound)
}

func (f *FragmentHandler) checkFragment(fid, fragment string) (bool, error) {
	forder, err := f.QueryDealMap(fid, -1)
	if err != nil {
		if !errors.Is(err, chain.ERR_RPC_EMPTY_VALUE) {
			return false, err
		}
		return false, nil
	}

	for i := 0; i < len(forder.SegmentList); i++ {
		for j := 0; j < len(forder.SegmentList[i].FragmentHash); j++ {
			if string(forder.SegmentList[i].FragmentHash[j][:]) == fragment {
				return true, nil
			}
		}
	}
	return false, nil
}

func (f *FragmentHandler) saveFormFile(c *gin.Context, fid string) (string, string, int64, int, error) {
	tmpPath := ""
	var err error
	var uid uuid.UUID
	for {
		uid, err = uuid.NewV7FromReader(rand.Reader)
		if err != nil {
			time.Sleep(time.Millisecond)
			continue
		}
		tmpPath = filepath.Join(f.GetTmpDir(), fid, uid.String())
		_, err = os.Stat(tmpPath)
		if err != nil {
			break
		}
	}
	fd, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return "", "", 0, http.StatusInternalServerError, err
	}
	defer fd.Close()
	formfile, _, err := c.Request.FormFile("file")
	if err != nil {
		return "", "", 0, http.StatusBadRequest, err
	}

	_, err = io.Copy(fd, formfile)
	if err != nil {
		return "", "", 0, http.StatusBadRequest, err
	}

	fragment, err := sutils.CalcFileSHA256(fd)
	if err != nil {
		return "", "", 0, http.StatusInternalServerError, err
	}

	finfo, err := fd.Stat()
	if err != nil {
		return "", "", 0, http.StatusInternalServerError, err
	}

	err = fd.Sync()
	if err != nil {
		return "", "", 0, http.StatusInternalServerError, err
	}

	return fragment, tmpPath, finfo.Size(), http.StatusOK, nil
}