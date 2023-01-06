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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/db"
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
		err       error
		minerInfo chain.MinerInfo
	)
	n.Logs.Space("info", fmt.Errorf(">>>>> Start task_space <<<<<"))
	time.Sleep(configs.BlockInterval)

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
		n.ManagementRegion()
		time.Sleep(configs.BlockInterval)
		n.AutonomousRegion()
		time.Sleep(configs.BlockInterval)
	}
}

func (n *Node) ManagementRegion() {
	var (
		err        error
		txHash     string
		fpath      string
		hash       string
		freeSpace  uint64
		fillerInfo = make([]chain.FillerMetaInfo, 0)
	)

	files, err := utils.WorkFiles(n.FillerDir)
	if err == nil {
		for i := 0; i < len(files); i++ {
			if len(filepath.Base(files[i])) < len(chain.FileHash{}) {
				os.Remove(files[i])
			}
		}
	}

	freeSpace, err = n.CalcManagementRegionFreeSpace()
	if err != nil {
		n.Logs.Space("err", err)
	}

	if freeSpace < configs.SIZE_SLICE {
		n.Logs.Space("info", errors.New("The space is full"))
		return
	}

	//Call sgx to generate a filler
	fpath = filepath.Join(n.FillerDir, fmt.Sprintf("%v", time.Now().Unix()))
	err = GetFillFileReq(fpath, configs.SIZE_SLICE_KiB, configs.URL_FillFile)
	if err != nil {
		n.Logs.Space("err", err)
		return
	}

	err = GetTagReq(fpath, configs.BlockSize, configs.SegmentSize, n.Cfile.GetSgxPortNum(), configs.URL_GetTag, configs.URL_GetTag_Callback, n.Cfile.GetServiceAddr())
	if err != nil {
		n.Logs.Space("err", err)
		return
	}

	var tag chain.Result
	timeout := time.NewTicker(configs.TimeOut_WaitTag)
	defer timeout.Stop()
	select {
	case <-timeout.C:
		n.Logs.Space("err", fmt.Errorf("Wait tag timeout"))
	case tag = <-Ch_Tag:
	}

	if tag.Status.StatusCode != configs.SgxReportSuc {
		n.Logs.Space("err", fmt.Errorf("Recv tag status code: %v", tag.Status.StatusCode))
		return
	}

	hash, err = utils.CalcPathSHA256(fpath)
	if err != nil {
		n.Logs.Space("err", err)
		return
	}

	os.Rename(fpath, filepath.Join(n.FillerDir, hash))
	f, err := os.Create(filepath.Join(n.FillerDir, hash+".tag"))
	if err != nil {
		n.Logs.Space("err", err)
		os.Remove(fpath)
		return
	}
	value, err := json.Marshal(tag)
	if err != nil {
		n.Logs.Space("err", err)
		os.Remove(fpath)
		os.Remove(filepath.Join(n.FillerDir, hash+".tag"))
		return
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
			time.Sleep(configs.BlockInterval)
			blockheigh, err := n.Chn.GetBlockHeightByHash(txHash)
			if err == nil {
				n.Cach.Put([]byte(Cach_Blockheight+hash), utils.Int64ToBytes(int64(blockheigh)))
			}
			n.Logs.Space("info", fmt.Errorf("Submit filler meta: %v", txHash))
			break
		}
		time.Sleep(configs.BlockInterval)
	}
}

func (n *Node) AutonomousRegion() {
	var (
		ok                 bool
		key                string
		hash               string
		txHash             string
		autonomousFileMeta chain.AutonomyFileMeta
		slicePath          = make([]string, 0)
		sliceHashPath      = make([]string, 0)
	)

	freeSpace, err := n.CalcAutonomousRegionFreeSpace()
	if err != nil {
		n.Logs.Space("err", err)
		return
	}

	if freeSpace < configs.SIZE_SLICE {
		n.Logs.Space("info", errors.New("The autonomous region space is full"))
		return
	}

	files, err := utils.WorkFiles(n.Cfile.GetAutonomousRegion())
	if err != nil {
		n.Logs.Space("err", err)
		return
	}
	if len(files) > 0 {
		for i := 0; i < len(files); i++ {
			//TODO:
			//judge last access time

			fname := filepath.Base(files[i])
			ok, err = n.Cach.Has([]byte(fname))
			if ok {
				continue
			}
			ok, err = n.Cach.Has([]byte(strings.TrimSuffix(fname, filepath.Ext(fname))))
			if ok {
				continue
			}

			key = url.QueryEscape(files[i])
			_, err = n.Cach.Get([]byte(key))
			if errors.Is(err, db.NotFound) {
				fsta, err := os.Stat(files[i])
				if err != nil {
					continue
				}
				autonomousFileMeta.File_size = types.U64(fsta.Size())
				sliceNum := fsta.Size() / configs.SIZE_SLICE
				if fsta.Size()%configs.SIZE_SLICE > 0 {
					sliceNum += 1
				}
				if freeSpace < uint64(sliceNum*configs.SIZE_SLICE) {
					continue
				}
				var buf = make([]byte, configs.SIZE_SLICE)
				var num int

				hash, err = utils.CalcPathSHA256(files[i])
				if err != nil {
					n.Logs.Space("err", err)
				}

				for j := 0; j < len(hash); j++ {
					autonomousFileMeta.File_hash[i] = types.U8(hash[i])
				}

				fsrc, err := os.Open(files[i])
				if err != nil {
					n.Logs.Space("err", err)
					continue
				}
				defer fsrc.Close()
				for j := 0; j < int(sliceNum); j++ {
					var fpath = fmt.Sprintf("%v.cess%d", files[i], j)
					slicePath = append(slicePath, fpath)
					fdst, err := os.Create(fpath)
					if err != nil {
						n.Logs.Space("err", err)
						continue
					}
					fsrc.Seek(int64(j)*sliceNum, 0)
					num, _ = fsrc.Read(buf)
					fdst.Write(buf[:num])
					if (j + 1) == int(sliceNum) {
						if num < configs.SIZE_SLICE {
							var appendBuf = make([]byte, configs.SIZE_SLICE-num)
							fdst.Write(appendBuf)
						}
					}
					fdst.Sync()
					fdst.Close()
				}

				// get tag
				autonomousFileMeta.Slice = make([]chain.FileHash, 0)
				timeout := time.NewTicker(configs.TimeOut_WaitTag)
				defer timeout.Stop()
				for j := 0; j < len(slicePath); j++ {
					err = GetTagReq(slicePath[j], configs.BlockSize, configs.SegmentSize, n.Cfile.GetSgxPortNum(), configs.URL_GetTag, configs.URL_GetTag_Callback, n.Cfile.GetServiceAddr())
					if err != nil {
						n.Logs.Space("err", err)
						for _, v := range slicePath {
							os.Remove(v)
						}
						return
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
						for _, v := range slicePath {
							os.Remove(v)
						}
						return
					}

					slicehash, err := utils.CalcPathSHA256(slicePath[j])
					if err != nil {
						n.Logs.Space("err", err)
						for _, v := range slicePath {
							os.Remove(v)
						}
						return
					}
					newPath := filepath.Join(filepath.Dir(slicePath[j]), slicehash)
					os.Rename(slicePath[j], newPath)
					sliceHashPath = append(sliceHashPath, newPath)
					ftag, err := os.Create(newPath + ".tag")
					if err != nil {
						n.Logs.Space("err", err)
						for _, v := range slicePath {
							os.Remove(v)
						}
						for _, v := range sliceHashPath {
							os.Remove(v)
						}
						return
					}
					value, err := json.Marshal(tag)
					if err != nil {
						n.Logs.Space("err", err)
						for _, v := range slicePath {
							os.Remove(v)
						}
						for _, v := range sliceHashPath {
							os.Remove(v)
						}
						os.Remove(newPath + ".tag")
						return
					}
					ftag.Write(value)
					ftag.Sync()
					ftag.Close()
					var sliceid chain.FileHash
					for k := 0; k < len(slicehash); k++ {
						sliceid[k] = types.U8(slicehash[k])
					}
					autonomousFileMeta.Slice = append(autonomousFileMeta.Slice, sliceid)
				}

				n.Cach.Put([]byte(key), []byte(hash))
				for j := 0; j < len(autonomousFileMeta.Slice); j++ {
					temp := string(autonomousFileMeta.Slice[j][:])
					n.Cach.Put([]byte(temp), nil)
				}

				//Submit filler info to chain
				for {
					txHash, err = n.Chn.SubmitAutonomousFileMeta(autonomousFileMeta)
					if err != nil {
						n.Logs.Space("err", err)
					}
					if txHash != "" {
						time.Sleep(configs.BlockInterval)
						blockheigh, err := n.Chn.GetBlockHeightByHash(txHash)
						if err == nil {
							n.Cach.Put([]byte(Cach_Blockheight+hash), utils.Int64ToBytes(int64(blockheigh)))
						}
						n.Logs.Space("info", fmt.Errorf("Submit autonomous file meta: %v", txHash))
						break
					}
					time.Sleep(configs.BlockInterval)
				}
			}
		}
	}
}

func (n *Node) CalcManagementRegionFreeSpace() (uint64, error) {
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

func (n *Node) CalcAutonomousRegionFreeSpace() (uint64, error) {
	var (
		err             error
		mountedPath     = "/"
		mountedPathInfo utils.MountPathInfo
	)
	if n.Cfile.GetAutonomousRegion() != "/" {
		temp := strings.Split(n.Cfile.GetAutonomousRegion(), "/")
		mountedPath = "/" + temp[1]
	}

	mountedPathInfo, err = utils.GetMountPathInfo(mountedPath)
	if err != nil {
		mountedPathInfo, err = utils.GetMountPathInfo("/")
		if err != nil {
			return 0, err
		}
	}

	return mountedPathInfo.Free, nil
}
