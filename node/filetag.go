/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
)

func (n *Node) serviceTag(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	var fragmentHash string
	var err error

	roothashs, err := utils.Dirs(filepath.Join(n.GetDirs().FileDir))
	if err != nil {
		n.Stag("err", fmt.Sprintf("[Dirs] %v", err))
		return
	}
	teePeerIds := n.GetAllTeeWorkPeerIdString()
	for _, fileDir := range roothashs {
		files, err := utils.DirFiles(fileDir, 0)
		if err != nil {
			n.Stag("err", fmt.Sprintf("[DirFiles] %v", err))
			continue
		}

		for _, f := range files {
			fragmentHash = filepath.Base(f)
			serviceTagPath := filepath.Join(n.GetDirs().ServiceTagDir, fragmentHash+".tag")
			_, err = os.Stat(serviceTagPath)
			if err == nil {
				continue
			}

			buf, err := os.ReadFile(f)
			if err != nil {
				n.Stag("err", fmt.Sprintf("ReadFile: %s", f))
				continue
			}

			if len(buf) < pattern.FragmentSize {
				n.Stag("err", fmt.Sprintf("Fragment Size: %d < 8388608", len(buf)))
				continue
			}
			utils.RandSlice(teePeerIds)
			for i := 0; i < len(teePeerIds); i++ {
				n.Stag("info", fmt.Sprintf("Will use tee: %v", teePeerIds[i]))
				addrInfo, ok := n.GetPeer(teePeerIds[i])
				if !ok {
					n.Stag("err", fmt.Sprintf("Not found tee: %s", teePeerIds[i]))
					continue
				}
				err = n.Connect(n.GetCtxQueryFromCtxCancel(), addrInfo)
				if err != nil {
					n.Stag("err", fmt.Sprintf("Connect %s err: %v", teePeerIds[i], err))
					continue
				}
				genTag, err := n.PoisServiceRequestGenTagP2P(
					addrInfo.ID,
					buf[:pattern.FragmentSize],
					filepath.Base(f),
					"",
					time.Duration(time.Minute*10),
				)
				if err != nil {
					n.Stag("err", fmt.Sprintf("[PoisServiceRequestGenTagP2P] err: %s", err))
					continue
				}
				buf, err = json.Marshal(genTag.Tag)
				if err != nil {
					n.Stag("err", fmt.Sprintf("[json.Marshal] err: %s", err))
					continue
				}
				ok, err = n.GetPodr2Key().VerifyAttest(genTag.Tag.T.Name, genTag.Tag.T.U, genTag.Tag.PhiHash, genTag.Tag.Attest, "")
				if err != nil {
					n.Stag("err", fmt.Sprintf("[VerifyAttest] err: %s", err))
					continue
				}
				if !ok {
					n.Stag("err", "VerifyAttest is false")
					continue
				}
				err = sutils.WriteBufToFile(buf, filepath.Join(n.GetDirs().ServiceTagDir, filepath.Base(f)+".tag"))
				if err != nil {
					n.Stag("err", fmt.Sprintf("[WriteBufToFile] err: %s", err))
					continue
				}
				n.Stag("info", fmt.Sprintf("Calc a service tag: %s", filepath.Join(n.GetDirs().ServiceTagDir, filepath.Base(f)+".tag")))
				break
			}
		}
	}
}

func (n *Node) stagMgt(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()
	n.Stag("info", ">>>>> Start stagMgt <<<<<")

	var err error
	var recordErr string

	for {
		if n.GetChainState() {
			err = n.calcFileTag()
			if err != nil {
				if recordErr != err.Error() {
					n.Stag("err", err.Error())
					recordErr = err.Error()
				}
			}
		} else {
			if recordErr != pattern.ERR_RPC_CONNECTION.Error() {
				n.Stag("err", pattern.ERR_RPC_CONNECTION.Error())
				recordErr = pattern.ERR_RPC_CONNECTION.Error()
			}
		}
		time.Sleep(pattern.BlockInterval)
	}
}

func (n *Node) calcFileTag() error {
	var fragmentHash string
	var code uint32
	var err error

	roothashs, err := utils.Dirs(n.GetDirs().FileDir)
	if err != nil {
		return errors.Wrapf(err, "[Dirs]")
	}

	tees := n.GetAllTeeWorkPeerId()

	timeout := time.NewTicker(time.Minute * 5)
	defer timeout.Stop()

	for _, fileDir := range roothashs {
		files, err := utils.DirFiles(fileDir, 0)
		if err != nil {
			n.Stag("err", fmt.Sprintf("[DirFiles] %v", err))
			continue
		}

		for _, f := range files {
			fragmentHash = filepath.Base(f)
			serviceTagPath := filepath.Join(n.GetDirs().ServiceTagDir, fragmentHash+".tag")
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
				teePeerId := base58.Encode(t)
				addr, ok := n.GetPeer(teePeerId)
				if !ok {
					continue
				}
				err = n.Connect(n.GetCtxQueryFromCtxCancel(), addr)
				if err != nil {
					continue
				}

				n.Stag("info", fmt.Sprintf("Send fragment [%s] tag req to tee: %s", filepath.Base(f), teePeerId))
				code, err = n.TagReq(addr.ID, filepath.Base(f), "", 0)
				if err != nil || code != 0 {
					n.Stag("err", fmt.Sprintf("[TagReq] err: %s code: %d", err, code))
					continue
				}

				n.Stag("info", fmt.Sprintf("Send fragment [%s] file req to tee: %s", filepath.Base(f), teePeerId))
				code, err = n.FileReq(addr.ID, filepath.Base(f), pb.FileType_CustomData, f)
				if err != nil || code != 0 {
					n.Stag("err", fmt.Sprintf("[FileReq] err: %s code: %d", err, code))
					continue
				}
				timeout.Reset(time.Minute * 5)
				select {
				case <-timeout.C:
					n.Stag("err", fmt.Sprintf("Waiting for fragment tag timeout: %s", f))
				case filetag := <-n.GetServiceTagCh():
					n.Stag("info", fmt.Sprintf("Received the fragment tag: %s", filepath.Base(filetag)))
				}
				break
			}
		}
	}
	return nil
}
