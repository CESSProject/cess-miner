package rpc

import (
	"cess-bucket/configs"
	. "cess-bucket/internal/logger"
	. "cess-bucket/internal/rpc/proto"
	"cess-bucket/tools"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/proto"
)

type MService struct {
}

// Init
func Rpc_Init() {
	if err := tools.CreatDirIfNotExist(configs.Confile.MinerData.MountedPath); err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
}

// Start websocket service.
// If an error occurs, it will exit immediately.
func Rpc_Main() {
	srv := NewServer()
	err := srv.Register(configs.RpcService_Local, MService{})
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
	err = http.ListenAndServe(":"+fmt.Sprintf("%d", configs.MinerServicePort), srv.WebsocketHandler([]string{"*"}))
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
}

// WritefileAction is used to handle scheduler service requests to upload files.
// The return code is 200 for success, non-200 for failure.
// The returned Msg indicates the result reason.
func (MService) WritefileAction(body []byte) (proto.Message, error) {
	var (
		err error
		t   int64
		b   FileUploadInfo
	)
	t = time.Now().Unix()
	Out.Sugar().Infof("[%v]Receive upload request", t)
	err = proto.Unmarshal(body, &b)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive upload request err:%v", t, err)
		return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
	}
	// Determine whether the storage path exists
	err = tools.CreatDirIfNotExist(configs.ServiceDir)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive upload request err:%v", t, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	fid := strings.Split(filepath.Base(b.FileId), ".")[0]
	fpath := filepath.Join(configs.ServiceDir, fid)
	if err = os.MkdirAll(fpath, os.ModeDir); err != nil {
		Out.Sugar().Infof("[%v]Receive upload request err:%v", t, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}

	// Save received file
	fii, err := os.OpenFile(filepath.Join(fpath, filepath.Base(b.FileId)), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive upload request err:%v", t, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	defer fii.Close()
	fii.Write(b.Data)
	Out.Sugar().Infof("[%v]Receive upload request suc:%v", t, filepath.Join(fpath, filepath.Base(b.FileId)))
	return &RespBody{Code: 200, Msg: "sucess", Data: nil}, nil
}

// ReadfileAction is used to handle scheduler service requests to download files.
// The return code is 200 for success, non-200 for failure.
// The returned Msg indicates the result reason.
func (MService) ReadfileAction(body []byte) (proto.Message, error) {
	var (
		err     error
		t       int64
		b       FileDownloadReq
		rtnData FileDownloadInfo
	)
	t = time.Now().Unix()
	Out.Sugar().Infof("[%v]Receive download request", t)
	err = proto.Unmarshal(body, &b)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive download request err:%v", t, err)
		return &RespBody{Code: 400, Msg: err.Error()}, nil
	}
	fid := strings.Split(b.FileId, ".")[0]
	fpath := filepath.Join(configs.ServiceDir, fid, b.FileId)
	_, err = os.Stat(fpath)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive download request err:%v", t, err)
		return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
	}
	// read file content
	buf, err := ioutil.ReadFile(fpath)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive download request err:%v", t, err)
		return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
	}
	// Calculate the number of slices
	slicesize, lastslicesize, num, err := cutDataRule(len(buf))
	if err != nil {
		Out.Sugar().Infof("[%v]Receive download request err:%v", t, err)
		return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
	}
	rtnData.FileId = b.FileId
	rtnData.Blocks = b.Blocks
	if b.Blocks+1 == int32(num) {
		rtnData.BlockSize = int32(lastslicesize)
		rtnData.Data = buf[len(buf)-lastslicesize:]
	} else {
		rtnData.BlockSize = int32(slicesize)
		rtnData.Data = buf[b.Blocks*int32(slicesize) : (b.Blocks+1)*int32(slicesize)]
	}
	rtnData.BlockNum = int32(num)
	rtnData_proto, err := proto.Marshal(&rtnData)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive download request err:%v", t, err)
		return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
	}
	Out.Sugar().Infof("[%v]Receive download request suc", t)
	return &RespBody{Code: 200, Msg: "success", Data: rtnData_proto}, nil
}

// Divide the size according to 2M
func cutDataRule(size int) (int, int, uint8, error) {
	if size <= 0 {
		return 0, 0, 0, errors.New("size is lt 0")
	}
	fmt.Println(size)
	num := size / (2 * 1024 * 1024)
	slicesize := size / (num + 1)
	tailsize := size - slicesize*(num+1)
	return slicesize, slicesize + tailsize, uint8(num) + 1, nil
}

//
func WriteData(cli *Client, service, method string, body []byte) ([]byte, error) {
	// dstip := "ws://" + tools.Base58Decoding(dst)
	// dstip = strings.Replace(dstip, " ", "", -1)
	req := &ReqMsg{
		Service: service,
		Method:  method,
		Body:    body,
	}
	// client, err := DialWebsocket(context.Background(), dstip, "")
	// if err != nil {
	// 	return nil, errors.Wrap(err, "DialWebsocket:")
	// }
	// defer client.Close()
	ctx, _ := context.WithTimeout(context.Background(), 90*time.Second)
	resp, err := cli.Call(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "Call err:")
	}

	var b RespBody
	err = proto.Unmarshal(resp.Body, &b)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal:")
	}
	if b.Code == 200 {
		return b.Data, nil
	}
	errstr := fmt.Sprintf("%d", b.Code)
	return nil, errors.New("return code:" + errstr)
}
