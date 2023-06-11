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
	"github.com/CESSProject/p2p-go/pb"
	"github.com/CESSProject/sdk-go/core/pattern"
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
	}

}

func (n *Node) calcFileTag() {
	var roothash string
	var code uint32
	tees, err := n.QueryTeeInfoList()
	if err != nil {
		n.Report("err", err.Error())
		return
	}
	roothashs, err := utils.Dirs(filepath.Join(n.GetDirs().FileDir))
	if err != nil {
		n.Report("err", err.Error())
		return
	}
	n.Report("info", fmt.Sprintf("Service files: %s", roothashs))
	for _, f := range roothashs {
		roothash = filepath.Base(f)
		n.Report("info", fmt.Sprintf("Service file: %s", roothash))
		files, err := utils.DirFiles(filepath.Join(n.GetDirs().FileDir, roothash), 0)
		if err != nil {
			n.Report("err", fmt.Sprintf("[DirFiles] %v", err))
			continue
		}

		for _, f := range files {
			serviceTagPath := filepath.Join(n.GetDirs().ServiceTagDir, filepath.Base(f)+".tag")
			n.Report("info", fmt.Sprintf("Service file tag: %s", serviceTagPath))
			_, err = os.Stat(serviceTagPath)
			if err == nil {
				n.Report("err", fmt.Sprintf("Found a service tag: %s", serviceTagPath))
				continue
			}

			finfo, err := os.Stat(f)
			if err != nil {
				n.Report("err", fmt.Sprintf("Service file not found: %s", f))
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
				hash, err := utils.CalcFileHash(f)
				if err != nil {
					continue
				}
				if hash != filepath.Base(f) {
					os.Remove(f)
					continue
				}
			}

			utils.RandSlice(tees)
			var id peer.ID
			for _, t := range tees {
				teePeerId := base58.Encode([]byte(string(t.PeerId[:])))
				if n.HasTeePeer(teePeerId) {
					id, err = peer.Decode(teePeerId)
					if err != nil {
						continue
					}
				}
				n.Report("info", fmt.Sprintf("Send file tag request to tee: %s", teePeerId))
				code, err = n.TagReq(id, filepath.Base(f), "", pattern.BlockNumber)
				if err != nil || code != 0 {
					n.Report("err", fmt.Sprintf("[TagReq] err: %s code: %d", err, code))
					continue
				}
				n.Report("info", fmt.Sprintf("Send file tag file request to tee: %s", teePeerId))
				code, err = n.FileReq(id, filepath.Base(f), pb.FileType_CustomData, f)
				if err != nil || code != 0 {
					n.Report("err", fmt.Sprintf("[FileReq] err: %s code: %d", err, code))
					continue
				}
				break
			}
		}
	}
}
