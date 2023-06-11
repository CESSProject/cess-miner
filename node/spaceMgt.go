/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/sdk-go/core/pattern"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58"
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
	var peerid string
	var blockheight uint32
	var teepuk []byte
	var tSatrt int64
	var idlefile pattern.IdleFileMeta

	n.Space("info", ">>>>> Start spaceMgt task")

	timeout := time.NewTimer(time.Duration(time.Minute * 2))
	defer timeout.Stop()

	for {
		for n.GetChainState() {

			teepuk, peerid, err = n.requsetIdlefile()
			if err != nil {
				n.Space("err", err.Error())
				continue
			}

			tSatrt = time.Now().Unix()

			n.Space("info", fmt.Sprintf("Requset a idle file to: %s", peerid))

			spacePath = ""
			tagPath = ""

			timeout.Reset(time.Duration(time.Minute * 2))
			for err == nil {
				select {
				case <-timeout.C:
					n.Space("err", fmt.Sprintf("Requset timeout: %s", peerid))
					err = errors.New("timeout")
				case spacePath = <-n.GetIdleDataCh():
				case tagPath = <-n.GetIdleTagCh():
				}

				if tagPath != "" && spacePath != "" {
					break
				}
			}

			if tagPath == "" || spacePath == "" {
				continue
			}

			n.SaveAndUpdateTeePeer(peerid, time.Now().Unix()-tSatrt)

			n.Space("info", fmt.Sprintf("Receive a idle file tag: %s", tagPath))
			n.Space("info", fmt.Sprintf("Receive a idle file: %s", spacePath))

			filehash, err = utils.CalcPathSHA256(spacePath)
			if err != nil {
				n.Space("err", err.Error())
				os.Remove(spacePath)
				os.Remove(tagPath)
				continue
			}

			os.Rename(spacePath, filepath.Join(n.GetDirs().IdleDataDir, filehash))
			os.Rename(tagPath, filepath.Join(n.GetDirs().IdleTagDir, filehash+".tag"))

			n.Space("info", fmt.Sprintf("Idle file %s hash: %s", spacePath, filehash))

			idlefile.BlockNum = pattern.BlockNumber
			idlefile.Hash = filehash
			idlefile.MinerAcc = n.GetStakingPublickey()
			txhash, err = n.SubmitIdleFile(teepuk, []pattern.IdleFileMeta{idlefile})
			if err != nil {
				n.Space("err", fmt.Sprintf("Submit idlefile metadata err: %v", err.Error()))
				if txhash != "" {
					err = n.Put([]byte(fmt.Sprintf("%s%s", Cach_prefix_idle, filehash)), []byte(fmt.Sprintf("%s", txhash)))
					if err != nil {
						n.Space("err", fmt.Sprintf("Record idlefile [%s] failed [%v]", filehash, err))
						continue
					}
				}
				n.Space("err", fmt.Sprintf("Submit idlefile [%s] err [%s] %v", filehash, txhash, err))
				continue
			}

			n.Space("info", fmt.Sprintf("Submit idle file %s suc: %s", filehash, txhash))

			blockheight, err = n.QueryBlockHeight(txhash)
			if err != nil {
				err = n.Put([]byte(fmt.Sprintf("%s%s", Cach_prefix_idle, filehash)), []byte(fmt.Sprintf("%s", txhash)))
				if err != nil {
					n.Space("err", fmt.Sprintf("Record idlefile [%s] failed [%v]", filehash, err))
				}
				continue
			}

			err = n.Put([]byte(fmt.Sprintf("%s%s", Cach_prefix_idle, filepath.Base(spacePath))), []byte(fmt.Sprintf("%d", blockheight)))
			if err != nil {
				n.Space("err", fmt.Sprintf("Record idlefile [%s] failed [%v]", filehash, err))
				continue
			}

			n.Space("info", fmt.Sprintf("Record idle file %s suc: %d", filehash, blockheight))
		}
		time.Sleep(pattern.BlockInterval)
	}
}

func (n *Node) requsetIdlefile() ([]byte, string, error) {
	var err error
	var teePeerId string
	var id peer.ID

	teelist, err := n.QueryTeeInfoList()
	if err != nil {
		return nil, teePeerId, err
	}
	utils.RandSlice(teelist)
	sign, err := n.Sign(n.GetPeerPublickey())
	if err != nil {
		return nil, teePeerId, err
	}

	for _, tee := range teelist {
		teePeerId = base58.Encode([]byte(string(tee.PeerId[:])))
		configs.Tip(fmt.Sprintf("Query a tee: %s", teePeerId))
		if n.HasTeePeer(teePeerId) {
			id, err = peer.Decode(teePeerId)
			if err != nil {
				continue
			}
			n.Space("info", fmt.Sprintf("Will req tee: %s", teePeerId))
			_, err = n.IdleReq(id, pattern.FragmentSize, pattern.BlockNumber, n.GetStakingPublickey(), sign)
			if err != nil {
				continue
			}
			return tee.ControllerAccount[:], teePeerId, nil
		}
	}

	return nil, teePeerId, err
}
