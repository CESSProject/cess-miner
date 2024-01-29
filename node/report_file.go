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
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
)

func (n *Node) reportFiles(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	chainSt := n.GetChainState()
	if !chainSt {
		return
	}

	minerSt := n.GetMinerState()
	if minerSt != pattern.MINER_STATE_POSITIVE &&
		minerSt != pattern.MINER_STATE_FROZEN {
		return
	}

	var (
		ok           bool
		reReport     bool
		roothash     string
		txhash       string
		metadata     pattern.FileMetadata
		storageorder pattern.StorageOrder
	)

	n.SetReportFileFlag(true)
	defer n.SetReportFileFlag(false)

	roothashs, err := utils.Dirs(n.GetDirs().TmpDir)
	if err != nil {
		n.Report("err", fmt.Sprintf("[Dirs(TmpDir)] %v", err))
		return
	}
	for _, v := range roothashs {
		roothash = filepath.Base(v)
		n.Report("info", fmt.Sprintf("fid: %v", roothash))
		ok, err = n.Has([]byte(Cach_prefix_File + roothash))
		if err == nil {
			if ok {
				n.Report("info", fmt.Sprintf("Cach.Has: %v", roothash))
				continue
			}
		} else {
			n.Report("err", err.Error())
		}

		metadata, err = n.QueryFileMetadata(roothash)
		if err != nil {
			n.Report("err", fmt.Sprintf("QueryFileMetadata: %v", err))
			if err.Error() != pattern.ERR_Empty {
				n.Report("err", err.Error())
				time.Sleep(pattern.BlockInterval)
				continue
			}
		} else {
			var deletedFrgmentList []string
			var savedFrgment []string
			for _, segment := range metadata.SegmentList {
				for _, fragment := range segment.FragmentList {
					if !sutils.CompareSlice(fragment.Miner[:], n.GetSignatureAccPulickey()) {
						deletedFrgmentList = append(deletedFrgmentList, string(fragment.Hash[:]))
						continue
					}
					savedFrgment = append(savedFrgment, string(fragment.Hash[:]))
				}
			}

			if len(savedFrgment) == 0 {
				for _, d := range deletedFrgmentList {
					_, err = os.Stat(filepath.Join(n.GetDirs().TmpDir, roothash, d))
					if err != nil {
						continue
					}
					err = os.Remove(filepath.Join(n.GetDirs().TmpDir, roothash, d))
					if err != nil {
						if !strings.Contains(err.Error(), configs.Err_file_not_fount) {
							n.Report("err", fmt.Sprintf("[Delete TmpFile (%s.%s)] %v", roothash, d, err))
						}
					}
				}
				continue
			}

			if _, err = os.Stat(filepath.Join(n.GetDirs().FileDir, roothash)); err != nil {
				err = os.Mkdir(filepath.Join(n.GetDirs().FileDir, roothash), os.ModeDir)
				if err != nil {
					n.Report("err", fmt.Sprintf("[Mkdir.FileDir(%s)] %v", roothash, err))
					continue
				}
			}
			for i := 0; i < len(savedFrgment); i++ {
				_, err = os.Stat(filepath.Join(n.GetDirs().TmpDir, roothash, savedFrgment[i]))
				if err != nil {
					n.Report("err", fmt.Sprintf("[os.Stat(%s)] %v", roothash, err))
					continue
				}
				err = os.Rename(filepath.Join(n.GetDirs().TmpDir, roothash, savedFrgment[i]),
					filepath.Join(n.GetDirs().FileDir, roothash, savedFrgment[i]))
				if err != nil {
					n.Report("err", fmt.Sprintf("[Rename TmpDir to FileDir (%s.%s)] %v", roothash, savedFrgment[i], err))
					continue
				}
			}

			err = n.Put([]byte(Cach_prefix_File+roothash), nil)
			if err != nil {
				n.Report("err", fmt.Sprintf("[Cach.Put(%s.%s)] %v", roothash, savedFrgment, err))
			}

			for _, d := range deletedFrgmentList {
				err = os.Remove(filepath.Join(n.GetDirs().TmpDir, roothash, d))
				if err != nil {
					if !strings.Contains(err.Error(), configs.Err_file_not_fount) {
						n.Report("err", fmt.Sprintf("[Delete TmpFile (%s.%s)] %v", roothash, d, err))
					}
				}
			}

			continue
		}

		storageorder, err = n.QueryStorageOrder(roothash)
		if err != nil {
			if err.Error() != pattern.ERR_Empty {
				n.Report("err", err.Error())
			}
			continue
		}

		reReport = true
		for _, completeMiner := range storageorder.CompleteList {
			if sutils.CompareSlice(completeMiner.Miner[:], n.GetSignatureAccPulickey()) {
				reReport = false
			}
		}

		if !reReport {
			continue
		}
		var sucCount uint8

		var sucIndex = make([]uint8, 0)
		for idx := uint8(0); idx < uint8(pattern.DataShards+pattern.ParShards); idx++ {
			sucCount = 0
			for i := 0; i < len(storageorder.SegmentList); i++ {
				for j := 0; j < len(storageorder.SegmentList[i].FragmentHash); j++ {
					if j == int(idx) {
						fstat, err := os.Stat(
							filepath.Join(
								n.GetDirs().TmpDir, roothash,
								string(storageorder.SegmentList[i].FragmentHash[j][:]),
							),
						)
						if err != nil {
							break
						}
						if fstat.Size() != pattern.FragmentSize {
							break
						}
						sucCount++
						break
					}
				}
			}
			if sucCount > 0 {
				for _, v := range storageorder.CompleteList {
					if uint8(v.Index) == uint8(idx+1) {
						sucCount = 0
						break
					}
				}
				if sucCount > 0 {
					sucIndex = append(sucIndex, idx+1)
				}
			}
		}

		if len(sucIndex) == 0 {
			continue
		}

		n.Report("info", fmt.Sprintf("Will report %s", roothash))
		for _, v := range sucIndex {
			txhash, err = n.ReportFile(v, roothash)
			if err != nil {
				n.Report("err", fmt.Sprintf("[%s] File reporting failed: [%s] %v", roothash, txhash, err))
				continue
			}
			n.Report("info", fmt.Sprintf("[%s] File reported successfully: %s", roothash, txhash))
			break
		}
	}
}
