/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package web

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/CESSProject/cess-go-sdk/chain"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/cess-miner/node/common"
	"github.com/CESSProject/cess-miner/node/logger"
	"github.com/CESSProject/cess-miner/node/workspace"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type FragmentHandler struct {
	chain.Chainer
	workspace.Workspace
	logger.Logger
}

func NewFragmentHandler(cli chain.Chainer, ws workspace.Workspace, lg logger.Logger) *FragmentHandler {
	return &FragmentHandler{Chainer: cli, Workspace: ws, Logger: lg}
}

func (f *FragmentHandler) RegisterRoutes(server *gin.Engine) {
	fragmentgroup := server.Group("/fragment")
	fragmentgroup.Use(func(ctx *gin.Context) {
		ok, err := VerifySignature(ctx)
		if !ok || err != nil {
			ctx.AbortWithStatusJSON(403, common.RespType{
				Code: 403,
				Msg:  common.ERR_InvalidSignature,
			})
			return
		}
		ctx.Next()
	})
	fragmentgroup.PUT("", f.putfragment)
	fragmentgroup.GET("", f.getfragment)
}

func (f *FragmentHandler) getfragment(c *gin.Context) {
	defer c.Request.Body.Close()
	fid := c.Request.Header.Get(common.Header_Fid)
	fragment := c.Request.Header.Get(common.Header_Fragment)
	rangedata := c.Request.Header.Get(common.Header_Range)
	clientIp := c.Request.Header.Get(common.Header_X_Forwarded_For)
	if clientIp == "" {
		clientIp = c.ClientIP()
	}
	if fid == "" || fragment == "" {
		f.Getf("err", clientIp+" fid or fragment is empty")
		common.ReturnJSON(c, 400, common.ERR_EmptyHashName, nil)
		return
	}

	if len(fid) != chain.FileHashLen || len(fragment) != chain.FileHashLen {
		f.Getf("err", clientIp+" fid or fragment is invalid")
		common.ReturnJSON(c, 400, common.ERR_HashLength, nil)
		return
	}

	fragmentpath, err := f.findFragment(fid, fragment)
	if err != nil {
		f.Getf("err", clientIp+" not found")
		common.ReturnJSON(c, 404, common.ERR_NotFound, nil)
		return
	}

	if rangedata != "" {
		err = ReturnFileRangeStream(c, rangedata, fragmentpath)
		if err != nil {
			f.Getf("err", clientIp+" ReturnFileRangeStream: "+err.Error())
		}
		return
	}

	c.File(fragmentpath)
}

func (f *FragmentHandler) putfragment(c *gin.Context) {
	defer c.Request.Body.Close()
	fid := c.Request.Header.Get(common.Header_Fid)
	fragment := c.Request.Header.Get(common.Header_Fragment)
	clientIp := c.Request.Header.Get(common.Header_X_Forwarded_For)
	if clientIp == "" {
		clientIp = c.ClientIP()
	}

	if fragment != "" {
		_, err := f.findFragment(fid, fragment)
		if err == nil {
			f.Putf("err", clientIp+" repeat upload: "+fid+" "+fragment)
			common.ReturnJSON(c, 200, common.OK, nil)
			return
		}
	}

	err := os.MkdirAll(filepath.Join(f.GetTmpDir(), fid), 0755)
	if err != nil {
		f.Putf("err", clientIp+" mk tmp dir: "+err.Error())
		common.ReturnJSON(c, 500, common.ERR_SystemErr, nil)
		return
	}

	if fragment == chain.ZeroFileHash_8M {
		err = os.MkdirAll(filepath.Join(f.GetReportDir(), fid), 0755)
		if err != nil {
			f.Putf("err", clientIp+" mk report dir: "+err.Error())
			common.ReturnJSON(c, 500, common.ERR_SystemErr, nil)
			return
		}

		err = sutils.WriteBufToFile(make([]byte, chain.FragmentSize), filepath.Join(f.GetReportDir(), fid, fragment))
		if err != nil {
			f.Putf("err", clientIp+" WriteBufToFile(ZeroFileHash_8M): "+err.Error())
			common.ReturnJSON(c, 500, common.ERR_SystemErr, nil)
			return
		}
		f.Putf("err", clientIp+" upload ZeroFileHash_8M suc "+fid)
		common.ReturnJSON(c, 200, common.OK, nil)
		return
	}

	fragmentpath, size, code, err := f.saveFormFile(c, fid)
	if err != nil {
		f.Putf("err", clientIp+" saveFormFile: "+err.Error())
		common.ReturnJSON(c, code, err.Error(), nil)
		return
	}

	fragment_upload, err := sutils.CalcPathSHA256(fragmentpath)
	if err != nil {
		f.Putf("err", clientIp+" CalcPathSHA256: "+err.Error())
		common.ReturnJSON(c, 500, common.ERR_SystemErr, nil)
		return
	}

	if size != chain.FragmentSize {
		f.Putf("err", clientIp+" fragment size not equal 8M")
		common.ReturnJSON(c, 400, common.ERR_FragmentSize, nil)
		return
	}

	if fragment != "" {
		if fragment_upload != fragment {
			common.ReturnJSON(c, 400, common.ERR_FragmentHash, nil)
			return
		}
	}

	ok, err := f.checkFragment(fid, fragment)
	if err != nil {
		f.Putf("err", clientIp+" checkFragment: "+err.Error())
		common.ReturnJSON(c, 403, common.ERR_RPCConnection, nil)
		return
	}

	if !ok {
		f.Putf("err", clientIp+" checkFragment false")
		common.ReturnJSON(c, 400, common.ERR_FragmentNotMatchFid, nil)
		return
	}

	err = os.MkdirAll(filepath.Join(f.GetReportDir(), fid), 0755)
	if err != nil {
		f.Putf("err", clientIp+" mk report dir: "+err.Error())
		common.ReturnJSON(c, 500, common.ERR_SystemErr, nil)
		return
	}

	err = os.Rename(fragmentpath, filepath.Join(f.GetReportDir(), fid, fragment))
	if err != nil {
		f.Putf("err", clientIp+" Rename: "+err.Error())
		common.ReturnJSON(c, 500, common.ERR_SystemErr, nil)
		return
	}
	common.ReturnJSON(c, 200, common.OK, nil)
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

func (f *FragmentHandler) saveFormFile(c *gin.Context, fid string) (string, int64, int, error) {
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
		return "", 0, http.StatusInternalServerError, errors.Wrapf(err, "[OpenFile]")
	}
	defer fd.Close()
	formfile, _, err := c.Request.FormFile("file")
	if err != nil {
		return "", 0, http.StatusBadRequest, errors.Wrapf(err, "[FormFile]")
	}

	_, err = io.Copy(fd, formfile)
	if err != nil {
		return "", 0, http.StatusBadRequest, errors.Wrapf(err, "[Copy]")
	}

	finfo, err := fd.Stat()
	if err != nil {
		return "", 0, http.StatusInternalServerError, errors.Wrapf(err, "[Stat]")
	}

	err = fd.Sync()
	if err != nil {
		return "", 0, http.StatusInternalServerError, errors.Wrapf(err, "[Sync]")
	}

	return tmpPath, finfo.Size(), http.StatusOK, nil
}

func VerifySignature(ctx *gin.Context) (bool, error) {
	account := ctx.Request.Header.Get(common.Header_Account)
	message := ctx.Request.Header.Get(common.Header_Message)
	signature := ctx.Request.Header.Get(common.Header_Signature)
	sign, err := hex.DecodeString(strings.TrimPrefix(signature, "0x"))
	if err != nil {
		return false, err
	}
	ok, _ := sutils.VerifySR25519WithPublickey(message, sign, account)
	if !ok {
		return sutils.VerifyPolkadotJsHexSign(account, message, signature)
	}
	return ok, nil
}

func ReturnFileRangeStream(c *gin.Context, rng string, file string) error {
	f, err := os.Open(file)
	if err != nil {
		common.ReturnJSON(c, 500, common.ERR_SystemErr, nil)
		return fmt.Errorf("stat file err: %v", err)
	}
	defer f.Close()

	fstat, err := f.Stat()
	if err != nil {
		common.ReturnJSON(c, 500, common.ERR_SystemErr, nil)
		return fmt.Errorf("stat file err: %v", err)
	}

	ranges := strings.Split(rng, "=")
	if len(ranges) != 2 || ranges[0] != "bytes" {
		ranges = strings.Split(rng, " ")
		if len(ranges) != 2 || ranges[0] != "bytes" {
			common.ReturnJSON(c, 416, common.ERR_InvalidRangeValue, nil)
			return fmt.Errorf("invalid range request: %s", rng)
		}
	}

	rangeParts := strings.Split(ranges[1], "/")
	if len(rangeParts) != 2 {
		common.ReturnJSON(c, 416, common.ERR_InvalidRangeValue, nil)
		return fmt.Errorf("invalid range request: %s", rng)
	}

	// total, err := strconv.ParseInt(rangeParts[1], 10, 64)
	// if err != nil {
	// 	common.ReturnJSON(c, 416, common.ERR_InvalidRangeTotal, nil)
	// 	return fmt.Errorf("invalid range total: %s", rng)
	// }

	rangeParts = strings.Split(rangeParts[0], "-")
	if len(rangeParts) != 2 {
		common.ReturnJSON(c, 416, common.ERR_InvalidRangeValue, nil)
		return fmt.Errorf("invalid range request: %s", rng)
	}

	start, err := strconv.ParseInt(rangeParts[0], 10, 64)
	if err != nil || start < 0 {
		common.ReturnJSON(c, 416, common.ERR_InvalidRangeStart, nil)
		return fmt.Errorf("invalid range start: %s", rng)
	}

	end, err := strconv.ParseInt(rangeParts[1], 10, 64)
	if err != nil || end < start || end > fstat.Size() {
		common.ReturnJSON(c, 416, common.ERR_InvalidRangeEnd, nil)
		return fmt.Errorf("invalid range end: %s", rng)
	}

	_, err = f.Seek(start, io.SeekStart)
	if err != nil {
		common.ReturnJSON(c, 500, common.ERR_SystemErr, nil)
		return fmt.Errorf("f.seek: %v", err)
	}

	var buf = make([]byte, end-start)
	_, err = f.Read(buf)
	if err != nil {
		common.ReturnJSON(c, 500, common.ERR_SystemErr, nil)
		return fmt.Errorf("f.seek: %v", err)
	}
	c.Data(206, "application/octet-stream", buf)
	return nil
}
