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
	"time"

	"github.com/CESSProject/cess-go-sdk/chain"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/pkg/utils"
)

func (n *Node) ReportFiles(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	roothashs, err := utils.Dirs(n.GetReportDir())
	if err != nil {
		n.Report("err", fmt.Sprintf("[Dirs(TmpDir)] %v", err))
		return
	}

	report := false
	fid := ""
	for _, file := range roothashs {
		fid = filepath.Base(file)
		report, err = n.checkfile(file)
		if err != nil {
			n.Report("err", fmt.Sprintf("[%s] check file err: %v", fid, err))
		}

		if report {
			n.Report("info", fmt.Sprintf("[%s] will report file", fid))
			err = n.reportfile(file)
			if err != nil {
				n.Report("err", fmt.Sprintf("[%s] report file err: %v", fid, err))
			}
		}

		if !n.GetCurrentRpcst() {
			return
		}
		time.Sleep(chain.BlockInterval)
	}
}

func (n *Node) checkfile(f string) (bool, error) {
	fid := filepath.Base(f)
	metadata, err := n.QueryFile(fid, -1)
	if err != nil {
		if !errors.Is(err, chain.ERR_RPC_EMPTY_VALUE) {
			return false, err
		}
		_, err = n.QueryDealMap(fid, -1)
		if err != nil {
			if !errors.Is(err, chain.ERR_RPC_EMPTY_VALUE) {
				return false, err
			}
			os.RemoveAll(filepath.Join(n.GetReportDir(), fid))
			n.Del("info", fmt.Sprintf("remove dir: %s", filepath.Join(n.GetReportDir(), fid)))
			return false, errors.New("not found")
		}
		return true, nil
	}

	var deletedFrgmentList []string
	var savedFrgment []string

	for _, segment := range metadata.SegmentList {
		for _, fragment := range segment.FragmentList {
			if sutils.CompareSlice(fragment.Miner[:], n.GetSignatureAccPulickey()) {
				savedFrgment = append(savedFrgment, string(fragment.Hash[:]))
			} else {
				deletedFrgmentList = append(deletedFrgmentList, string(fragment.Hash[:]))
			}
		}
	}

	if len(savedFrgment) == 0 {
		for _, d := range deletedFrgmentList {
			err = os.Remove(filepath.Join(n.GetReportDir(), fid, d))
			if err != nil {
				continue
			}
			n.Del("info", filepath.Join(n.GetReportDir(), fid, d))
		}
		return false, nil
	}

	if _, err = os.Stat(filepath.Join(n.GetFileDir(), fid)); err != nil {
		err = os.Mkdir(filepath.Join(n.GetFileDir(), fid), configs.FileMode)
		if err != nil {
			return false, err
		}
	}

	for i := 0; i < len(savedFrgment); i++ {
		_, err = os.Stat(filepath.Join(n.GetReportDir(), fid, savedFrgment[i]))
		if err != nil {
			return false, err
		}
		err = os.Rename(filepath.Join(n.GetReportDir(), fid, savedFrgment[i]),
			filepath.Join(n.GetFileDir(), fid, savedFrgment[i]))
		if err != nil {
			return false, err
		}
	}

	for _, d := range deletedFrgmentList {
		err = os.Remove(filepath.Join(n.GetReportDir(), fid, d))
		if err != nil {
			continue
		}
		n.Del("info", filepath.Join(n.GetReportDir(), fid, d))
	}
	return false, nil
}

func (n *Node) reportfile(f string) error {
	fid := filepath.Base(f)
	storageorder, err := n.QueryDealMap(fid, -1)
	if err != nil {
		if err.Error() != chain.ERR_Empty {
			return err
		}
		return nil
	}

	reReport := true
	for _, completeMiner := range storageorder.CompleteList {
		if sutils.CompareSlice(completeMiner.Miner[:], n.GetSignatureAccPulickey()) {
			reReport = false
			break
		}
	}

	if !reReport {
		return nil
	}

	var sucCount int
	var sucIndex = make([]uint8, 0)
	for idx := uint8(0); idx < uint8(chain.DataShards+chain.ParShards); idx++ {
		sucCount = 0
		for i := 0; i < len(storageorder.SegmentList); i++ {
			fstat, err := os.Stat(filepath.Join(n.GetReportDir(), fid, string(storageorder.SegmentList[i].FragmentHash[idx][:])))
			if err != nil {
				break
			}
			if fstat.Size() != chain.FragmentSize {
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
		txhash, err = n.TransferReport(v, fid)
		if err != nil {
			n.Report("err", fmt.Sprintf("[%s] report err: %v bloakhash: %s", fid, err, txhash))
			continue
		}
		n.Report("info", fmt.Sprintf("[%s] report successful, blockhash: %s", fid, txhash))
		break
	}
	return nil
}
