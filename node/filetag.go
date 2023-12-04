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
		return
	}

	var hasOrder bool
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
		if utils.ContainsIpv4(v.End_point) {
			teeEndPoints = append(teeEndPoints, strings.TrimPrefix(v.End_point, "http://"))
		} else {
			teeEndPoints = append(teeEndPoints, v.End_point)
		}
	}

	for _, fileDir := range roothashs {
		metadata, err := n.QueryFileMetadata(filepath.Base(fileDir))
		if err != nil {
			if err.Error() != pattern.ERR_Empty {
				n.Report("err", fmt.Sprintf("[QueryFileMetadata] %v", err))
				continue
			}
		} else {
			var deletedFrgmentList []string
			var savedFrgmentList = make(map[string]struct{}, 0)
			for _, segment := range metadata.SegmentList {
				for _, fragment := range segment.FragmentList {
					if !sutils.CompareSlice(fragment.Miner[:], n.GetSignatureAccPulickey()) {
						deletedFrgmentList = append(deletedFrgmentList, string(fragment.Hash[:]))
					} else {
						savedFrgmentList[string(fragment.Hash[:])] = struct{}{}
					}
				}
			}
			for _, d := range deletedFrgmentList {
				if _, ok := savedFrgmentList[d]; ok {
					continue
				}
				os.Remove(filepath.Join(fileDir, d))
			}
		}

		sorder, err := n.QueryStorageOrder(filepath.Base(fileDir))
		if err != nil {
			if err.Error() != pattern.ERR_Empty {
				n.Report("err", fmt.Sprintf("[QueryStorageOrder] %v", err))
				continue
			}
			hasOrder = false
		} else {
			hasOrder = true
			if uint8(sorder.Stage) != configs.OrserState_CalcTag {
				continue
			}
		}

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

			if !hasOrder {
				_, err = n.GenerateRestoralOrder(filepath.Base(fileDir), fragmentHash)
				if err != nil {
					n.Restore("err", fmt.Sprintf("[GenerateRestoralOrder] %v", err))
					continue
				}
				n.Put([]byte(Cach_prefix_MyLost+fragmentHash), nil)
				continue
			}

			buf, err := os.ReadFile(f)
			if err != nil {
				n.Stag("err", fmt.Sprintf("ReadFile: %s", f))
				continue
			}

			if len(buf) < pattern.FragmentSize {
				n.Stag("err", fmt.Sprintf("Fragment Size: %d < %d", len(buf), pattern.FragmentSize))
				continue
			}
			utils.RandSlice(teeEndPoints)
			for i := 0; i < len(teeEndPoints); i++ {
				n.Stag("info", fmt.Sprintf("Will calc file tag: %v", fragmentHash))
				n.Stag("info", fmt.Sprintf("Will calc file tag roothash: %v", filepath.Base(fileDir)))
				n.Stag("info", fmt.Sprintf("Will use tee: %v", teeEndPoints[i]))

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
					if strings.Contains(err.Error(), "no such file") {
						_, err = n.GenerateRestoralOrder(filepath.Base(fileDir), fragmentHash)
						if err != nil {
							n.Restore("err", fmt.Sprintf("[GenerateRestoralOrder] %v", err))
							continue
						}
						n.Put([]byte(Cach_prefix_MyLost+fragmentHash), nil)
					}
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
