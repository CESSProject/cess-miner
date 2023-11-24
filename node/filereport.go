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

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
)

func (n *Node) reportFiles(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	if n.state.Load() == configs.State_Offline {
		return
	}

	var (
		reReport     bool
		roothash     string
		txhash       string
		metadata     pattern.FileMetadata
		storageorder pattern.StorageOrder
	)

	n.Report("info", ">>>>> start reportFiles <<<<<")

	roothashs, err := utils.Dirs(n.GetDirs().TmpDir)
	if err != nil {
		n.Report("err", fmt.Sprintf("[Dirs] %v", err))
		return
	}
	n.Report("info", fmt.Sprintf("roothashs: %v", roothashs))

	for _, v := range roothashs {
		roothash = filepath.Base(v)
		n.Report("info", fmt.Sprintf("roothash: %v", roothash))
		metadata, err = n.QueryFileMetadata(roothash)
		if err != nil {
			if err.Error() != pattern.ERR_Empty {
				n.Report("err", fmt.Sprintf("[QueryFileMetadata] %v", err))
				return
			}
		} else {
			if _, err = os.Stat(filepath.Join(n.GetDirs().TmpDir, roothash)); err == nil {
				err = RenameDir(filepath.Join(n.GetDirs().TmpDir, roothash), filepath.Join(n.GetDirs().FileDir, roothash))
				if err != nil {
					n.Report("err", fmt.Sprintf("[RenameDir %s] %v", roothash, err))
					continue
				}
				n.Put([]byte(Cach_prefix_metadata+roothash), []byte(fmt.Sprintf("%v", metadata.Completion)))
			}
			continue
		}

		storageorder, err = n.QueryStorageOrder(roothash)
		if err != nil {
			if err.Error() != pattern.ERR_Empty {
				n.Report("err", fmt.Sprintf("[QueryStorageOrder] %v", err))
				return
			}
			n.Report("err", fmt.Sprintf("[QueryStorageOrder] %v", err))
			n.Report("err", fmt.Sprintf("[%s] will delete files", roothash))
			//os.RemoveAll(v)
			continue
		}
		reReport = true
		for _, completeMiner := range storageorder.CompleteInfo {
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
						fstat, err := os.Stat(filepath.Join(n.GetDirs().TmpDir, roothash, string(storageorder.SegmentList[i].FragmentHash[j][:])))
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
				sucIndex = append(sucIndex, idx+1)
			}
		}

		if len(sucIndex) == 0 {
			continue
		}

		n.Report("info", fmt.Sprintf("Will report %s", roothash))
		for _, v := range sucIndex {
			txhash, err = n.ReportFile(v, roothash)
			if err != nil {
				n.Report("err", fmt.Sprintf("[%s] File transfer report failed: [%s] %v", roothash, txhash, err))
				continue
			}
			n.Report("info", fmt.Sprintf("[%s] File transfer reported successfully: %s", roothash, txhash))
			break
		}
	}
}

func RenameDir(oldDir, newDir string) error {
	files, err := utils.DirFiles(oldDir, 0)
	if err != nil {
		return err
	}
	fstat, err := os.Stat(newDir)
	if err != nil {
		err = os.MkdirAll(newDir, pattern.DirMode)
		if err != nil {
			return err
		}
	} else {
		if !fstat.IsDir() {
			return fmt.Errorf("%s not a dir", newDir)
		}
	}

	for _, v := range files {
		name := filepath.Base(v)
		err = os.Rename(filepath.Join(oldDir, name), filepath.Join(newDir, name))
		if err != nil {
			return err
		}
	}
	return nil
}
