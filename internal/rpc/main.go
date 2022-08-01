package rpc

import (
	"cess-bucket/configs"
	. "cess-bucket/configs"
	"cess-bucket/internal/chain"
	. "cess-bucket/internal/logger"
	api "cess-bucket/internal/proof/apiv1"
	"cess-bucket/internal/pt"
	. "cess-bucket/internal/rpc/proto"
	"cess-bucket/tools"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	keyring "github.com/CESSProject/go-keyring"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/soheilhy/cmux"
	"storj.io/common/base58"

	"google.golang.org/protobuf/proto"
)

type MService struct {
}

type RespSpaceInfo struct {
	FileId string `json:"fileId"`
	Token  string `json:"token"`
	T      api.FileTagT
	Sigmas [][]byte `json:"sigmas"`
}

type kvpair struct {
	K string
	V int32
}

type baseFiller struct {
	MinerIp  []string `json:"minerIp"`
	FillerId string   `json:"fillerId"`
}

var globalTransport *http.Transport

// Init
func init() {
	globalTransport = &http.Transport{
		DisableKeepAlives: true,
	}
}

// Start http service.
func Rpc_Main() {
	go task_management()
	l, err := net.Listen("tcp", ":"+fmt.Sprintf("%d", C.ServicePort))
	if err != nil {
		log.Fatal(err)
	}

	m := cmux.New(l)
	conn_ws := m.Match(cmux.HTTP1HeaderField("Upgrade", "websocket"))
	conn_http := m.Match(cmux.HTTP1Fast())

	go serveWs(conn_ws)
	go serveHttp(conn_http)

	log.Println("Start and listen on port ", configs.C.ServicePort, "...")
	if err := m.Serve(); err != nil {
		Err.Sugar().Errorf("%v", err)
	}
}

// Start the proof module
func task_management() {
	var (
		channel_1 = make(chan bool, 1)
		channel_2 = make(chan bool, 1)
		channel_3 = make(chan bool, 1)
	)
	go task_SpaceManagement(channel_1)
	go task_HandlingChallenges(channel_2)
	go task_RemoveInvalidFiles(channel_3)
	for {
		select {
		case <-channel_1:
			go task_SpaceManagement(channel_1)
		case <-channel_2:
			go task_HandlingChallenges(channel_2)
		case <-channel_3:
			go task_RemoveInvalidFiles(channel_3)
		}
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
	config.AllowHeaders = []string{"cache-control", "x-requested-with", "*"}
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
			fpath = filepath.Join(configs.SpaceDir, fid)
			_, err = os.Stat(fpath)
			if err != nil {
				Err.Sugar().Errorf("[%v] file not found", c.ClientIP())
				c.JSON(http.StatusNotFound, "Not found")
				return
			}
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
			schds, _, err = chain.GetSchedulingNodes()
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
	if b.BlockIndex > uint32(blockTotal) || b.BlockIndex == 0 {
		Out.Sugar().Infof("[%v]Err:Invalid block index", b.FileId)
		return &RespBody{Code: Code_400, Msg: "Invalid block index"}, nil
	}

	//Collate returned data
	rtnData.BlockTotal = uint32(blockTotal)
	rtnData.BlockIndex = b.BlockIndex
	var tmp = make([]byte, configs.RpcFileBuffer)
	f.Seek(int64((b.BlockIndex-1)*configs.RpcFileBuffer), 0)
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
		err error
		b   ReadTagReq
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

	//Get fileid and Calculate absolute file path
	filetagfullpath := ""
	if b.FileId[:4] != "cess" {
		filetagfullpath = filepath.Join(configs.SpaceDir, b.FileId+".tag")
	} else {
		filetagfullpath = filepath.Join(configs.FilesDir, b.FileId+".tag")
	}

	//Check if the file exists
	_, err = os.Stat(filetagfullpath)
	if err != nil {
		Out.Sugar().Infof("[T:%v][%v]Err:%v", t, b.FileId, err)
		return &RespBody{Code: Code_404, Msg: err.Error(), Data: nil}, nil
	}

	// read file content
	buf, err := ioutil.ReadFile(filetagfullpath)
	if err != nil {
		Out.Sugar().Infof("[T:%v][%v]Err:%v", t, b.FileId, err)
		return &RespBody{Code: Code_500, Msg: err.Error(), Data: nil}, nil
	}

	Out.Sugar().Infof("[T:%v]Suc:[%v]", t, filetagfullpath)
	return &RespBody{Code: Code_200, Msg: "success", Data: buf}, nil
}

// Divide the size according to 2M
func cutDataRule(size int) (int, int, uint8, error) {
	if size <= 0 {
		return 0, 0, 0, errors.New("size is lt 0")
	}

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

//The task_SpaceManagement task is to automatically allocate hard disk space.
//It will help you use your allocated hard drive space, until the size you set in the config file is reached.
//It keeps running as a subtask.
func task_SpaceManagement(ch chan bool) {
	var (
		availableSpace uint64
		reconn         bool
		tSpace         time.Time
		reqspace       SpaceReq
		reqspacefile   SpaceFileReq
		tagInfo        pt.TagInfo
		respspace      RespSpaceInfo
		client         *Client
	)
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
		ch <- true
	}()
	Flr.Info("-----> Start task_SpaceManagement <-----")

	pubkey, err := chain.GetAccountPublickey(configs.C.SignatureAcc)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	availableSpace, err = calcAvailableSpace()
	if err != nil {
		Flr.Sugar().Errorf("%v", err)
	} else {
		tSpace = time.Now()
	}

	reqspace.Publickey = pubkey

	kr, _ := keyring.FromURI(configs.C.SignatureAcc, keyring.NetSubstrate{})

	for {
		time.Sleep(time.Second)
		if client == nil || reconn {
			schds, _, err := chain.GetSchedulingNodes()
			if err != nil {
				Flr.Sugar().Errorf("   %v", err)
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(10, 30)))
				continue
			}
			client, err = connectionScheduler(schds)
			if err != nil {
				Flr.Sugar().Errorf("-->Err: All schedules unavailable")
				for i := 0; i < len(schds); i++ {
					Err.Sugar().Errorf("   %v", string(schds[i].Ip))
				}
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(10, 30)))
				continue
			}
		}

		if time.Since(tSpace).Minutes() >= 10 {
			availableSpace, err = calcAvailableSpace()
			if err != nil {
				Flr.Sugar().Errorf(" %v", err)
			} else {
				tSpace = time.Now()
			}
		}

		if availableSpace < uint64(8*configs.Space_1MB) {
			Flr.Info("Your space is certified")
			time.Sleep(time.Minute * time.Duration(tools.RandomInRange(10, 30)))
			continue
		}

		// sign message
		msg := []byte(fmt.Sprintf("%v", tools.RandomInRange(100000, 999999)))
		sig, _ := kr.Sign(kr.SigningContext(msg))
		reqspace.Msg = msg
		reqspace.Sign = sig[:]

		req_b, err := proto.Marshal(&reqspace)
		if err != nil {
			Flr.Sugar().Errorf(" %v", err)
			continue
		}

		respCode, respBody, clo, err := WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_Space, req_b)
		reconn = clo
		if err != nil {
			Flr.Sugar().Errorf(" %v", err)
			continue
		}

		if respCode == 201 {
			var basefiller baseFiller
			err = json.Unmarshal(respBody, &basefiller)
			if err != nil {
				Flr.Sugar().Errorf(" %v", err)
				continue
			}
			var fillerurl string = "http://" + string(base58.Decode(basefiller.MinerIp[0])) + "/" + basefiller.FillerId
			var fillertagurl string = fillerurl + ".tag"
			fillerbody, err := getFiller(fillerurl)
			if err != nil {
				Flr.Sugar().Errorf(" %v", err)
				continue
			}
			spacefilefullpath := filepath.Join(configs.SpaceDir, basefiller.FillerId)
			err = write_file(spacefilefullpath, fillerbody)
			if err != nil {
				os.Remove(spacefilefullpath)
				Flr.Sugar().Errorf(" %v", err)
				continue
			}
			fillertagbody, err := getFiller(fillertagurl)
			if err != nil {
				Flr.Sugar().Errorf(" %v", err)
				continue
			}
			tagfilename := basefiller.FillerId + ".tag"
			tagfilefullpath := filepath.Join(configs.SpaceDir, tagfilename)
			err = write_file(tagfilefullpath, fillertagbody)
			if err != nil {
				os.Remove(tagfilefullpath)
				Flr.Sugar().Errorf(" %v", err)
				continue
			}
			hash, err := tools.CalcFileHash(spacefilefullpath)
			if err != nil {
				os.Remove(tagfilefullpath)
				os.Remove(spacefilefullpath)
				Flr.Sugar().Errorf(" %v", err)
				continue
			}
			//
			var req_back FillerBackReq
			req_back.Publickey = pubkey
			req_back.FileId = []byte(basefiller.FillerId)
			req_back.FileHash = []byte(hash)
			req_back_req, err := proto.Marshal(&req_back)
			if err != nil {
				Flr.Sugar().Errorf(" %v", err)
				continue
			}

			_, _, clo, err := WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_FillerBack, req_back_req)
			reconn = clo
			if err != nil {
				Flr.Sugar().Errorf(" %v", err)
			}
			continue
		}

		if respCode != 200 {
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(10, 60)))
			continue
		}

		err = json.Unmarshal(respBody, &respspace)
		if err != nil {
			Flr.Sugar().Errorf(" %v", err)
			continue
		}

		//save space file tag
		tagfilename := respspace.FileId + ".tag"
		tagfilefullpath := filepath.Join(configs.SpaceDir, tagfilename)
		tagInfo.T = respspace.T
		tagInfo.Sigmas = respspace.Sigmas
		tag, err := json.Marshal(tagInfo)
		if err != nil {
			Flr.Sugar().Errorf(" %v", err)
			continue
		}
		err = write_file(tagfilefullpath, tag)
		if err != nil {
			os.Remove(tagfilefullpath)
			Flr.Sugar().Errorf(" %v", err)
			continue
		}

		spacefilefullpath := filepath.Join(configs.SpaceDir, respspace.FileId)
		f, err := os.OpenFile(spacefilefullpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
		if err != nil {
			os.Remove(tagfilefullpath)
			Flr.Sugar().Errorf(" %v", err)
			continue
		}
		reqspacefile.Token = respspace.Token

		for i := 0; i < 17; i++ {
			reqspacefile.BlockIndex = uint32(i)
			req_b, err = proto.Marshal(&reqspacefile)
			if err != nil {
				Flr.Sugar().Errorf(" %v", err)
				f.Close()
				os.Remove(tagfilefullpath)
				os.Remove(spacefilefullpath)
				break
			}
			respCode, respBody, clo, err = WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_Spacefile, req_b)
			reconn = clo
			if err != nil {
				Flr.Sugar().Errorf(" %v", err)
				f.Close()
				os.Remove(tagfilefullpath)
				os.Remove(spacefilefullpath)
				break
			}
			if i < 16 {
				f.Write(respBody)
				if i == 15 {
					f.Close()
				}
			}
		}
	}
}

//The task_HandlingChallenges task will automatically help you complete file challenges.
//Apart from human influence, it ensures that you submit your certificates in a timely manner.
//It keeps running as a subtask.
func task_HandlingChallenges(ch chan bool) {
	var (
		err             error
		code            int
		fileid          string
		fileFullPath    string
		fileTagFullPath string
		blocksize       int64
		filetag         pt.TagInfo
		poDR2prove      api.PoDR2Prove
		proveResponse   api.PoDR2ProveResponse
		puk             chain.Chain_SchedulerPuk
		chlng           []chain.ChallengesInfo
	)
	defer func() {
		err := recover()
		if err != nil {
			Err.Error(tools.RecoverError(err))
		}
		ch <- true
	}()
	Out.Info(">>>>> Start task_HandlingChallenges <<<<<")

	//Get the scheduling service public key
	for {
		puk, _, err = chain.GetPublicKey()
		if err != nil {
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
			continue
		}
		Out.Info("Get the scheduling public key")
		break
	}

	pubkey, err := chain.GetAccountPublickey(configs.C.SignatureAcc)
	if err != nil {
		Err.Sugar().Errorf("[%v] %v", fileid, err)
		os.Exit(1)
	}

	for {
		chlng, code, err = chain.GetChallenges(configs.C.SignatureAcc)
		if err != nil {
			if code != configs.Code_404 {
				Out.Sugar().Infof("[ERR] %v", err)
			}
			time.Sleep(time.Minute * time.Duration(tools.RandomInRange(3, 5)))
			continue
		}

		if len(chlng) == 0 {
			time.Sleep(time.Minute * time.Duration(tools.RandomInRange(2, 10)))
			continue
		}
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(30, 60)))
		Out.Sugar().Infof("--> Number of challenges: %v ", len(chlng))
		for x := 0; x < len(chlng); x++ {
			Out.Sugar().Infof("  %v: %s ", x, string(chlng[x].File_id))
		}
		var proveInfos = make([]chain.ProveInfo, 0)
		for i := 0; i < len(chlng); i++ {
			if len(proveInfos) > 80 {
				break
			}

			fileid = string(chlng[i].File_id)
			if chlng[i].File_type == 1 {
				//space file
				fileFullPath = filepath.Join(configs.SpaceDir, fileid)
				fileTagFullPath = filepath.Join(configs.SpaceDir, fileid+".tag")
			} else {
				//user file
				fileFullPath = filepath.Join(configs.FilesDir, fileid)
				fileTagFullPath = filepath.Join(configs.FilesDir, fileid+".tag")
			}

			fstat, err := os.Stat(fileFullPath)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", fileid, err)
				continue
			}
			if chlng[i].File_type == 1 {
				blocksize = configs.BlockSize
			} else {
				blocksize, _ = calcFileBlockSizeAndScanSize(fstat.Size())
			}

			qSlice, err := api.PoDR2ChallengeGenerateFromChain(chlng[i].Block_list, chlng[i].Random)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", fileid, err)
				continue
			}

			ftag, err := ioutil.ReadFile(fileTagFullPath)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", fileid, err)
				continue
			}
			err = json.Unmarshal(ftag, &filetag)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", fileid, err)
				continue
			}

			poDR2prove.QSlice = qSlice
			poDR2prove.T = filetag.T
			poDR2prove.Sigmas = filetag.Sigmas

			matrix, _, err := split(fileFullPath, blocksize, fstat.Size())
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", fileid, err)
				continue
			}

			poDR2prove.Matrix = matrix
			poDR2prove.S = blocksize
			proveResponseCh := poDR2prove.PoDR2ProofProve(puk.Spk, string(puk.Shared_params), puk.Shared_g, int64(configs.ScanBlockSize))
			select {
			case proveResponse = <-proveResponseCh:
			}
			if proveResponse.StatueMsg.StatusCode != api.Success {
				Err.Sugar().Errorf("[%v] %v", fileid, err)
				continue
			}

			proveInfoTemp := chain.ProveInfo{}
			proveInfoTemp.Cinfo = chlng[i]
			proveInfoTemp.FileId = chlng[i].File_id

			var mus []types.Bytes = make([]types.Bytes, len(proveResponse.MU))
			for i := 0; i < len(proveResponse.MU); i++ {
				mus[i] = make(types.Bytes, 0)
				mus[i] = append(mus[i], proveResponse.MU[i]...)
			}
			proveInfoTemp.Mu = mus
			proveInfoTemp.Sigma = types.Bytes(proveResponse.Sigma)
			proveInfoTemp.MinerAcc = types.NewAccountID(pubkey)
			proveInfos = append(proveInfos, proveInfoTemp)
		}
		// proof up chain
		ts := time.Now().Unix()
		code = 0
		txhash := ""
		for code != int(configs.Code_200) && code != int(configs.Code_600) {
			txhash, code, err = chain.SubmitProofs(configs.C.SignatureAcc, proveInfos)
			if txhash != "" {
				Out.Sugar().Infof("Proofs submitted successfully [%v]", txhash)
				break
			}
			if time.Since(time.Unix(ts, 0)).Minutes() > 2.0 {
				Err.Sugar().Errorf("[%v] %v", fileid, err)
				break
			}
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 20)))
		}
	}
}

//The task_RemoveInvalidFiles task automatically checks its own failed files and clears them.
//Delete from the local disk first, and then notify the chain to delete.
//It keeps running as a subtask.
func task_RemoveInvalidFiles(ch chan bool) {
	defer func() {
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
		ch <- true
	}()
	Out.Info(">>>>> Start task_RemoveInvalidFiles <<<<<")
	for {
		invalidFiles, code, err := chain.GetInvalidFiles(configs.C.SignatureAcc)
		if err != nil {
			if code != configs.Code_404 {
				Out.Sugar().Infof("%v", err)
			}
			time.Sleep(time.Minute * time.Duration(tools.RandomInRange(5, 10)))
			continue
		}

		if len(invalidFiles) == 0 {
			time.Sleep(time.Minute * time.Duration(tools.RandomInRange(5, 10)))
			continue
		}

		Out.Sugar().Infof("--> Prepare to remove invalid files [%v]", len(invalidFiles))
		for x := 0; x < len(invalidFiles); x++ {
			Out.Sugar().Infof("   %v: %s", x, string(invalidFiles[x]))
		}

		for i := 0; i < len(invalidFiles); i++ {
			fileid := string(invalidFiles[i])
			filefullpath := ""
			filetagfullpath := ""
			if fileid[:4] != "cess" {
				filefullpath = filepath.Join(configs.SpaceDir, fileid)
				filetagfullpath = filepath.Join(configs.SpaceDir, fileid+".tag")
			} else {
				filefullpath = filepath.Join(configs.FilesDir, fileid)
				filetagfullpath = filepath.Join(configs.FilesDir, fileid+".tag")
			}
			txhash, err := chain.ClearInvalidFiles(configs.C.SignatureAcc, invalidFiles[i])
			if txhash != "" {
				Out.Sugar().Infof("[%v] Cleared %v", string(invalidFiles[i]), txhash)
			} else {
				Out.Sugar().Infof("[err] [%v] Clear: %v", string(invalidFiles[i]), err)
			}
			os.Remove(filefullpath)
			os.Remove(filetagfullpath)
		}
	}
}

// Calculate available space
func calcAvailableSpace() (uint64, error) {
	var err error

	usedSpace, err := tools.DirSize(configs.BaseDir)
	if err != nil {
		return 0, err
	}

	sspace := configs.C.StorageSpace * configs.Space_1GB
	mountP, err := pt.GetMountPathInfo(configs.C.MountedPath)
	if err != nil {
		return 0, err
	}

	if sspace <= usedSpace {
		return 0, nil
	}

	if mountP.Free > configs.Space_1MB*100 {
		if usedSpace < sspace {
			return sspace - usedSpace, nil
		}
	}
	return 0, nil
}

func calcFileBlockSizeAndScanSize(fsize int64) (int64, int64) {
	var (
		blockSize     int64
		scanBlockSize int64
	)
	if fsize < configs.ByteSize_1Kb {
		return fsize, fsize
	}
	if fsize > math.MaxUint32 {
		blockSize = math.MaxUint32
		scanBlockSize = blockSize / 8
		return blockSize, scanBlockSize
	}
	blockSize = fsize / 16
	scanBlockSize = blockSize / 8
	return blockSize, scanBlockSize
}

func connectionScheduler(schds []chain.SchedulerInfo) (*Client, error) {
	var (
		ok    bool
		err   error
		state = make(map[string]int32)
		cli   *Client
	)
	if len(schds) == 0 {
		return nil, errors.New("No scheduler service available")
	}
	var wsURL string
	for i := 0; i < len(schds); i++ {
		wsURL = "ws://" + string(base58.Decode(string(schds[i].Ip)))
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		cli, err = DialWebsocket(ctx, wsURL, "")
		if err != nil {
			continue
		}
		respCode, respBody, _, _ := WriteData(cli, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_State, nil)
		if respCode != 200 {
			cli.Close()
			continue
		}
		var resu int32
		if len(respBody) == 4 {
			resu += int32(respBody[0])
			resu = resu << 8
			resu += int32(respBody[1])
			resu = resu << 8
			resu += int32(respBody[2])
			resu = resu << 8
			resu += int32(respBody[3])
		}
		state[wsURL] = resu
		cli.Close()
	}
	tmpList := make([]kvpair, 0)
	for k, v := range state {
		tmpList = append(tmpList, kvpair{K: k, V: v})
	}
	sort.Slice(tmpList, func(i, j int) bool {
		return tmpList[i].V < tmpList[j].V
	})
	ok = false
	for _, pair := range tmpList {
		t := time.Duration(1)
		for i := 0; i < 3; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(t*time.Second))
			cli, err = DialWebsocket(ctx, pair.K, "")
			cancel()
			if err == nil {
				ok = true
				break
			}
			t += 2
		}
		if ok {
			break
		}
	}
	return cli, err
}

func split(filefullpath string, blocksize, filesize int64) ([][]byte, uint64, error) {
	file, err := os.Open(filefullpath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	if filesize/blocksize == 0 {
		return nil, 0, errors.New("filesize invalid")
	}
	n := filesize / blocksize
	if n == 0 {
		n = 1
	}
	// matrix is indexed as m_ij, so the first dimension has n items and the second has s.
	matrix := make([][]byte, n)
	for i := int64(0); i < n; i++ {
		piece := make([]byte, blocksize)
		_, err := file.Read(piece)
		if err != nil {
			return nil, 0, err
		}
		matrix[i] = piece
	}
	return matrix, uint64(n), nil
}

func write_file(fpath string, data []byte) error {
	ft, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer ft.Close()
	_, err = ft.Write(data)
	if err != nil {
		return err
	}
	return ft.Sync()
}

func getFiller(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)

	client := &http.Client{
		Transport: globalTransport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
