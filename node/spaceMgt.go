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
	"github.com/libp2p/go-libp2p/core/peer"
)

// spaceMgt is a subtask for managing spaces
func (n *Node) spaceMgt(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	var err error
	var spacePath string
	var tagPath string
	var txhash string
	var filehash string
	var blockheight uint32
	var teepuk []byte
	var idlefile pattern.IdleFileMeta

	n.Space("info", ">>>>> Start spaceMgt task")

	timeout := time.NewTimer(time.Duration(time.Minute * 2))
	defer timeout.Stop()

	for {
		if err != nil {
			time.Sleep(time.Minute)
		}

		teepuk, err = n.requsetIdlefile()
		if err != nil {
			n.Space("err", err.Error())
			continue
		}

		spacePath = ""
		tagPath = ""

		timeout.Reset(time.Duration(time.Minute * 2))
		for {
			select {
			case <-timeout.C:
				break
			case spacePath = <-n.GetIdleDataCh():
			case tagPath = <-n.GetIdleTagCh():
			}

			if tagPath != "" && spacePath != "" {
				break
			}
		}

		configs.Tip(fmt.Sprintf("Receive a tag: %s", tagPath))
		configs.Tip(fmt.Sprintf("Receive a idlefile: %s", spacePath))

		if tagPath == "" || spacePath == "" {
			n.Space("err", spacePath)
			n.Space("err", tagPath)
			continue
		}

		filehash, err = utils.CalcPathSHA256(spacePath)
		if err != nil {
			n.Space("err", err.Error())
			os.Remove(spacePath)
			os.Remove(tagPath)
			continue
		}

		os.Rename(spacePath, filepath.Join(n.GetDirs().IdleDataDir, filehash))
		os.Rename(tagPath, filepath.Join(n.GetDirs().IdleTagDir, filehash+".tag"))

		idlefile.BlockNum = pattern.BlockNumber
		idlefile.Hash = filehash
		idlefile.MinerAcc = n.GetStakingPublickey()
		txhash, err = n.SubmitIdleFile(teepuk, []pattern.IdleFileMeta{idlefile})
		if err != nil {
			configs.Err(fmt.Sprintf("Submit idlefile metadata err: %v", err))
			if txhash != "" {
				err = n.Put([]byte(fmt.Sprintf("%s%s", Cach_prefix_idle, filepath.Base(spacePath))), []byte(fmt.Sprintf("%s", txhash)))
				if err != nil {
					n.Space("err", fmt.Sprintf("Record idlefile [%s] failed [%v]", filepath.Base(spacePath), err))
					continue
				}
			}
			n.Space("err", fmt.Sprintf("Submit idlefile [%s] err [%s] %v", filepath.Base(spacePath), txhash, err))
			continue
		}
		configs.Ok(fmt.Sprintf("Submit idlefile metadata suc: %s", txhash))

		blockheight, err = n.QueryBlockHeight(txhash)
		if err != nil {
			err = n.Put([]byte(fmt.Sprintf("%s%s", Cach_prefix_idle, filepath.Base(spacePath))), []byte(fmt.Sprintf("%s", txhash)))
			if err != nil {
				n.Space("err", fmt.Sprintf("Record idlefile [%s] failed [%v]", filepath.Base(spacePath), err))
			}
			continue
		}

		err = n.Put([]byte(fmt.Sprintf("%s%s", Cach_prefix_idle, filepath.Base(spacePath))), []byte(fmt.Sprintf("%d", blockheight)))
		if err != nil {
			n.Space("err", fmt.Sprintf("Record idlefile [%s] failed [%v]", filepath.Base(spacePath), err))
			continue
		}

		n.Space("info", fmt.Sprintf("Submit idlefile [%s] suc [%s]", filepath.Base(spacePath), txhash))
	}
}

func (n *Node) requsetIdlefile() ([]byte, error) {
	var err error
	var teePeerId string
	var id peer.ID

	teelist, err := n.QueryTeeInfoList()
	if err != nil {
		return nil, err
	}

	sign, err := n.Sign(n.GetPeerPublickey())
	if err != nil {
		return nil, err
	}

	for _, tee := range teelist {
		teePeerId, err = n.GetPeerIdFromPubkey([]byte(string(tee.PeerId[:])))
		if err != nil {
			continue
		}
		if n.Has(teePeerId) {
			id, err = peer.Decode(teePeerId)
			if err != nil {
				continue
			}
			_, err = n.IdleReq(id, pattern.FragmentSize, pattern.BlockNumber, n.GetStakingPublickey(), sign)
			if err != nil {
				continue
			}
			return tee.ControllerAccount[:], nil
		}
	}

	return nil, err
}

// func generateSpace_8MB(dir string) (string, error) {
// 	fpath := filepath.Join(dir, fmt.Sprintf("%v", time.Now().UnixNano()))
// 	defer os.Remove(fpath)
// 	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0)
// 	if err != nil {
// 		return "", err
// 	}

// 	for i := uint64(0); i < 2048; i++ {
// 		f.WriteString(utils.RandStr(4095) + "\n")
// 	}
// 	err = f.Sync()
// 	if err != nil {
// 		os.Remove(fpath)
// 		return "", err
// 	}
// 	f.Close()

// 	hash, err := utils.CalcFileHash(fpath)
// 	if err != nil {
// 		return "", err
// 	}

// 	hashpath := filepath.Join(dir, hash)
// 	err = os.Rename(fpath, hashpath)
// 	if err != nil {
// 		return "", err
// 	}
// 	return hashpath, nil
// }
