/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/sdk-go/core/client"
)

type Bucket interface {
	Run()
}

type Node struct {
	Cfg  confile.Confile
	Log  logger.Logger
	Cach cache.Cache
	Cli  *client.Cli
}

// New is used to build a node instance
func New() *Node {
	return &Node{}
}

func (n *Node) Run() {
	go n.TaskMgt()
	configs.Ok("Start successfully")
	select {}
}
