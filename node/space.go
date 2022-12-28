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
	"errors"
	"fmt"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/utils"
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
		freeSpace  uint64
		minerInfo  chain.MinerInfo
		fillerInfo = make([]chain.FillerMetaInfo, configs.NumOfFillerSubmitted)
	)
	n.Logs.Space("info", fmt.Errorf(">>>>> Start task_space <<<<<"))
	time.Sleep(configs.BlockInterval)
	for {
		minerInfo, err = n.Chn.GetMinerInfo(n.Chn.GetPublicKey())
		if err != nil {
			n.Logs.Space("err", err)
		}
		if string(minerInfo.State) != chain.MINER_STATE_POSITIVE {
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

		//TODO:
		for i := 0; i < configs.NumOfFillerSubmitted; i++ {
			//Call sgx to generate a filler
			fillerInfo[i] = fillerInfo[i]
		}

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
		fillerInfo = make([]chain.FillerMetaInfo, configs.NumOfFillerSubmitted)
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
