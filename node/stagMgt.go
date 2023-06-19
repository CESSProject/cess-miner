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
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58"
)

func (n *Node) stagMgt(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	for {
		n.calcFileTag()
		time.Sleep(pattern.BlockInterval)
	}

}

func (n *Node) calcFileTag() {
	var roothash string
	var code uint32
	var id peer.ID

	tees, err := n.QueryTeeInfoList()
	if err != nil {
		n.Stag("err", err.Error())
		return
	}
	roothashs, err := utils.Dirs(filepath.Join(n.GetDirs().FileDir))
	if err != nil {
		n.Stag("err", err.Error())
		return
	}
	timeout := time.NewTicker(time.Minute * 2)
	defer timeout.Stop()

	for _, fileDir := range roothashs {
		roothash = filepath.Base(fileDir)
		n.Stag("info", fmt.Sprintf("Check file: %s", fileDir))
		files, err := utils.DirFiles(fileDir, 0)
		if err != nil {
			n.Stag("err", fmt.Sprintf("[DirFiles:%s] %v", fileDir, err))
			continue
		}

		for _, f := range files {
			serviceTagPath := filepath.Join(n.GetDirs().ServiceTagDir, roothash+".tag")
			_, err = os.Stat(serviceTagPath)
			if err == nil {
				continue
			}

			finfo, err := os.Stat(f)
			if err != nil {
				n.Stag("err", fmt.Sprintf("Service fragment not found: %s", f))
				continue
			}

			if finfo.Size() > pattern.FragmentSize {
				var buf = make([]byte, pattern.FragmentSize)
				fs, err := os.Open(f)
				if err != nil {
					continue
				}
				fs.Read(buf)
				fs.Close()
				fs, err = os.OpenFile(f, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
				if err != nil {
					continue
				}
				fs.Write(buf)
				fs.Sync()
				fs.Close()
				hash, err := sutils.CalcPathSHA256(f)
				if err != nil {
					continue
				}
				if hash != filepath.Base(f) {
					os.Remove(f)
					continue
				}
			}

			utils.RandSlice(tees)
			for _, t := range tees {
				teePeerId := base58.Encode([]byte(string(t.PeerId[:])))
				if !n.HasTeePeer(teePeerId) {
					continue
				}

				id, err = peer.Decode(teePeerId)
				if err != nil {
					n.Stag("err", fmt.Sprintf("[peer.Decode:%s] err: %v", teePeerId, err))
					continue
				}

				n.Stag("info", fmt.Sprintf("Send file tag request to tee: %s", teePeerId))
				code, err = n.TagReq(id, filepath.Base(f), "", pattern.BlockNumber)
				if err != nil || code != 0 {
					n.Stag("err", fmt.Sprintf("[TagReq] err: %s code: %d", err, code))
					continue
				}
				n.Stag("info", fmt.Sprintf("Send file tag file request to tee: %s", teePeerId))
				code, err = n.FileReq(id, filepath.Base(f), pb.FileType_CustomData, f)
				if err != nil || code != 0 {
					n.Stag("err", fmt.Sprintf("[FileReq] err: %s code: %d", err, code))
					continue
				}
				timeout.Reset(time.Minute * 3)
				select {
				case <-timeout.C:
					n.Stag("err", fmt.Sprintf("Waiting for service file tag timeout: %s", f))
				case filetag := <-n.GetServiceTagCh():
					n.Stag("info", fmt.Sprintf("Received a service file tag: %s", filetag))
				}
				break
			}
		}
	}
}
