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
	"github.com/CESSProject/cess-bucket/pkg/db"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/utils"
)

// FileRouter
type FileRouter struct {
	BaseRouter
	Logs    logger.ILog
	Cach    db.ICache
	FileDir string
	TmpDir  string
}

type MsgFile struct {
	Token     string `json:"token"`
	RootHash  string `json:"roothash"`
	SliceHash string `json:"slicehash"`
	FileSize  int64  `json:"filesize"`
	Lastfile  bool   `json:"lastfile"`
	Data      []byte `json:"data"`
}

// FileRouter Handle
func (f *FileRouter) Handle(ctx context.CancelFunc, request IRequest) {
	fmt.Println("Call FileRouter Handle msgId=", request.GetMsgID())

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

	if msg.FileSize > int64(configs.SIZE_SLICE) {
		request.GetConnection().SendMsg(Msg_ClientErr, nil)
		return
	}

	ok, _ := f.Cach.Has([]byte(TokenKey_Token + msg.Token))
	if !ok {
		request.GetConnection().SendMsg(Msg_Forbidden, nil)
		return
	}

	fpath := filepath.Join(f.TmpDir, msg.SliceHash)
	finfo, err := os.Stat(fpath)
	if err == nil {
		if finfo.Size() == configs.SIZE_SLICE {
			hash, _ := utils.CalcPathSHA256(fpath)
			if hash == msg.SliceHash {
				request.GetConnection().SendBuffMsg(Msg_OK_FILE, nil)
				return
			} else {
				os.Remove(fpath)
				request.GetConnection().SendMsg(Msg_ClientErr, nil)
				return
			}
		} else if finfo.Size() > configs.SIZE_SLICE {
			request.GetConnection().SendMsg(Msg_ClientErr, nil)
			os.Remove(fpath)
			return
		}
	}
	fs, err := os.OpenFile(fpath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		fmt.Println("OpenFile  error")
		request.GetConnection().SendMsg(Msg_ServerErr, nil)
		return
	}

	fs.Write(msg.Data)
	err = fs.Sync()
	if err != nil {
		fs.Close()
		fmt.Println("Sync  error")
		request.GetConnection().SendMsg(Msg_ServerErr, nil)
		return
	}

	finfo, err = fs.Stat()
	if finfo.Size() == msg.FileSize {
		// Fill to 512MB
		if msg.FileSize < configs.SIZE_SLICE {
			appendBuf := make([]byte, configs.SIZE_SLICE-msg.FileSize)
			fs.Write(appendBuf)
			fs.Sync()
		}
		fs.Close()
		hash, _ := utils.CalcPathSHA256(fpath)
		if hash == msg.SliceHash {
			request.GetConnection().SendBuffMsg(Msg_OK_FILE, nil)
			// TODO:
			// Calc tag call sgx
		} else {
			os.Remove(fpath)
			request.GetConnection().SendMsg(Msg_ClientErr, nil)
		}
		return
	}

	err = request.GetConnection().SendMsg(Msg_OK, nil)
	if err != nil {
		fmt.Println(err)
	}
	fs.Close()
}
