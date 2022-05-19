package rpc

import (
	"cess-bucket/configs"
	. "cess-bucket/configs"
	"cess-bucket/internal/chain"
	. "cess-bucket/internal/logger"
	"cess-bucket/internal/pt"

	. "cess-bucket/internal/rpc/proto"
	"cess-bucket/tools"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"

	"google.golang.org/protobuf/proto"
)

type MService struct {
}

// Init
func Rpc_Init() {
	if err := tools.CreatDirIfNotExist(C.MountedPath); err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
}

// Start websocket service.
// If an error occurs, it will exit immediately.
func Rpc_Main() {
	srv := NewServer()
	err := srv.Register(RpcService_Local, MService{})
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
	err = http.ListenAndServe(":"+fmt.Sprintf("%d", C.ServicePort), srv.WebsocketHandler([]string{"*"}))
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
}

// Writefile is used to receive files uploaded by the scheduling service.
// The return code is 200 for success, non-200 for failure.
// The returned Msg indicates the result reason.
func (MService) WritefileAction(body []byte) (proto.Message, error) {
	var (
		err error
		b   PutFileToBucket
	)
	//Generate a random number to track the log record of this request
	t := tools.RandomInRange(100000000, 999999999)
	Out.Sugar().Infof("[T:%v]Write file request.....", t)

	//Parse the requested data
	err = proto.Unmarshal(body, &b)
	if err != nil {
		Out.Sugar().Infof("[T:%v][%v]Err:%v", t, len(body), err)
		return &RespBody{Code: Code_400, Msg: err.Error(), Data: nil}, nil
	}

	//Determine whether the data base directory exists
	err = tools.CreatDirIfNotExist(FilesDir)
	if err != nil {
		Out.Sugar().Infof("[T:%v]Err:%v", t, err)
		return &RespBody{Code: Code_500, Msg: err.Error(), Data: nil}, nil
	}

	//Get the suffix of fileid
	ext := filepath.Ext(b.FileId)
	if ext == "" {
		Out.Sugar().Infof("[T:%v][%v]Err:Invalid dupl id", t, b.FileId)
		return &RespBody{Code: Code_400, Msg: "Invalid dupl id", Data: nil}, nil
	}

	//get fileid
	fid := strings.TrimSuffix(b.FileId, ext)

	//get file path
	fpath := filepath.Join(FilesDir, fid)
	_, err = os.Stat(fpath)
	if err != nil {
		err = os.MkdirAll(fpath, os.ModeDir)
		if err != nil {
			Out.Sugar().Infof("[T:%v]Err:%v", t, err)
			return &RespBody{Code: Code_500, Msg: err.Error(), Data: nil}, nil
		}
	}
	filefullpath := filepath.Join(fpath, b.FileId)

	if b.BlockIndex == 0 {
		os.Remove(filefullpath)
	}

	//save to local file
	fii, err := os.OpenFile(filefullpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		Out.Sugar().Infof("[T:%v]Err:%v", t, err)
		return &RespBody{Code: Code_500, Msg: err.Error(), Data: nil}, nil
	}
	defer fii.Close()
	fii.Write(b.BlockData)

	//flush to disk
	err = fii.Sync()
	if err != nil {
		Out.Sugar().Infof("[T:%v]Err:%v", t, err)
		return &RespBody{Code: Code_500, Msg: err.Error(), Data: nil}, nil
	}
	Out.Sugar().Infof("[T:%v]Suc:[%v] [%v]", t, b.FileId, b.BlockIndex)
	return &RespBody{Code: Code_200, Msg: "success", Data: nil}, nil
}

// Readfile is used to return file information to the scheduling service.
// The return code is 200 for success, non-200 for failure.
// The returned Msg indicates the result reason.
func (MService) ReadfileAction(body []byte) (proto.Message, error) {
	var (
		err     error
		b       FileDownloadReq
		rtnData FileDownloadInfo
	)

	//Generate a random number to track the log record of this request
	t := tools.RandomInRange(100000000, 999999999)
	Out.Sugar().Infof("[T:%v]Read file request.....", t)

	//Parse the requested data
	err = proto.Unmarshal(body, &b)
	if err != nil {
		Out.Sugar().Infof("[%v][%v]Err:%v", t, len(body), err)
		return &RespBody{Code: Code_400, Msg: err.Error(), Data: nil}, nil
	}

	//Get the suffix of fileid
	ext := filepath.Ext(b.FileId)
	if ext == "" {
		Out.Sugar().Infof("[T:%v][%v]Err:Invalid dupl id", t, b.FileId)
		return &RespBody{Code: Code_400, Msg: "Invalid dupl id", Data: nil}, nil
	}

	//get fileid
	fid := strings.TrimSuffix(b.FileId, ext)

	//get file path
	fpath := filepath.Join(FilesDir, fid, b.FileId)
	_, err = os.Stat(fpath)
	if err != nil {
		Out.Sugar().Infof("[T:%v][%v]Err:%v", t, b.FileId, err)
		return &RespBody{Code: Code_404, Msg: err.Error(), Data: nil}, nil
	}

	// read file content
	buf, err := ioutil.ReadFile(fpath)
	if err != nil {
		Out.Sugar().Infof("[T:%v][%v]Err:%v", t, b.FileId, err)
		return &RespBody{Code: Code_500, Msg: err.Error(), Data: nil}, nil
	}

	// Calculate the number of slices
	slicesize, lastslicesize, num, err := cutDataRule(len(buf))
	if err != nil {
		Out.Sugar().Infof("[T:%v][%v]Err:%v", t, b.FileId, err)
		return &RespBody{Code: Code_400, Msg: err.Error(), Data: nil}, nil
	}

	//Collate returned data
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

	//proto encoding
	rtnData_proto, err := proto.Marshal(&rtnData)
	if err != nil {
		Out.Sugar().Infof("[T:%v][%v]Err:%v", t, b.FileId, err)
		return &RespBody{Code: Code_500, Msg: err.Error(), Data: nil}, nil
	}

	Out.Sugar().Infof("[T:%v]Suc:[%v]", t, b.FileId)
	return &RespBody{Code: Code_200, Msg: "success", Data: rtnData_proto}, nil
}

// Writefiletag is used to receive the file tag uploaded by the scheduling service.
// The return code is 200 for success, non-200 for failure.
// The returned Msg indicates the result reason.
func (MService) WritefiletagAction(body []byte) (proto.Message, error) {
	var (
		err     error
		b       PutTagToBucket
		tagInfo pt.TagInfo
	)
	//Generate a random number to track the log record of this request
	t := tools.RandomInRange(100000000, 999999999)
	Out.Sugar().Infof("[T:%v]Write file tag request.....", t)

	//Parse the requested data
	err = proto.Unmarshal(body, &b)
	if err != nil {
		Out.Sugar().Infof("[T:%v][%v]Err:%v", t, len(body), err)
		return &RespBody{Code: Code_400, Msg: err.Error(), Data: nil}, nil
	}

	//Get the suffix of fileid
	ext := filepath.Ext(b.FileId)
	if ext == "" {
		Out.Sugar().Infof("[T:%v][%v]Err:Invalid dupl id", b.FileId, t)
		return &RespBody{Code: Code_400, Msg: "Invalid dupl id", Data: nil}, nil
	}

	//get fileid
	fid := strings.TrimSuffix(b.FileId, ext)

	//get file path
	fpath := filepath.Join(FilesDir, fid)
	_, err = os.Stat(fpath)
	if err != nil {
		Out.Sugar().Infof("[T:%v][%v]Err:%v", t, b.FileId, err)
		return &RespBody{Code: Code_404, Msg: err.Error(), Data: nil}, nil
	}

	//Save tag information
	tagInfo.T.T0.Name = b.Name
	tagInfo.T.T0.N = b.N
	tagInfo.T.T0.U = b.U
	tagInfo.T.Signature = b.Signature
	tagInfo.Sigmas = b.Sigmas
	tag, err := json.Marshal(tagInfo)
	if err != nil {
		Out.Sugar().Infof("[T:%v][%v]Err:%v", t, b.FileId, err)
		return &RespBody{Code: Code_500, Msg: err.Error(), Data: nil}, nil
	}

	filetagname := b.FileId + ".tag"
	filefullpath := filepath.Join(fpath, filetagname)

	//Save tag information to file
	ftag, err := os.OpenFile(filefullpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		Out.Sugar().Infof("[T:%v][%v]Err:%v", t, b.FileId, err)
		return &RespBody{Code: Code_500, Msg: err.Error(), Data: nil}, nil
	}
	ftag.Write(tag)

	//flush to disk
	err = ftag.Sync()
	if err != nil {
		Out.Sugar().Infof("[T:%v][%v]Err:%v", t, b.FileId, err)
		ftag.Close()
		os.Remove(filefullpath)
		return &RespBody{Code: Code_500, Msg: err.Error(), Data: nil}, nil
	}
	ftag.Close()
	Out.Sugar().Infof("[T:%v]Suc:[%v]", t, filefullpath)
	return &RespBody{Code: Code_200, Msg: "success", Data: nil}, nil
}

// Readfiletag is used to return the file tag to the scheduling service.
// The return code is 200 for success, non-200 for failure.
// The returned Msg indicates the result reason.
func (MService) ReadfiletagAction(body []byte) (proto.Message, error) {
	var (
		err          error
		flag         bool
		filefullpath string
		b            ReadTagReq
	)
	//Generate a random number to track the log record of this request
	t := tools.RandomInRange(100000000, 999999999)
	Out.Sugar().Infof("[T:%v]Read file tag request.....", t)

	//Parse the requested data
	err = proto.Unmarshal(body, &b)
	if err != nil {
		Out.Sugar().Infof("[T:%v][%v]Err:%v", t, len(body), err)
		return &RespBody{Code: Code_400, Msg: err.Error(), Data: nil}, nil
	}

	//Query on-chain scheduling service information
	sd, code, err := chain.GetSchedulerInfoOnChain()
	if err != nil {
		if code == Code_404 {
			Out.Sugar().Infof("[T:%v][%v]Err:Not found scheduler info", t, b.FileId)
			return &RespBody{Code: Code_404, Msg: "Not found scheduler info", Data: nil}, nil
		}
		Out.Sugar().Infof("[T:%v][%v]Err:%v", t, b.FileId, err)
		return &RespBody{Code: Code_500, Msg: err.Error(), Data: nil}, nil
	}

	//Determine whether to use the new test chain address prefix
	var pre []byte
	if configs.NewTestAddr {
		pre = tools.ChainCessTestPrefix
	} else {
		pre = tools.SubstratePrefix
	}

	//Parse address
	pubkey, err := tools.DecodeToPub(b.Acc, pre)
	if err != nil {
		Out.Sugar().Infof("[T:%v][%v]Err:%v", t, b.FileId, err)
		return &RespBody{Code: Code_400, Msg: err.Error(), Data: nil}, nil
	}

	//Whether the judge scheduling is registered
	for _, v := range sd {
		if v.Controller_user == types.NewAccountID(pubkey) {
			flag = true
			break
		}
	}
	if !flag {
		Out.Sugar().Infof("[T:%v][%v]Err:Not found scheduler info", b.FileId, t)
		return &RespBody{Code: Code_404, Msg: "Not found scheduler info", Data: nil}, nil
	}

	//Get fileid and Calculate absolute file path
	ext := filepath.Ext(b.FileId)
	if ext == "" {
		filefullpath = filepath.Join(SpaceDir, b.FileId, b.FileId+".tag")
	} else {
		filefullpath = filepath.Join(FilesDir, strings.TrimSuffix(b.FileId, ext), b.FileId+".tag")
	}

	//Check if the file exists
	_, err = os.Stat(filefullpath)
	if err != nil {
		Out.Sugar().Infof("[T:%v][%v]Err:%v", t, b.FileId, err)
		return &RespBody{Code: Code_404, Msg: err.Error(), Data: nil}, nil
	}

	// read file content
	buf, err := ioutil.ReadFile(filefullpath)
	if err != nil {
		Out.Sugar().Infof("[T:%v][%v]Err:%v", t, b.FileId, err)
		return &RespBody{Code: Code_500, Msg: err.Error(), Data: nil}, nil
	}

	Out.Sugar().Infof("[T:%v]Suc:[%v]", t, filefullpath)
	return &RespBody{Code: Code_200, Msg: "success", Data: buf}, nil
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
	if b.Code == 202 || b.Code == 201 {
		return b.Data, nil
	}
	errstr := fmt.Sprintf("%d", b.Code)
	return nil, errors.New("return code:" + errstr)
}
