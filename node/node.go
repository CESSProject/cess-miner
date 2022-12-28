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

package node

import (
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/db"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/serve"
	"github.com/gin-gonic/gin"
)

type Bucket interface {
	Run()
}

type Node struct {
	Ser       serve.IServer
	Cfile     confile.IConfile
	Chn       chain.IChain
	Logs      logger.ILog
	Cach      db.ICache
	CallBack  *gin.Engine
	FillerDir string
	FileDir   string
	TmpDir    string
}

// New is used to build a node instance
func New() *Node {
	return &Node{}
}

func (n *Node) Run() {
	// Start subtask manager
	go n.CoroutineMgr()
	// Start Service
	n.Ser.Serve()
}
