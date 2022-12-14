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
	"sync"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/db"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/utils"
)

// FileRouter
type FileRouter struct {
	BaseRouter
	Chain   chain.IChain
	Logs    logger.Logger
	Cach    db.ICache
	FileDir string
}

type MsgFile struct {
	Token    string `json:"token"`
	RootHash string `json:"roothash"`
	FileHash string `json:"filehash"`
	FileSize int64  `json:"filesize"`
	LastSize int64  `json:"lastsize"`
	LastFile bool   `json:"lastfile"`
	Data     []byte `json:"data"`
}

var sendFileBufPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, configs.SIZE_1MiB)
	},
}

// FileRouter Handle
func (f *FileRouter) Handle(ctx context.CancelFunc, request IRequest) {
	fmt.Println("Call FileRouter Handle")
	fmt.Println("recv from client : msgId=", request.GetMsgID())

	if request.GetMsgID() != Msg_File {
		fmt.Println("MsgId error")
		ctx()
		return
	}

	var msg MsgFile
	err := json.Unmarshal(request.GetData(), &msg)
	if err != nil {
		fmt.Println("Msg format error")
		ctx()
		return
	}

	if !Tokens.Update(msg.Token) {
		request.GetConnection().SendMsg(Msg_Forbidden, nil)
		return
	}

	fpath := filepath.Join(f.FileDir, msg.FileHash)
	finfo, err := os.Stat(fpath)
	if err != nil {
		request.GetConnection().SendBuffMsg(Msg_ServerErr, nil)
		return
	}

	if finfo.Size() == configs.SIZE_SLICE {
		hash, _ := utils.CalcPathSHA256(fpath)
		if hash == msg.FileHash {
			request.GetConnection().SendBuffMsg(Msg_OK_FILE, nil)
			return
		}
	} else if finfo.Size() > configs.SIZE_SLICE {
		request.GetConnection().SendMsg(Msg_ClientErr, nil)
		return
	}

	fs, err := os.OpenFile(msg.FileHash, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		fmt.Println("OpenFile  error")
		ctx()
		return
	}
	defer fs.Close()

	fs.Write(msg.Data)
	err = fs.Sync()
	if err != nil {
		fmt.Println("Sync  error")
		ctx()
		return
	}

	if msg.LastFile {
		finfo, err = fs.Stat()
		if finfo.Size() == msg.LastSize {
			request.GetConnection().SendMsg(Msg_OK_FILE, nil)
			//
			appendBuf := make([]byte, configs.SIZE_SLICE-msg.LastSize)
			fs.Write(appendBuf)
			fs.Sync()
			return
		}
	}
	err = request.GetConnection().SendMsg(Msg_OK, nil)
	if err != nil {
		fmt.Println(err)
	}
}
