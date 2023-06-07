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
	"github.com/ethereum/go-ethereum/common/math"
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
		if n.Key != nil && n.Key.Spk.N != nil {
			time.Sleep(pattern.BlockInterval)
		}

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
		for {
			select {
			case <-timeout.C:
				n.Space("err", fmt.Sprintf("Requset timeout: %s", peerid))
				break
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
}

func (n *Node) requsetIdlefile() ([]byte, string, error) {
	var err error
	var teePeerId string
	var id peer.ID

	teelist, err := n.getTeeSortedByTime()
	if err != nil {
		return nil, teePeerId, err
	}

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
			_, err = n.IdleReq(id, pattern.FragmentSize, pattern.BlockNumber, n.GetStakingPublickey(), sign)
			if err != nil {
				continue
			}
			return tee.ControllerAccount[:], teePeerId, nil
		}
	}

	return nil, teePeerId, err
}

func (n *Node) getTeeSortedByTime() ([]pattern.TeeWorkerInfo, error) {
	teelist, err := n.QueryTeeInfoList()
	if err != nil {
		return nil, err
	}
	var result = make([]pattern.TeeWorkerInfo, 0)
	var newTee = make(map[string]int64, 0)
	err = n.deepCopyPeers(&newTee, &n.TeePeer)
	if err != nil {
		return teelist, nil
	}
	var minTee string
	var minTime int64 = math.MaxInt64
	for len(newTee) > 1 {
		for k, v := range newTee {
			if minTime > v {
				minTime = v
				minTee = k
			}
		}
		for _, v := range teelist {
			if minTee == base58.Encode([]byte(string(v.PeerId[:]))) {
				result = append(result, v)
				break
			}
		}
		delete(newTee, minTee)
	}
	return result, nil
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
