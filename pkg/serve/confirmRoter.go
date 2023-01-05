/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package serve

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/db"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

// FileRouter
type ConfirmRouter struct {
	BaseRouter
	Chn     chain.IChain
	Logs    logger.ILog
	Cach    db.ICache
	Cfile   confile.IConfile
	FileDir string
	TmpDir  string
}

type MsgConfirm struct {
	Token     string `json:"token"`
	RootHash  string `json:"roothash"`
	SliceHash string `json:"slicehash"`
	ShardId   string `json:"shardId"`
}

// FileRouter Handle
func (c *ConfirmRouter) Handle(ctx context.CancelFunc, request IRequest) {
	fmt.Println("Call ConfirmRouter Handle msgId=", request.GetMsgID())

	if request.GetMsgID() != Msg_Confirm {
		fmt.Println("MsgId error")
		ctx()
		return
	}

	var msg MsgConfirm
	err := json.Unmarshal(request.GetData(), &msg)
	if err != nil {
		fmt.Println("Msg format error")
		ctx()
		return
	}

	fpath := filepath.Join(c.TmpDir, msg.SliceHash)
	_, err = os.Stat(fpath)
	if err != nil {
		fmt.Println("file not found: ", fpath)
		request.GetConnection().SendMsg(Msg_ClientErr, nil)
		return
	}

	var message chain.MessageType
	if len(msg.ShardId) != len(chain.SliceId{}) {
		fmt.Println("invalid ShardId: ", msg.ShardId)
		request.GetConnection().SendMsg(Msg_ClientErr, nil)
		return
	}
	message.ShardId = msg.ShardId

	if len(msg.SliceHash) != len(chain.FileHash{}) {
		fmt.Println("invalid SliceHash: ", msg.SliceHash)
		request.GetConnection().SendMsg(Msg_ClientErr, nil)
		return
	}
	message.SliceHash = msg.SliceHash

	ips := strings.Split(c.Cfile.GetServiceAddr(), ".")

	message.MinerIp = fmt.Sprintf("%v/%v/%v/%v/%d", ips[0], ips[1], ips[2], ips[3], c.Cfile.GetServicePortNum())

	val, err := json.Marshal(&message)
	if err != nil {
		fmt.Println("Marshal err: ", err)
		request.GetConnection().SendMsg(Msg_ServerErr, nil)
		return
	}

	err = GetSignReq(string(val), configs.URL_GetSign, c.Cfile.GetServiceAddr(), configs.URL_GetSign_Callback, c.Cfile.GetSgxPortNum())
	if err != nil {
		fmt.Println("GetSignReq err: ", err)
		request.GetConnection().SendMsg(Msg_ServerErr, nil)
		return
	}

	var sign chain.SliceSummary
	timeout := time.NewTicker(configs.TimeOut_WaitSign)
	defer timeout.Stop()
	select {
	case <-timeout.C:
		c.Logs.Space("err", fmt.Errorf("Wait tag timeout"))
		return
	case v := <-configs.Ch_Sign:
		b, err := hex.DecodeString(v)
		if err != nil {
			fmt.Println("DecodeString err: ", err)
			request.GetConnection().SendMsg(Msg_ServerErr, nil)
			return
		}
		if len(b) != len(chain.Signature{}) {
			fmt.Println("Invalid sign: ", v)
			request.GetConnection().SendMsg(Msg_ServerErr, nil)
			return
		}
		for i := 0; i < len(b); i++ {
			sign.Signature[i] = types.U8(b[i])
		}
	}

	sign.Message = val
	sign.Miner_acc = types.NewAccountID(c.Chn.GetPublicKey())

	b, err := json.Marshal(&sign)
	if err != nil {
		fmt.Println("Marshal sliceSum err ", fpath)
		request.GetConnection().SendMsg(Msg_ServerErr, nil)
		return
	}

	err = request.GetConnection().SendMsg(Msg_OK, b)
	if err != nil {
		fmt.Println(err)
		return
	}
	os.Rename(fpath, filepath.Join(c.FileDir, msg.SliceHash))
}

func GetSignReq(msg string, callUrl, callbackIp, callbackRouter string, callbackPort int) error {
	callbackurl := fmt.Sprintf("http://%v:%d%v", callbackIp, callbackPort, callbackRouter)
	param := struct {
		Msg          string `json:"msg"`
		Callback_url string `json:"callback_url"`
	}{
		Msg:          msg,
		Callback_url: callbackurl,
	}
	data, err := json.Marshal(param)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, callUrl, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	cli := http.Client{
		Transport: configs.GlobalTransport,
	}

	_, err = cli.Do(req)
	if err != nil {
		return err
	}

	return nil
}
