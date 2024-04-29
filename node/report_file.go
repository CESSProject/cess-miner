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
	"sync"
	"time"

	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/cess-go-sdk/core/sdk"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/pkg/logger"
	"github.com/CESSProject/cess-miner/pkg/utils"
)

var (
	reportedFileLock *sync.Mutex
	reportedFile     map[string]struct{}
)

func init() {
	reportedFileLock = new(sync.Mutex)
	reportedFile = make(map[string]struct{}, 0)
}

func ReportFiles(ch chan<- bool, cli sdk.SDK, r *RunningState, l *logger.Lg, fileDir, tmpDir string) {
	defer func() { ch <- true }()

	roothashs, err := utils.Dirs(tmpDir)
	if err != nil {
		l.Report("err", fmt.Sprintf("[Dirs(TmpDir)] %v", err))
		return
	}

	ok := false
	fid := ""
	for _, file := range roothashs {
		fid = filepath.Base(file)
		reportedFileLock.Lock()
		_, ok = reportedFile[fid]
		reportedFileLock.Unlock()
		if ok {
			l.Report("info", fmt.Sprintf("[%s] prepare to check the file", fid))
			err = check_file(cli, l, file, fileDir, tmpDir)
			if err != nil {
				l.Report("err", fmt.Sprintf("[%s] check the file err: %v", fid, err))
			}
		} else {
			r.SetReportFileFlag(true)
			l.Report("info", fmt.Sprintf("[%s] prepare to report the file", fid))
			err = report_file(cli, l, file, fileDir, tmpDir)
			if err != nil {
				l.Report("err", fmt.Sprintf("[%s] report file err: %v", fid, err))
			}
			r.SetReportFileFlag(false)
		}
		if !cli.GetChainState() {
			return
		}
		time.Sleep(pattern.BlockInterval)
	}
}

func check_file(cli sdk.SDK, l logger.Logger, f string, fileDir, tmpDir string) error {
	fid := filepath.Base(f)
	metadata, err := cli.QueryFileMetadata(fid)
	if err != nil {
		if !errors.Is(err, pattern.ERR_RPC_EMPTY_VALUE) {
			return err
		}
		sorder, err := cli.QueryStorageOrder(fid)
		if err != nil {
			if !errors.Is(err, pattern.ERR_RPC_EMPTY_VALUE) {
				return err
			}
		}
		for _, v := range sorder.CompleteList {
			if sutils.CompareSlice(v.Miner[:], cli.GetSignatureAccPulickey()) {
				return nil
			}
		}
		reportedFileLock.Lock()
		delete(reportedFile, fid)
		reportedFileLock.Unlock()
		os.RemoveAll(f)
		l.Del("info", fmt.Sprintf("delete folder: %s", f))
		return err
	}

	var deletedFrgmentList []string
	var savedFrgment []string

	for _, segment := range metadata.SegmentList {
		for _, fragment := range segment.FragmentList {
			if sutils.CompareSlice(fragment.Miner[:], cli.GetSignatureAccPulickey()) {
				savedFrgment = append(savedFrgment, string(fragment.Hash[:]))
			} else {
				deletedFrgmentList = append(deletedFrgmentList, string(fragment.Hash[:]))
			}
		}
	}

	if len(savedFrgment) == 0 {
		os.RemoveAll(f)
		l.Del("info", fmt.Sprintf("Delete folder: %s", f))
		return nil
	}

	if _, err = os.Stat(filepath.Join(fileDir, fid)); err != nil {
		err = os.Mkdir(filepath.Join(fileDir, fid), configs.FileMode)
		if err != nil {
			return err
		}
	}

	for i := 0; i < len(savedFrgment); i++ {
		_, err = os.Stat(filepath.Join(tmpDir, fid, savedFrgment[i]))
		if err != nil {
			return err
		}
		err = os.Rename(filepath.Join(tmpDir, fid, savedFrgment[i]),
			filepath.Join(fileDir, fid, savedFrgment[i]))
		if err != nil {
			return err
		}
	}

	for _, d := range deletedFrgmentList {
		err = os.Remove(filepath.Join(tmpDir, fid, d))
		if err != nil {
			continue
		}
		l.Del("info", filepath.Join(tmpDir, fid, d))
	}
	return nil
}

func report_file(cli sdk.SDK, l logger.Logger, f string, fileDir, tmpDir string) error {
	fid := filepath.Base(f)
	storageorder, err := cli.QueryStorageOrder(fid)
	if err != nil {
		if err.Error() != pattern.ERR_Empty {
			return err
		}
		reportedFileLock.Lock()
		reportedFile[fid] = struct{}{}
		reportedFileLock.Unlock()
		return nil
	}

	reReport := true
	for _, completeMiner := range storageorder.CompleteList {
		if sutils.CompareSlice(completeMiner.Miner[:], cli.GetSignatureAccPulickey()) {
			reReport = false
			break
		}
	}

	if !reReport {
		reportedFileLock.Lock()
		reportedFile[fid] = struct{}{}
		reportedFileLock.Unlock()
		return nil
	}

	var sucCount int
	var sucIndex = make([]uint8, 0)
	for idx := uint8(0); idx < uint8(pattern.DataShards+pattern.ParShards); idx++ {
		sucCount = 0
		for i := 0; i < len(storageorder.SegmentList); i++ {
			fstat, err := os.Stat(filepath.Join(tmpDir, fid, string(storageorder.SegmentList[i].FragmentHash[idx][:])))
			if err != nil {
				break
			}
			if fstat.Size() != pattern.FragmentSize {
				break
			}
			sucCount++
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

	if len(sucIndex) == 0 {
		return nil
	}
	txhash := ""
	for _, v := range sucIndex {
		txhash, err = cli.ReportFile(v, fid)
		if err != nil {
			l.Report("err", fmt.Sprintf("[%s] report err: %v bloakhash: %s", fid, err, txhash))
			continue
		}
		l.Report("info", fmt.Sprintf("[%s] report successful, blockhash: %s", fid, txhash))
		reportedFileLock.Lock()
		reportedFile[fid] = struct{}{}
		reportedFileLock.Unlock()
		break
	}
	return nil
}
