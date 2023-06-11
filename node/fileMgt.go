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
	"strconv"
	"time"

	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/sdk-go/core/pattern"
)

// fileMgr
func (n *Node) fileMgt(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	var roothash string
	var failfile bool
	var storageorder pattern.StorageOrder
	var metadata pattern.FileMetadata

	n.Report("info", ">>>>> Start fileMgt task")

	for {
		time.Sleep(pattern.BlockInterval)

		n.calcFileTag()

		roothashs, err := utils.Dirs(filepath.Join(n.GetDirs().TmpDir))
		if err != nil {
			n.Report("err", err.Error())
			time.Sleep(time.Minute)
			continue
		}

		for _, v := range roothashs {
			failfile = false
			roothash = filepath.Base(v)
			metadata, err = n.QueryFileMetadata(roothash)
			if err != nil {
				n.Report("err", err.Error())
				if err.Error() != pattern.ERR_Empty {
					continue
				}
			} else {
				if _, err = os.Stat(filepath.Join(n.GetDirs().TmpDir, roothash)); err == nil {
					err = RenameDir(filepath.Join(n.GetDirs().TmpDir, roothash), filepath.Join(n.GetDirs().FileDir, roothash))
					if err != nil {
						n.Report("err", err.Error())
						continue
					}
					n.Delete([]byte(Cach_prefix_report + roothash))
					n.Put([]byte(Cach_prefix_metadata+roothash), []byte(fmt.Sprintf("%v", metadata.Completion)))
				}
				continue
			}

			n.Report("info", fmt.Sprintf("Will report %s", roothash))

			storageorder, err = n.QueryStorageOrder(roothash)
			if err != nil {
				n.Report("err", err.Error())
				if err.Error() == pattern.ERR_Empty {
					// delete
				}
				continue
			}

			b, err := n.Get([]byte(Cach_prefix_report + roothash))
			if err == nil {
				count, err := strconv.ParseInt(string(b), 10, 64)
				if err != nil {
					n.Report("err", err.Error())
				} else {
					if count == int64(storageorder.Count) {
						n.Report("info", fmt.Sprintf("Alreaey report: %s", roothash))
						continue
					}
				}
			}

			var assignedFragmentHash = make([]string, 0)
			for i := 0; i < len(storageorder.AssignedMiner); i++ {
				assignedAddr, _ := utils.EncodeToCESSAddr(storageorder.AssignedMiner[i].Account[:])
				if n.GetStakingAcc() == assignedAddr {
					for j := 0; j < len(storageorder.AssignedMiner[i].Hash); j++ {
						assignedFragmentHash = append(assignedFragmentHash, string(storageorder.AssignedMiner[i].Hash[j][:]))
					}
				}
			}

			n.Report("info", fmt.Sprintf("Query [%s], files: %v", roothash, assignedFragmentHash))
			failfile = false
			for i := 0; i < len(assignedFragmentHash); i++ {
				n.Report("info", fmt.Sprintf("Check: %s", filepath.Join(n.GetDirs().TmpDir, roothash, assignedFragmentHash[i])))
				fstat, err := os.Stat(filepath.Join(n.GetDirs().TmpDir, roothash, assignedFragmentHash[i]))
				if err != nil || fstat.Size() != pattern.FragmentSize {
					failfile = true
					break
				}
				n.Report("info", "Check success")
			}
			if failfile {
				continue
			}

			txhash, _, err := n.ReportFiles([]string{roothash})
			if err != nil {
				n.Report("err", err.Error())
				continue
			}

			n.Report("info", fmt.Sprintf("Report file [%s] suc: %s", roothash, txhash))
			err = n.Put([]byte(Cach_prefix_report+roothash), []byte(fmt.Sprintf("%v", storageorder.Count)))
			if err != nil {
				n.Report("info", fmt.Sprintf("Report file [%s] suc, record failed: %v", roothash, err))
			}
			n.Report("info", fmt.Sprintf("Report file [%s] suc, record suc", roothash))
		}

		// roothashs, err = utils.Dirs(filepath.Join(n.Workspace(), n.GetDirs().FileDir))
		// if err != nil {
		// 	n.Report("err", err.Error())
		// 	continue
		// }

		// for _, v := range roothashs {
		// 	roothash = filepath.Base(v)
		// 	_, err = n.QueryFileMetadata(roothash)
		// 	if err != nil {
		// 		if err.Error() == pattern.ERR_Empty {
		// 			os.RemoveAll(v)
		// 		}
		// 		continue
		// 	}
		// }
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

	return os.RemoveAll(oldDir)
}
