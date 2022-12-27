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

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/db"
	"github.com/CESSProject/cess-bucket/pkg/logger"
)

// FileRouter
type DownRouter struct {
	BaseRouter
	Chain   chain.IChain
	Logs    logger.ILog
	Cach    db.ICache
	FileDir string
	TmpDir  string
}

type MsgDown struct {
	Token     string `json:"token"`
	SliceHash string `json:"slicehash"`
	FileSize  int64  `json:"filesize"`
	Index     uint32 `json:"index"`
}

// FileRouter Handle
func (d *DownRouter) Handle(ctx context.CancelFunc, request IRequest) {
	fmt.Println("Call DownRouter Handle from client : msgId=", request.GetMsgID())

	if request.GetMsgID() != Msg_Down {
		fmt.Println("MsgId error")
		ctx()
		return
	}

	var msg MsgDown
	err := json.Unmarshal(request.GetData(), &msg)
	if err != nil {
		fmt.Println("Msg format error")
		ctx()
		return
	}

	ok, _ := d.Cach.Has([]byte(TokenKey_Token + msg.Token))
	if !ok {
		request.GetConnection().SendMsg(Msg_Forbidden, nil)
		return
	}

	fpath := filepath.Join(d.FileDir, msg.SliceHash)
	_, err = os.Stat(fpath)
	if err != nil {
		request.GetConnection().SendMsg(Msg_NotFound, nil)
		return
	}

	fs, err := os.Open(fpath)
	if err != nil {
		fmt.Println("OpenFile  error")
		request.GetConnection().SendMsg(Msg_ServerErr, nil)
		return
	}
	defer fs.Close()

	fs.Seek(int64(msg.Index), 0)
	var buf = make([]byte, configs.SIZE_1MiB)
	fs.Read(buf)

	err = request.GetConnection().SendMsg(Msg_OK, buf)
	if err != nil {
		fmt.Println(err)
	}
}
