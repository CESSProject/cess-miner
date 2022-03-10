package rpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"storage-mining/configs"
	"storage-mining/tools"
	"strings"
	"time"

	myproto "storage-mining/rpc/proto"

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
func (MService) TestAction(body []byte) (proto.Message, error) {
	return &Err{Msg: "test hello"}, nil
}

// Write file from scheduler
func (MService) WritefileAction(body []byte) (proto.Message, error) {
	var (
		b myproto.FileUploadInfo
	)
	err := proto.Unmarshal(body, &b)
	if err != nil {
		return &Err{Code: 400, Msg: "body format error"}, nil
	}

	return &Err{Code: 0, Msg: "sucess"}, nil
}

// Read file from client
func (MService) ReadfileAction(body []byte) (proto.Message, error) {
	var (
		b myproto.FileDownloadReq
	)
	err := proto.Unmarshal(body, &b)
	if err != nil {
		return &Err{Code: 400, Msg: "body format error"}, nil
	}

	return &Err{Code: 500, Msg: "fail"}, nil
}

//
func writeFile(dst string, body []byte) error {
	dstip := tools.Base58Decoding(dst)
	wsURL := "ws:" + strings.TrimPrefix(dstip, "http:")
	req := &ReqMsg{
		Service: configs.RpcService_Scheduler,
		Method:  configs.RpcMethod_Scheduler_Writefile,
		Body:    body,
	}
	client, err := DialWebsocket(context.Background(), wsURL, "")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := client.Call(ctx, req)
	if err != nil {
		return err
	}
	cancel()
	var b Err
	err = proto.Unmarshal(resp.Body, &b)
	if err != nil {
		fmt.Println(err)
	}
	if b.Code == 0 {
		return nil
	}
	errstr := fmt.Sprintf("%d", b.Code)
	return errors.New("return code:" + errstr)
}

//
func readFile(dst string, body []byte) ([]byte, error) {
	dstip := tools.Base58Decoding(dst)
	wsURL := "ws:" + strings.TrimPrefix(dstip, "http:")
	req := &ReqMsg{
		Service: configs.RpcService_Scheduler,
		Method:  configs.RpcMethod_Scheduler_Readfile,
		Body:    body,
	}
	client, err := DialWebsocket(context.Background(), wsURL, "")
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := client.Call(ctx, req)
	if err != nil {
		return nil, err
	}
	cancel()
	var b Err
	err = proto.Unmarshal(resp.Body, &b)
	if err != nil {
		return resp.Body, nil
	}
	errstr := fmt.Sprintf("%d", b.Code)
	return nil, errors.New(errstr)
}
