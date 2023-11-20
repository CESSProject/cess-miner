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
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (n *Node) serviceTag(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	if n.state.Load() == configs.State_Offline {
		time.Sleep(time.Minute)
		return
	}

	var fragmentHash string
	var err error

	roothashs, err := utils.Dirs(filepath.Join(n.GetDirs().FileDir))
	if err != nil {
		n.Stag("err", fmt.Sprintf("[Dirs] %v", err))
		return
	}

	var teeEndPoints = make([]string, 0)

	teeList, err := n.QueryTeeWorkerList()
	if err != nil {
		n.Stag("err", fmt.Sprintf("[QueryTeeWorkerList] %v", err))
		return
	}
	for _, v := range teeList {
		teeEndPoints = append(teeEndPoints, v.End_point)
	}

	for _, fileDir := range roothashs {
		files, err := utils.DirFiles(fileDir, 0)
		if err != nil {
			n.Stag("err", fmt.Sprintf("[DirFiles] %v", err))
			continue
		}

		for _, f := range files {
			fragmentHash = filepath.Base(f)
			serviceTagPath := filepath.Join(n.DataDir.TagDir, fragmentHash+".tag")
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
			utils.RandSlice(teeEndPoints)
			for i := 0; i < len(teeEndPoints); i++ {
				n.Stag("info", fmt.Sprintf("Will use tee: %v", teeEndPoints[i]))
				genTag, err := n.PoisServiceRequestGenTag(
					strings.TrimPrefix(teeEndPoints[i], "http://"),
					buf[:pattern.FragmentSize],
					filepath.Base(f),
					"",
					time.Duration(time.Minute*10),
					grpc.WithTransportCredentials(insecure.NewCredentials()),
				)
				if err != nil {
					n.Stag("err", fmt.Sprintf("[PoisServiceRequestGenTag] %v", err))
					continue
				}
				buf, err = json.Marshal(genTag.Tag)
				if err != nil {
					n.Stag("err", fmt.Sprintf("[json.Marshal] err: %s", err))
					continue
				}
				ok, err := n.GetPodr2Key().VerifyAttest(genTag.Tag.T.Name, genTag.Tag.T.U, genTag.Tag.PhiHash, genTag.Tag.Attest, "")
				if err != nil {
					n.Stag("err", fmt.Sprintf("[VerifyAttest] err: %s", err))
					continue
				}
				if !ok {
					n.Stag("err", "VerifyAttest is false")
					continue
				}
				err = sutils.WriteBufToFile(buf, filepath.Join(n.DataDir.TagDir, filepath.Base(f)+".tag"))
				if err != nil {
					n.Stag("err", fmt.Sprintf("[WriteBufToFile] err: %s", err))
					continue
				}
				n.Stag("info", fmt.Sprintf("Calc a service tag: %s", filepath.Join(n.DataDir.TagDir, filepath.Base(f)+".tag")))
				break
			}
		}
	}
}
