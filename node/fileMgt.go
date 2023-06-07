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
	"github.com/CESSProject/p2p-go/pb"
	"github.com/CESSProject/sdk-go/core/pattern"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58"
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
			b, err := n.Get([]byte(Cach_prefix_report + roothash))
			if err == nil {
				t, err := strconv.ParseInt(string(b), 10, 64)
				if err != nil {
					n.Delete([]byte(Cach_prefix_report + roothash))
					continue
				}
				tnow := time.Now().Unix()
				if tnow > t && (tnow-t) < int64(180) {
					metadata, err = n.QueryFileMetadata(roothash)
					if err != nil {
						if err.Error() != pattern.ERR_Empty {
							n.Report("err", err.Error())
							continue
						}
					} else {
						if metadata.State == Active {
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
					continue
				}
			}

			n.Report("info", fmt.Sprintf("Will report %s", roothash))

			storageorder, err = n.QueryStorageOrder(roothash)
			if err != nil {
				if err.Error() == pattern.ERR_Empty {
					metadata, err = n.QueryFileMetadata(roothash)
					if err != nil {
						if err.Error() == pattern.ERR_Empty {
							os.RemoveAll(v)
							continue
						}
						n.Report("err", err.Error())
						continue
					}
					if metadata.State == Active {
						err = RenameDir(filepath.Join(n.GetDirs().TmpDir, roothash), filepath.Join(n.GetDirs().FileDir, roothash))
						if err != nil {
							n.Report("err", err.Error())
							continue
						}
						n.Delete([]byte(Cach_prefix_report + roothash))
						n.Put([]byte(Cach_prefix_metadata+roothash), nil)
						continue
					}
				}
				n.Report("err", err.Error())
				continue
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

			txhash, failed, err := n.ReportFiles([]string{roothash})
			if err != nil {
				n.Report("err", err.Error())
				continue
			}

			if failed == nil {
				n.Report("info", fmt.Sprintf("Report file [%s] suc: %s", roothash, txhash))
				err = n.Put([]byte(Cach_prefix_report+roothash), []byte(fmt.Sprintf("%v", time.Now().Unix())))
				if err != nil {
					n.Report("info", fmt.Sprintf("Report file [%s] suc, record failed: %v", roothash, err))
				}
				n.Report("info", fmt.Sprintf("Report file [%s] suc, record suc", roothash))
				continue
			}
			n.Report("err", fmt.Sprintf("Report file [%s] failed: %s", roothash, txhash))
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

	for _, f := range roothashs {
		roothash = filepath.Base(f)
		files, err := utils.DirFiles(filepath.Join(n.GetDirs().FileDir, roothash), 0)
		if err != nil {
			continue
		}
		for _, f := range files {
			serviceTagPath := filepath.Join(n.GetDirs().ServiceTagDir, roothash+".tag")
			_, err = os.Stat(serviceTagPath)
			if err == nil {
				n.Report("err", fmt.Sprintf("Service tag not found: %s", serviceTagPath))
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

			var id peer.ID
			for _, t := range tees {
				teePeerId := base58.Encode([]byte(string(t.PeerId[:])))
				if n.HasTeePeer(teePeerId) {
					id, err = peer.Decode(teePeerId)
					if err != nil {
						continue
					}
				}

				code, err = n.TagReq(id, filepath.Base(f), "", pattern.BlockNumber)
				if err != nil {
					fmt.Println("Tag req err:", err)
				}
				if code != 0 {
					continue
				}
				code, err = n.FileReq(id, filepath.Base(f), pb.FileType_CustomData, f)
				if err != nil {
					continue
				}
				break
			}
		}
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
