package rpc

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"storage-mining/configs"
	. "storage-mining/rpc/proto"
	"storage-mining/tools"
	"strings"

	"github.com/golang/protobuf/proto"
)

type MService struct {
}

func Rpc_Init() {
	if err := tools.CreatDirIfNotExist(configs.Confile.MinerData.MountedPath); err != nil {
		panic(err)
	}
}

func Rpc_Main() {
	srv := NewServer()
	srv.Register("mservice", MService{})
	err := http.ListenAndServe(":"+fmt.Sprintf("%d", configs.MinerServicePort), srv.WebsocketHandler([]string{"*"}))
	if err != nil {
		panic(err)
	}
}

// Test
// func (MService) TestAction(body []byte) (proto.Message, error) {
// 	fmt.Println("**** recv a test connect ****")
// 	return &RespMsg{Body: []byte("test hello")}, nil
// }

// Write file from scheduler
func (MService) WritefileAction(body []byte) (proto.Message, error) {
	fmt.Println("**** recv a writefile connect ****")
	var (
		b FileUploadInfo
	)

	err := proto.Unmarshal(body, &b)
	if err != nil {
		return &RespBody{Code: 400, Msg: "body format error", Data: nil}, nil
	}

	//fmt.Println("**** recv a writefile connect-2 ****")
	err = tools.CreatDirIfNotExist(configs.ServiceDir)
	if err != nil {
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	fid := strings.Split(filepath.Base(b.FileId), ".")[0]
	fpath := filepath.Join(configs.ServiceDir, fid)
	if err = os.MkdirAll(fpath, os.ModeDir); err != nil {
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}

	//fmt.Println("fpath+b.fid: ", fpath+b.FileId)
	fii, err := os.OpenFile(filepath.Join(fpath, filepath.Base(b.FileId)), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		//TODO
		fmt.Println(err)
		return &RespBody{Code: 500, Msg: err.Error()}, nil
	}
	defer fii.Close()
	fii.Write(b.Data)
	fmt.Println(filepath.Join(fpath, filepath.Base(b.FileId)))
	//fmt.Println("**** recv a writefile connect-3 ****")
	return &RespBody{Code: 0, Msg: "sucess, i am miner"}, nil
}

// Read file from client
func (MService) ReadfileAction(body []byte) (proto.Message, error) {
	var (
		b FileDownloadReq
	)
	fmt.Println("**** recv a readfile connect ****")
	err := proto.Unmarshal(body, &b)
	if err != nil {
		return &RespBody{Code: 400, Msg: err.Error()}, nil
	}
	fmt.Println("req info: ", b)
	fid := strings.Split(b.FileId, ".")[0]
	fmt.Println("fid: ", fid)
	fpath := filepath.Join(configs.ServiceDir, fid, b.FileId)
	fmt.Println("fpath: ", fpath)
	_, err = os.Stat(fpath)
	if err != nil {
		return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
	}
	buf, err := ioutil.ReadFile(fpath)
	if err != nil {
		fmt.Println(err)
		return &RespBody{Code: 400, Msg: err.Error()}, nil
	}
	slicesize, lastslicesize, num, err := cutDataRule(len(buf))
	if err != nil {
		fmt.Println("cutDataRule err: ", err)
		return &RespBody{Code: 400, Msg: err.Error()}, nil
	}
	var rtnData FileDownloadInfo
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
		fmt.Println("Marshal err: ", err)
		return &RespBody{Code: 400, Msg: err.Error()}, nil
	}
	return &RespBody{Code: 0, Msg: "success", Data: rtnData_proto}, nil
}

//
// func writeFile(dst string, body []byte) error {
// 	dstip := tools.Base58Decoding(dst)
// 	wsURL := "ws:" + strings.TrimPrefix(dstip, "http:")
// 	req := &ReqMsg{
// 		Service: configs.RpcService_Scheduler,
// 		Method:  configs.RpcMethod_Scheduler_Writefile,
// 		Body:    body,
// 	}
// 	client, err := DialWebsocket(context.Background(), wsURL, "")
// 	if err != nil {
// 		return err
// 	}
// 	defer client.Close()
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()
// 	resp, err := client.Call(ctx, req)
// 	if err != nil {
// 		return err
// 	}
// 	var b RespBody
// 	err = proto.Unmarshal(resp.Body, &b)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	if b.Code == 0 {
// 		return nil
// 	}
// 	errstr := fmt.Sprintf("%d", b.Code)
// 	return errors.New("return code:" + errstr)
// }

//
// func readFile(dst string, body []byte) ([]byte, error) {
// 	dstip := tools.Base58Decoding(dst)
// 	wsURL := "ws:" + strings.TrimPrefix(dstip, "http:")
// 	req := &ReqMsg{
// 		Service: configs.RpcService_Scheduler,
// 		Method:  configs.RpcMethod_Scheduler_Readfile,
// 		Body:    body,
// 	}
// 	client, err := DialWebsocket(context.Background(), wsURL, "")
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer client.Close()
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()
// 	resp, err := client.Call(ctx, req)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var b RespBody
// 	err = proto.Unmarshal(resp.Body, &b)
// 	if err != nil {
// 		return resp.Body, nil
// 	}
// 	errstr := fmt.Sprintf("%d", b.Code)
// 	return nil, errors.New(errstr)
// }

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
