package rpc

import (
	"cess-bucket/configs"
	. "cess-bucket/configs"
	"cess-bucket/internal/chain"
	. "cess-bucket/internal/logger"
	"cess-bucket/internal/pt"
	"encoding/json"
	"log"
	"net"
	"net/http"

	. "cess-bucket/internal/rpc/proto"
	"cess-bucket/tools"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/soheilhy/cmux"

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

// Start http service.
func Rpc_Main() {
	l, err := net.Listen("tcp", ":"+fmt.Sprintf("%d", C.ServicePort))
	if err != nil {
		log.Fatal(err)
	}

	m := cmux.New(l)
	conn_ws := m.Match(cmux.HTTP1HeaderField("Upgrade", "websocket"))
	conn_http := m.Match(cmux.HTTP1Fast())

	go serveWs(conn_ws)
	go serveHttp(conn_http)

	if err := m.Serve(); err != nil {
		Err.Sugar().Errorf("%v", err)
	}
}

func serveWs(l net.Listener) {
	srv := NewServer()
	srv.Register(RpcService_Local, MService{})

	s_websocket := &http.Server{
		Handler: srv.WebsocketHandler([]string{"*"}),
	}

	if err := s_websocket.Serve(l); err != nil {
		fmt.Println("ws serve err: ", err)
	}
}

func serveHttp(l net.Listener) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET"}
	r.Use(cors.New(config))
	r.GET("/:fid", func(c *gin.Context) {
		fid := c.Param("fid")
		if fid == "" {
			Err.Sugar().Errorf("[%v] fid is empty", c.ClientIP())
			c.JSON(http.StatusNotFound, "fid is empty")
			return
		}
		fpath := filepath.Join(configs.FilesDir, fid)
		_, err := os.Stat(fpath)
		if err != nil {
			Err.Sugar().Errorf("[%v] file not found", c.ClientIP())
			c.JSON(http.StatusNotFound, "Not found")
			return
		}
		c.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%v", fid))
		c.Writer.Header().Add("Content-Type", "application/octet-stream")
		c.File(fpath)
	})

	s_http := &http.Server{
		Handler: r,
	}

	if err := s_http.Serve(l); err != nil {
		fmt.Println("http server err: ", err)
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

	//Parse the requested data
	err = proto.Unmarshal(body, &b)
	if err != nil {
		return &RespBody{Code: 400, Msg: "Bad Requset"}, nil
	}

	ok := false
	fpath := filepath.Join(FilesDir, b.FileId)
	if b.BlockIndex == 0 {
		var schds []chain.SchedulerInfo
		for i := 0; i < 3; i++ {
			_, schds, err = chain.GetAllSchedulerInfo()
			if err == nil {
				for _, v := range schds {
					if v.Controller_user == types.NewAccountID(b.Publickey) {
						ok = true
						break
					}
				}
				break
			}
			time.Sleep(time.Second * 3)
		}
		if !ok {
			Uld.Sugar().Infof("[%v] Forbid: [%v] %v", b.FileId, b.Publickey, err)
			return &RespBody{Code: 403, Msg: "Forbid"}, nil
		}

		//Determine whether the data base directory exists
		err = tools.CreatDirIfNotExist(FilesDir)
		if err != nil {
			Uld.Sugar().Infof("[%v] CreatDirIfNotExist [%v] err: %v", b.FileId, FilesDir, err)
			return &RespBody{Code: Code_500, Msg: err.Error()}, nil
		}

		_, err = os.Stat(fpath)
		if err == nil {
			os.Remove(fpath)
		}

		_, err = os.Create(fpath)
		if err != nil {
			Uld.Sugar().Infof("[%v]Err:%v", b.FileId, err)
			return &RespBody{Code: Code_500, Msg: err.Error()}, nil
		}

		Uld.Sugar().Infof("+++> Upload file [%v] ", b.FileId)
	}

	//save to local file
	fii, err := os.OpenFile(fpath, os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		Uld.Sugar().Infof("[%v]Err:%v", b.FileId, err)
		return &RespBody{Code: Code_500, Msg: err.Error()}, nil
	}
	defer fii.Close()
	_, err = fii.Write(b.BlockData)
	if err != nil {
		Uld.Sugar().Infof("[%v]Err:%v", b.FileId, err)
		return &RespBody{Code: Code_500, Msg: err.Error()}, nil
	}
	//flush to disk
	err = fii.Sync()
	if err != nil {
		Uld.Sugar().Infof("[%v]Err:%v", b.FileId, err)
		return &RespBody{Code: Code_500, Msg: err.Error()}, nil
	}
	Uld.Sugar().Infof("[%v]Suc:[%v]", b.FileId, b.BlockIndex)
	return &RespBody{Code: Code_200, Msg: "success"}, nil
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

	//Parse the requested data
	err = proto.Unmarshal(body, &b)
	if err != nil {
		return &RespBody{Code: 400, Msg: "Request error"}, nil
	}

	//get file path
	fpath := filepath.Join(FilesDir, b.FileId)
	fstat, err := os.Stat(fpath)
	if err != nil {
		Out.Sugar().Infof("[%v] Stat Err: %v", b.FileId, err)
		return &RespBody{Code: Code_404, Msg: err.Error()}, nil
	}

	// read file content
	f, err := os.OpenFile(fpath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		Out.Sugar().Infof("[%v] OpenFile Err: %v", b.FileId, err)
		return &RespBody{Code: Code_500, Msg: err.Error()}, nil
	}

	// Calculate the number of slices
	blockTotal := fstat.Size() / configs.RpcFileBuffer
	if fstat.Size()%configs.RpcFileBuffer != 0 {
		blockTotal++
	}
	if b.BlockIndex > uint32(blockTotal) {
		Out.Sugar().Infof("[%v]Err:Invalid block index", b.FileId)
		return &RespBody{Code: Code_400, Msg: "Invalid block index"}, nil
	}

	//Collate returned data
	rtnData.BlockTotal = uint32(blockTotal)
	rtnData.BlockIndex = b.BlockIndex
	var tmp = make([]byte, configs.RpcFileBuffer)
	f.Seek(int64(b.BlockIndex*configs.RpcFileBuffer), 0)
	n, _ := f.Read(tmp)
	rtnData.Data = tmp[:n]
	f.Close()
	//proto encoding
	rtnData_proto, err := proto.Marshal(&rtnData)
	if err != nil {
		Out.Sugar().Infof("[%v]Marshal Err:%v", b.FileId, err)
		return &RespBody{Code: Code_500, Msg: err.Error(), Data: nil}, nil
	}

	Out.Sugar().Infof("[%v]Download suc [%v-%v]", b.FileId, blockTotal, b.BlockIndex)
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

	//Parse the requested data
	err = proto.Unmarshal(body, &b)
	if err != nil {
		return &RespBody{Code: 400, Msg: "Request error"}, nil
	}

	//Save tag information
	tagInfo.T.T0.Name = b.Name
	tagInfo.T.T0.N = b.N
	tagInfo.T.T0.U = b.U
	tagInfo.T.Signature = b.Signature
	tagInfo.Sigmas = b.Sigmas
	tag, err := json.Marshal(tagInfo)
	if err != nil {
		Out.Sugar().Infof("[%v]Err:%v", b.FileId, err)
		return &RespBody{Code: Code_500, Msg: err.Error()}, nil
	}

	filetagname := b.FileId + ".tag"
	filefullpath := filepath.Join(FilesDir, filetagname)

	//Save tag information to file
	ftag, err := os.OpenFile(filefullpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		Out.Sugar().Infof("[%v]Err:%v", b.FileId, err)
		return &RespBody{Code: Code_500, Msg: err.Error()}, nil
	}
	ftag.Write(tag)

	//flush to disk
	err = ftag.Sync()
	if err != nil {
		Out.Sugar().Infof("[%v]Err:%v", b.FileId, err)
		ftag.Close()
		os.Remove(filefullpath)
		return &RespBody{Code: Code_500, Msg: err.Error()}, nil
	}
	ftag.Close()
	Out.Sugar().Infof("[%v]Save tag suc", b.FileId)
	return &RespBody{Code: Code_200, Msg: "success"}, nil
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
	num := size / configs.RpcFileBuffer
	slicesize := size / (num + 1)
	tailsize := size - slicesize*(num+1)
	return slicesize, slicesize + tailsize, uint8(num) + 1, nil
}

//
func WriteData(cli *Client, service, method string, body []byte) (int, []byte, bool, error) {
	req := &ReqMsg{
		Service: service,
		Method:  method,
		Body:    body,
	}
	ctx, _ := context.WithTimeout(context.Background(), 120*time.Second)
	resp, err := cli.Call(ctx, req)
	if err != nil {
		cli.Close()
		return 0, nil, true, errors.Wrap(err, "Call err:")
	}

	var b RespBody
	err = proto.Unmarshal(resp.Body, &b)
	if err != nil {
		return 0, nil, false, errors.Wrap(err, "Unmarshal:")
	}
	// if b.Code == configs.Code_200 || b.Code == configs.Code_600 {
	// 	return 0,b.Data, false, nil
	// }
	//errstr := fmt.Sprintf("%d", b.Code)
	return int(b.Code), b.Data, false, nil
}
