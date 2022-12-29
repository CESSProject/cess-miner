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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

func (n *Node) task_space(ch chan<- bool) {
	defer func() {
		if err := recover(); err != nil {
			n.Logs.Pnc(utils.RecoverError(err))
		}
		ch <- true
	}()
	var (
		err        error
		txHash     string
		fpath      string
		hash       string
		freeSpace  uint64
		minerInfo  chain.MinerInfo
		fillerInfo = make([]chain.FillerMetaInfo, 0)
	)
	n.Logs.Space("info", fmt.Errorf(">>>>> Start task_space <<<<<"))
	time.Sleep(configs.BlockInterval)
	timeout := time.NewTicker(configs.TimeOut_WaitTag)
	defer timeout.Stop()
	for {

		minerInfo, err = n.Chn.GetMinerInfo(n.Chn.GetPublicKey())
		if err != nil {
			n.Logs.Space("err", err)
		}

		if string(minerInfo.State) != chain.MINER_STATE_POSITIVE {
			n.Logs.Space("err", fmt.Errorf("Miner state is %v", string(minerInfo.State)))
			time.Sleep(time.Minute)
			continue
		}

		freeSpace, err = n.calcAvailableSpace()
		if err != nil {
			n.Logs.Space("err", err)
		}

		if freeSpace < configs.SIZE_SLICE {
			n.Logs.Space("info", errors.New("The space is full"))
			time.Sleep(time.Minute)
			continue
		}

		//Call sgx to generate a filler
		fpath = filepath.Join(n.FillerDir, fmt.Sprintf("%v", time.Now().Unix()))
		err = GetFillFileReq(fpath, configs.SIZE_SLICE_KiB, configs.URL_FillFile)
		if err != nil {
			n.Logs.Space("err", err)
			time.Sleep(configs.BlockInterval)
			continue
		}

		err = GetTagReq(fpath, configs.BlockSize, n.Cfile.GetSgxPortNum(), configs.URL_GetTag, configs.URL_GetTag_Callback, n.Cfile.GetServiceAddr())
		if err != nil {
			n.Logs.Space("err", err)
			time.Sleep(configs.BlockInterval)
			continue
		}

		var tag chain.Result
		timeout.Reset(configs.TimeOut_WaitTag)
		select {
		case <-timeout.C:
			n.Logs.Space("err", fmt.Errorf("Wait tag timeout"))
		case tag = <-Ch_Tag:
		}

		if tag.Status.StatusCode != configs.SgxReportSuc {
			n.Logs.Space("err", fmt.Errorf("Recv tag status code: %v", tag.Status.StatusCode))
			continue
		}

		hash, err = utils.CalcPathSHA256(fpath)
		if err != nil {
			n.Logs.Space("err", err)
			time.Sleep(configs.BlockInterval)
			continue
		}

		os.Rename(fpath, filepath.Join(n.FillerDir, hash))
		f, err := os.Create(filepath.Join(n.FillerDir, hash+".tag"))
		if err != nil {
			n.Logs.Space("err", err)
			os.Remove(fpath)
			time.Sleep(configs.BlockInterval)
			continue
		}
		value, err := json.Marshal(tag)
		if err != nil {
			n.Logs.Space("err", err)
			time.Sleep(configs.BlockInterval)
			os.Remove(fpath)
			os.Remove(filepath.Join(n.FillerDir, hash+".tag"))
			continue
		}
		f.Write(value)
		f.Sync()
		f.Close()
		var filler chain.FillerMetaInfo
		for j := 0; j < len(hash); j++ {
			filler.Hash[j] = types.U8(hash[j])
		}
		filler.Miner_acc = types.NewAccountID(n.Chn.GetPublicKey())
		filler.Size = configs.SIZE_SLICE
		fillerInfo = append(fillerInfo, filler)

		//Submit filler info to chain
		for {
			txHash, err = n.Chn.SubmitFillerMeta(fillerInfo)
			if err != nil {
				n.Logs.Space("err", err)
			}
			if txHash != "" {
				n.Logs.Space("info", fmt.Errorf("Submit filler meta: %v", txHash))
				break
			}
			time.Sleep(configs.BlockInterval)
		}
		fillerInfo = make([]chain.FillerMetaInfo, 0)
	}
}

func (n *Node) calcAvailableSpace() (uint64, error) {
	var err error

	fileUsedSpace, err := utils.DirSize(n.FileDir)
	if err != nil {
		return 0, err
	}

	fillerUsedSpace, err := utils.DirSize(n.FillerDir)
	if err != nil {
		return 0, err
	}

	logUsedSpace, err := utils.DirSize(n.LogDir)
	if err != nil {
		return 0, err
	}

	tmpUsedSpace, err := utils.DirSize(n.TmpDir)
	if err != nil {
		return 0, err
	}

	cacheUsedSpace, err := utils.DirSize(n.CacheDir)
	if err != nil {
		return 0, err
	}

	allUsedSpace := fileUsedSpace + fillerUsedSpace + logUsedSpace + tmpUsedSpace + cacheUsedSpace
	allocatedSpace := n.Cfile.GetStorageSpace() * configs.SIZE_1GiB

	if allocatedSpace <= allUsedSpace {
		return 0, nil
	} else {
		mountInfo, err := utils.GetMountPathInfo(n.Cfile.GetMountedPath())
		if err != nil {
			return 0, err
		}
		if mountInfo.Free > configs.SIZE_SLICE {
			return allUsedSpace - allocatedSpace, nil
		}
	}

	return 0, nil
}
