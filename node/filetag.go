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

	chainSt := n.GetChainState()
	if chainSt {
		return
	}

	minerSt := n.GetMinerState()
	if minerSt != pattern.MINER_STATE_POSITIVE &&
		minerSt != pattern.MINER_STATE_FROZEN {
		return
	}

	var ok bool
	var recover bool
	var fid string
	var fragmentHash string

	roothashs, err := utils.Dirs(n.GetDirs().FileDir)
	if err != nil {
		n.Stag("err", fmt.Sprintf("[Dirs(%s)] %v", n.GetDirs().FileDir, err))
		return
	}

	teeEndPoints := n.GetPriorityTeeList()
	teeEndPoints = append(teeEndPoints, n.GetAllMarkerTeeEndpoint()...)

	for _, fileDir := range roothashs {
		fid = filepath.Base(fileDir)
		ok, err = n.Has([]byte(Cach_prefix_File + fid))
		if err == nil {
			if !ok {
				continue
			}
		} else {
			n.Report("err", err.Error())
			time.Sleep(time.Second)
			continue
		}

		files, err := utils.DirFiles(fileDir, 0)
		if err != nil {
			n.Stag("err", fmt.Sprintf("[DirFiles(%s)] %v", fid, err))
			time.Sleep(time.Second)
			continue
		}

		for _, f := range files {
			recover = false
			fragmentHash = filepath.Base(f)
			_, err = os.Stat(filepath.Join(n.DataDir.TagDir, fragmentHash+".tag"))
			if err == nil {
				continue
			}

			buf, err := os.ReadFile(f)
			if err != nil {
				if strings.Contains(err.Error(), "no such file") {
					recover = true
					n.Stag("err", fmt.Sprintf("[%s] Missing a file segment: %s", fid, fragmentHash))
				} else {
					n.Stag("err", fmt.Sprintf("[ReadFile(%s.%s)]: %v", fid, fragmentHash, err))
					continue
				}
			} else {
				if len(buf) != pattern.FragmentSize {
					recover = true
					n.Stag("err", fmt.Sprintf("[%s.%s] File fragment size [%d] is not equal to %d", fid, fragmentHash, len(buf), pattern.FragmentSize))
				}
			}

			if recover {
				buf, err = n.GetFragmentFromOss(fragmentHash)
				if err != nil {
					n.Stag("err", fmt.Sprintf("Recovering fragment from cess gateway failed: %v", err))
					continue
				}
				if len(buf) < pattern.FragmentSize {
					n.Stag("err", fmt.Sprintf("[%s.%s] Fragment size [%d] received from CESS gateway is wrong", fid, fragmentHash, len(buf)))
					continue
				}
				err = os.WriteFile(f, buf, os.ModePerm)
				if err != nil {
					n.Stag("err", fmt.Sprintf("[%s] [WriteFile(%s)]: %v", fid, fragmentHash, err))
					continue
				}
			}

			for i := 0; i < len(teeEndPoints); i++ {
				n.Stag("info", fmt.Sprintf("[%s] Will calc file tag: %v", fid, fragmentHash))
				n.Stag("info", fmt.Sprintf("[%s] Will use tee: %v", fid, teeEndPoints[i]))
				genTag, err := n.PoisServiceRequestGenTag(
					teeEndPoints[i],
					buf[:pattern.FragmentSize],
					filepath.Base(fileDir),
					fragmentHash,
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
				err = sutils.WriteBufToFile(buf, filepath.Join(n.DataDir.TagDir, fmt.Sprintf("%s.tag", fragmentHash)))
				if err != nil {
					n.Stag("err", fmt.Sprintf("[WriteBufToFile] err: %s", err))
					continue
				}
				n.Stag("info", fmt.Sprintf("Calc a service tag: %s", filepath.Join(n.DataDir.TagDir, fmt.Sprintf("%s.tag", fragmentHash))))
				//TODO: Wait for the tee to complete the tag calculation interface
				// n.ReportTagCalculated()
				// n.Put([]byte(Cach_prefix_Tag+fragmentHash), []byte(blocknumber))
				break
			}
		}
	}
}
