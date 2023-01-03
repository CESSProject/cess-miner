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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/db"
	"github.com/CESSProject/cess-bucket/pkg/logger"
)

// FileRouter
type ConfirmRouter struct {
	BaseRouter
	Chain   chain.IChain
	Logs    logger.ILog
	Cach    db.ICache
	FileDir string
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

	fpath := filepath.Join(c.FileDir, msg.SliceHash)
	_, err = os.Stat(fpath)
	if err != nil {
		fmt.Println("file not found: ", fpath)
		request.GetConnection().SendMsg(Msg_ClientErr, nil)
		return
	}

	//TODO
	//Call sgx to generate sign

	var sliceSum chain.SliceSummary
	b, err := json.Marshal(sliceSum)
	if err != nil {
		fmt.Println("Marshal sliceSum err ", fpath)
		request.GetConnection().SendMsg(Msg_ServerErr, nil)
		return
	}

	err = request.GetConnection().SendMsg(Msg_OK, b)
	if err != nil {
		fmt.Println(err)
	}
}
