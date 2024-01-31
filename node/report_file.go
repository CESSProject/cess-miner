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

	n.SetReportFileFlag(true)
	defer n.SetReportFileFlag(false)

	roothashs, err := utils.Dirs(n.GetDirs().TmpDir)
	if err != nil {
		n.Report("err", fmt.Sprintf("[Dirs(TmpDir)] %v", err))
		return
	}
	for _, file := range roothashs {
		err = n.reportFile(file)
		if err != nil {
			n.Report("err", fmt.Sprintf("[%s] [reportFile] %v", filepath.Base(file), err))
		}
		time.Sleep(time.Second)
	}
}

func (n *Node) reportFile(file string) error {
	var (
		ok            bool
		err           error
		reReport      bool
		txhash        string
		queryFileMeta bool
		metadata      pattern.FileMetadata
		storageorder  pattern.StorageOrder
	)
	queryFileMeta = true
	fid := filepath.Base(file)
	n.Report("info", fmt.Sprintf("[%s] will report file", fid))

	ok, _ = n.Has([]byte(Cach_prefix_File + fid))
	if ok {
		n.Report("info", fmt.Sprintf("[%s] already reported file", fid))
		if _, err = os.Stat(filepath.Join(n.GetDirs().FileDir, fid)); err == nil {
			return nil
		}
		metadata, err = n.QueryFileMetadata(fid)
		if err == nil {
			queryFileMeta = false
		} else {
			return err
		}
	} else {
		metadata, err = n.QueryFileMetadata(fid)
		if err == nil {
			queryFileMeta = false
			for _, segment := range metadata.SegmentList {
				for _, fragment := range segment.FragmentList {
					if sutils.CompareSlice(fragment.Miner[:], n.GetSignatureAccPulickey()) {
						err = n.Put([]byte(Cach_prefix_File+fid), nil)
						if err != nil {
							n.Report("err", fmt.Sprintf("[%s] Cach.Put: %v", fid, err))
						}
					}
				}
			}
		}
	}
	var deletedFrgmentList []string
	var savedFrgment []string
	if queryFileMeta {
		metadata, err = n.QueryFileMetadata(fid)
		if err != nil {
			n.Report("err", fmt.Sprintf("[%s] QueryFileMetadata: %v", fid, err))
			if err.Error() != pattern.ERR_Empty {
				time.Sleep(pattern.BlockInterval)
				return nil
			}
		}
	} else {
		for _, segment := range metadata.SegmentList {
			for _, fragment := range segment.FragmentList {
				if sutils.CompareSlice(fragment.Miner[:], n.GetSignatureAccPulickey()) {
					n.Report("info", fmt.Sprintf("[%s] fragment should be save: %s", fid, string(fragment.Hash[:])))
					savedFrgment = append(savedFrgment, string(fragment.Hash[:]))
				} else {
					n.Report("info", fmt.Sprintf("[%s] fragment should be delete: %s", fid, string(fragment.Hash[:])))
					deletedFrgmentList = append(deletedFrgmentList, string(fragment.Hash[:]))
				}
			}
		}

		if len(savedFrgment) == 0 {
			for _, d := range deletedFrgmentList {
				_, err = os.Stat(filepath.Join(n.GetDirs().TmpDir, fid, d))
				if err != nil {
					n.Report("info", fmt.Sprintf("[%s] delete the fragment [%s] failed: %v", fid, d, err))
					continue
				}
				err = os.Remove(filepath.Join(n.GetDirs().TmpDir, fid, d))
				if err != nil {
					n.Report("err", fmt.Sprintf("[%s] delete the fragment [%s] failed: %v", fid, d, err))
					continue
				}
				n.Report("info", fmt.Sprintf("[%s] deleted the fragment: %s", fid, d))
			}
			return nil
		}

		if _, err = os.Stat(filepath.Join(n.GetDirs().FileDir, fid)); err != nil {
			err = os.Mkdir(filepath.Join(n.GetDirs().FileDir, fid), os.ModeDir)
			if err != nil {
				n.Report("err", fmt.Sprintf("[%s] Mkdir: %v", fid, err))
				return nil
			}
		}

		for i := 0; i < len(savedFrgment); i++ {
			_, err = os.Stat(filepath.Join(n.GetDirs().TmpDir, fid, savedFrgment[i]))
			if err != nil {
				n.Report("err", fmt.Sprintf("[%s] os.Stat(%s): %v", fid, savedFrgment[i], err))
				return nil
			}
			err = os.Rename(filepath.Join(n.GetDirs().TmpDir, fid, savedFrgment[i]),
				filepath.Join(n.GetDirs().FileDir, fid, savedFrgment[i]))
			if err != nil {
				n.Report("err", fmt.Sprintf("[%s] move [%s] to filedir: %v", fid, savedFrgment[i], err))
				return nil
			}
			n.Report("info", fmt.Sprintf("[%s] move [%s] to filedir", fid, savedFrgment[i]))
		}

		err = n.Put([]byte(Cach_prefix_File+fid), nil)
		if err != nil {
			n.Report("err", fmt.Sprintf("[%s] Cach.Put: %v", fid, err))
		}

		for _, d := range deletedFrgmentList {
			err = os.Remove(filepath.Join(n.GetDirs().TmpDir, fid, d))
			if err != nil {
				n.Report("err", fmt.Sprintf("[%s] delete the fragment [%s] failed: %v", fid, d, err))
				continue
			}
			n.Report("info", fmt.Sprintf("[%s] deleted the fragment: %s", fid, d))
		}
		return nil
	}

	storageorder, err = n.QueryStorageOrder(fid)
	if err != nil {
		n.Report("err", err.Error())
		time.Sleep(pattern.BlockInterval)
		return nil
	}

	reReport = true
	for _, completeMiner := range storageorder.CompleteList {
		if sutils.CompareSlice(completeMiner.Miner[:], n.GetSignatureAccPulickey()) {
			reReport = false
			break
		}
	}

	if !reReport {
		n.Report("info", fmt.Sprintf("[%s] already report", fid))
		return nil
	}

	var sucCount int
	var sucIndex = make([]uint8, 0)
	for idx := uint8(0); idx < uint8(pattern.DataShards+pattern.ParShards); idx++ {
		sucCount = 0
		n.Report("info", fmt.Sprintf("[%s] check the %d batch fragments", fid, idx))
		for i := 0; i < len(storageorder.SegmentList); i++ {
			fstat, err := os.Stat(
				filepath.Join(n.GetDirs().TmpDir, fid, string(storageorder.SegmentList[i].FragmentHash[idx][:])),
			)
			if err != nil {
				break
			}
			if fstat.Size() != pattern.FragmentSize {
				break
			}
			sucCount++
			n.Report("info", fmt.Sprintf("[%s] the %d segment's %d fragment saved", fid, i, idx))
		}
		if sucCount == len(storageorder.SegmentList) {
			for _, v := range storageorder.CompleteList {
				if uint8(v.Index) == uint8(idx+1) {
					sucCount = 0
					break
				}
			}
			if sucCount > 0 {
				sucIndex = append(sucIndex, (idx + 1))
			}
		}
	}

	n.Report("info", fmt.Sprintf("[%s] successfully stored index: %v", fid, sucIndex))

	if len(sucIndex) == 0 {
		return nil
	}

	for _, v := range sucIndex {
		n.Report("info", fmt.Sprintf("[%s] will report index: %d", fid, v))
		txhash, err = n.ReportFile(v, fid)
		if err != nil {
			n.Report("err", fmt.Sprintf("[%s] report failed: [%s] %v", fid, txhash, err))
			continue
		}
		n.Report("info", fmt.Sprintf("[%s] reported successfully: %s", fid, txhash))
		return nil
	}
	return nil
}
