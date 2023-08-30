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
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
)

func (n *Node) reportFiles(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	var (
		reReport     bool
		failfile     bool
		roothash     string
		txhash       string
		metadata     pattern.FileMetadata
		storageorder pattern.StorageOrder
	)
	roothashs, err := utils.Dirs(n.GetDirs().TmpDir)
	if err != nil {
		n.Report("err", fmt.Sprintf("[Dirs] %v", err))
		return
	}

	for _, v := range roothashs {
		failfile = false
		roothash = filepath.Base(v)
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
		for _, completeMiner := range storageorder.CompleteList {
			if sutils.CompareSlice(completeMiner[:], n.GetSignatureAccPulickey()) {
				reReport = false
			}
		}

		if !reReport {
			continue
		}

		var assignedFragmentHash = make([]string, 0)
		for i := 0; i < len(storageorder.AssignedMiner); i++ {
			if sutils.CompareSlice(storageorder.AssignedMiner[i].Account[:], n.GetSignatureAccPulickey()) {
				for j := 0; j < len(storageorder.AssignedMiner[i].Hash); j++ {
					assignedFragmentHash = append(assignedFragmentHash, string(storageorder.AssignedMiner[i].Hash[j][:]))
				}
			}
		}

		if len(assignedFragmentHash) == 0 {
			continue
		}

		failfile = false
		for i := 0; i < len(assignedFragmentHash); i++ {
			fstat, err := os.Stat(filepath.Join(n.GetDirs().TmpDir, roothash, assignedFragmentHash[i]))
			if err != nil {
				failfile = true
				break
			} else {
				if fstat.Size() != pattern.FragmentSize {
					failfile = true
					break
				}
			}
		}

		if failfile {
			continue
		}

		n.Report("info", fmt.Sprintf("Will report %s", roothash))
		txhash, _, err = n.ReportFiles([]string{roothash})
		if err != nil {
			n.Report("err", fmt.Sprintf("[ReportFiles %s] %v", roothash, err))
			continue
		}
		n.Report("info", fmt.Sprintf("Report file [%s] suc: %s", roothash, txhash))
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
