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

	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/pkg/errors"
)

// replaceMgr
// func (n *Node) replaceMgr(ch chan<- bool) {
// 	defer func() {
// 		ch <- true
// 		if err := recover(); err != nil {
// 			n.Pnc(utils.RecoverError(err))
// 		}
// 	}()

// 	var err error

// 	n.Replace("info", ">>>>> Start replaceMgt <<<<<")

// 	tikSpace := time.NewTicker(time.Hour)
// 	defer tikSpace.Stop()

// 	for {
// 		select {
// 		case <-tikSpace.C:
// 			err = n.resizeSpace()
// 			if err != nil {
// 				n.Replace("err", err.Error())
// 			}
// 		}
// 	}
// }

func (n *Node) resizeSpace() error {
	var err error
	var txhash string
	var allSpace = make([]string, 0)
	allSpace, err = n.Cache.QueryPrefixKeyList(Cach_prefix_idle)
	if err != nil {
		return errors.Wrapf(err, "[QueryPrefixKeyList]")
	}
	for _, v := range allSpace {
		_, err = n.QueryFillerMap(v)
		if err != nil {
			if err.Error() == pattern.ERR_Empty {
				os.Remove(filepath.Join(n.GetDirs().IdleDataDir, v))
				os.Remove(filepath.Join(n.GetDirs().IdleTagDir, v+".tag"))
				n.Delete([]byte(Cach_prefix_idle + v))
				continue
			}
			return errors.Wrapf(err, "[QueryFillerMap]")
		}
		_, err = os.Stat(filepath.Join(n.GetDirs().IdleDataDir, v))
		if err != nil {
			os.Remove(filepath.Join(n.GetDirs().IdleTagDir, v+".tag"))
			txhash, err = n.DeleteFiller(v)
			if err != nil {
				n.Replace("err", err.Error())
			} else {
				n.Replace("info", fmt.Sprintf("delete %v suc: %v", v, txhash))
				n.Delete([]byte(Cach_prefix_idle + v))
			}
			continue
		}
		_, err = os.Stat(filepath.Join(n.GetDirs().IdleTagDir, v+".tag"))
		if err != nil {
			os.Remove(filepath.Join(n.GetDirs().IdleDataDir, v))
			txhash, err = n.DeleteFiller(v)
			if err != nil {
				n.Replace("err", err.Error())
			} else {
				n.Replace("info", fmt.Sprintf("delete %v suc: %v", v, txhash))
				n.Delete([]byte(Cach_prefix_idle + v))
			}
			continue
		}
	}
	return nil
}

func (n *Node) replaceFiller() {
	count, err := n.QueryPendingReplacements(n.GetStakingPublickey())
	if err != nil {
		if err.Error() != pattern.ERR_Empty {
			n.Replace("err", err.Error())
		}
		return
	}

	if count == 0 {
		return
	}

	if count > MaxReplaceFiles {
		count = MaxReplaceFiles
	}
	files, err := SelectIdleFiles(n.GetDirs().IdleDataDir, count)
	if err != nil {
		n.Replace("err", err.Error())
		return
	}

	if len(files) == 0 {
		return
	}

	txhash, _, err := n.ReplaceFile(files)
	if err != nil {
		n.Replace("err", err.Error())
		return
	}

	n.Replace("info", fmt.Sprintf("Replace files: %v suc: [%s]", files, txhash))
	for i := 0; i < len(files); i++ {
		os.Remove(filepath.Join(n.GetDirs().IdleDataDir, files[i]))
		os.Remove(filepath.Join(n.GetDirs().IdleTagDir, files[i]+".tag"))
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
