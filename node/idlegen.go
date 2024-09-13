/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"strings"
	"time"

	"github.com/CESSProject/cess-go-sdk/chain"
	out "github.com/CESSProject/cess-miner/pkg/fout"
	"github.com/CESSProject/cess-miner/pkg/utils"
)

func (n *Node) GenIdle(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	decSpace, validSpace, usedSpace, lockSpace := n.GetMinerSpaceInfo()
	if (validSpace + usedSpace + lockSpace) >= decSpace {
		n.Space("info", "The declared space has been authenticated")
		time.Sleep(time.Minute * 10)
		return
	}

	configSpace := n.ReadUseSpace() * chain.SIZE_1GiB
	if configSpace < minSpace {
		n.Space("err", "The configured space is less than the minimum space requirement")
		time.Sleep(time.Minute * 10)
		return
	}

	if (validSpace + usedSpace + lockSpace) > (configSpace - minSpace) {
		n.Space("info", "The space for authentication has reached the configured space size")
		time.Sleep(time.Hour)
		return
	}

	dirfreeSpace, err := utils.GetDirFreeSpace(n.GetRootDir())
	if err != nil {
		n.Space("err", fmt.Sprintf("[GetDirFreeSpace] %v", err))
		time.Sleep(time.Minute)
		return
	}

	if dirfreeSpace < minSpace {
		n.Space("err", fmt.Sprintf("The disk space is less than %dG", minSpace/chain.SIZE_1GiB))
		time.Sleep(time.Minute * 10)
		return
	}

	n.Space("info", "Start generating idle files")
	n.SetGeneratingIdle(true)
	err = n.GenerateIdleFileSet()
	n.SetGeneratingIdle(false)
	if err != nil {
		if strings.Contains(err.Error(), "not enough space") {
			out.Err("Your workspace is out of capacity")
			n.Space("err", "workspace is out of capacity")
		} else {
			n.Space("err", fmt.Sprintf("[GenerateIdleFileSet] %v", err))
		}
		time.Sleep(time.Minute * 10)
		return
	}
	n.Space("info", "generate a idle file")
}
