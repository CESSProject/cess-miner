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

	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/cess-miner/pkg/logger"
	"github.com/CESSProject/cess-miner/pkg/utils"
	"github.com/CESSProject/cess_pois/pois"
	"github.com/CESSProject/p2p-go/out"
)

func GenIdle(l *logger.Lg, prover *pois.Prover, r *RunningState, workspace string, useSpace uint64, ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			l.Pnc(utils.RecoverError(err))
		}
	}()

	decSpace, validSpace, usedSpace, lockSpace := r.GetMinerSpaceInfo()
	if (validSpace + usedSpace + lockSpace) >= decSpace {
		l.Space("info", "The declared space has been authenticated")
		time.Sleep(time.Minute * 10)
		return
	}

	configSpace := useSpace * pattern.SIZE_1GiB
	if configSpace < minSpace {
		l.Space("err", "The configured space is less than the minimum space requirement")
		time.Sleep(time.Minute * 10)
		return
	}

	if (validSpace + usedSpace + lockSpace) > (configSpace - minSpace) {
		l.Space("info", "The space for authentication has reached the configured space size")
		time.Sleep(time.Hour)
		return
	}

	dirfreeSpace, err := utils.GetDirFreeSpace(workspace)
	if err != nil {
		l.Space("err", fmt.Sprintf("[GetDirFreeSpace] %v", err))
		time.Sleep(time.Minute)
		return
	}

	if dirfreeSpace < minSpace {
		l.Space("err", fmt.Sprintf("The disk space is less than %dG", minSpace/pattern.SIZE_1GiB))
		time.Sleep(time.Minute * 10)
		return
	}

	l.Space("info", "Start generating idle files")
	r.SetGenIdleFlag(true)
	err = prover.GenerateIdleFileSet()
	r.SetGenIdleFlag(false)
	if err != nil {
		if strings.Contains(err.Error(), "not enough space") {
			out.Err("Your workspace is out of capacity")
			l.Space("err", "workspace is out of capacity")
		} else {
			l.Space("err", fmt.Sprintf("[GenerateIdleFileSet] %v", err))
		}
		time.Sleep(time.Minute * 10)
		return
	}
	l.Space("info", "generate a idle file")
}
