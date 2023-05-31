/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/sdk-go/core/pattern"
)

// replaceMgr
func (n *Node) replaceMgr(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	var err error
	var txhash string
	var count uint32
	var spacedir = filepath.Join(n.Workspace(), configs.SpaceDir)

	n.Replace("info", ">>>>> Start replaceMgr task")

	for {
		if err != nil && n.Key != nil && n.Key.Spk.N != nil {
			time.Sleep(time.Minute)
		}

		count, err = n.QueryPendingReplacements(n.GetStakingPublickey())
		if err != nil {
			n.Replace("err", err.Error())
			time.Sleep(time.Minute)
			continue
		}

		if count == 0 {
			time.Sleep(time.Minute)
			continue
		}

		if count > MaxReplaceFiles {
			count = MaxReplaceFiles
		}
		files, err := SelectIdleFiles(spacedir, count)
		if err != nil {
			n.Replace("err", err.Error())
			time.Sleep(time.Minute)
			continue
		}

		txhash, _, err = n.ReplaceFile(files)
		if err != nil {
			n.Replace("err", err.Error())
			time.Sleep(pattern.BlockInterval)
			continue
		}

		n.Replace("info", fmt.Sprintf("Replace files: %v suc: [%s]", files, txhash))
		for i := 0; i < len(files); i++ {
			os.Remove(filepath.Join(spacedir, files[i]))
		}
	}
}

func SelectIdleFiles(dir string, count uint32) ([]string, error) {
	files, err := utils.DirFiles(dir, count)
	if err != nil {
		return nil, err
	}
	var result = make([]string, 0)
	for i := 0; i < len(files); i++ {
		result = append(result, filepath.Base(files[i]))
	}
	return result, nil
}
