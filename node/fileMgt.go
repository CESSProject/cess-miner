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

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/CESSProject/sdk-go/core/chain"
	"github.com/CESSProject/sdk-go/core/rule"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
)

// fileMgr
func (n *Node) fileMgt(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Log.Pnc(utils.RecoverError(err))
		}
	}()

	var roothash string
	var failfile bool
	var storageorder chain.StorageOrder
	var metadata chain.FileMetadata

	_, err := n.Cli.AddMultiaddrToPearstore("/ip4/221.122.79.3/tcp/10010/p2p/12D3KooWAdyc4qPWFHsxMtXvSrm7CXNFhUmKPQdoXuKQXki69qBo", time.Hour*999)
	if err != nil {
		panic(errors.Wrapf(err, "[AddMultiaddrToPearstore]"))
	}

	for {
		n.calcFileTag()

		roothashs, err := utils.Dirs(filepath.Join(n.Cli.TmpDir))
		if err != nil {
			n.Log.Report("err", err.Error())
			time.Sleep(time.Minute)
			continue
		}

		for _, v := range roothashs {
			failfile = false
			roothash = filepath.Base(v)
			b, err := n.Cach.Get([]byte(Cach_prefix_report + roothash))
			if err == nil {
				t, err := strconv.ParseInt(string(b), 10, 64)
				if err != nil {
					n.Cach.Delete([]byte(Cach_prefix_report + roothash))
					continue
				}
				tnow := time.Now().Unix()
				if tnow > t && (tnow-t) < 180 {
					metadata, err = n.Cli.QueryFileMetadata(roothash)
					if err != nil {
						if err.Error() != chain.ERR_Empty {
							n.Log.Report("err", err.Error())
							continue
						}
					} else {
						if metadata.State == Active {
							err = RenameDir(filepath.Join(n.Cli.TmpDir, roothash), filepath.Join(n.Cli.FileDir, roothash))
							if err != nil {
								n.Log.Report("err", err.Error())
								continue
							}
							n.Cach.Delete([]byte(Cach_prefix_report + roothash))
							n.Cach.Put([]byte(Cach_prefix_metadata+roothash), []byte(fmt.Sprintf("%v", metadata.Completion)))
						}
						continue
					}
					continue
				}
			}

			n.Log.Report("info", fmt.Sprintf("Will report %s", roothash))

			storageorder, err = n.Cli.QueryStorageOrder(roothash)
			if err != nil {
				if err.Error() == chain.ERR_Empty {
					metadata, err = n.Cli.QueryFileMetadata(roothash)
					if err != nil {
						if err.Error() == chain.ERR_Empty {
							os.RemoveAll(v)
							continue
						}
						n.Log.Report("err", err.Error())
						continue
					}
					if metadata.State == Active {
						err = RenameDir(filepath.Join(n.Cli.TmpDir, roothash), filepath.Join(n.Cli.FileDir, roothash))
						if err != nil {
							n.Log.Report("err", err.Error())
							continue
						}
						n.Cach.Delete([]byte(Cach_prefix_report + roothash))
						n.Cach.Put([]byte(Cach_prefix_metadata+roothash), nil)
						continue
					}
				}
				n.Log.Report("err", err.Error())
				continue
			}

			var assignedFragmentHash = make([]string, 0)
			for i := 0; i < len(storageorder.AssignedMiner); i++ {
				assignedAddr, _ := utils.EncodeToCESSAddr(storageorder.AssignedMiner[i].Account[:])
				if n.Cfg.GetAccount() == assignedAddr {
					for j := 0; j < len(storageorder.AssignedMiner[i].Hash); j++ {
						assignedFragmentHash = append(assignedFragmentHash, string(storageorder.AssignedMiner[i].Hash[j][:]))
					}
				}
			}

			n.Log.Report("info", fmt.Sprintf("Query [%s], files: %v", roothash, assignedFragmentHash))
			failfile = false
			for i := 0; i < len(assignedFragmentHash); i++ {
				fmt.Println("Check: ", filepath.Join(n.Cli.TmpDir, roothash, assignedFragmentHash[i]))
				fstat, err := os.Stat(filepath.Join(n.Cli.TmpDir, roothash, assignedFragmentHash[i]))
				if err != nil || fstat.Size() != rule.FragmentSize {
					fmt.Println(err)
					fmt.Println(fstat.Size())
					failfile = true
					break
				}
			}
			if failfile {
				continue
			}

			txhash, failed, err := n.Cli.ReportFiles([]string{roothash})
			if err != nil {
				n.Log.Report("err", err.Error())
				continue
			}

			if failed == nil {
				n.Log.Report("info", fmt.Sprintf("Report file [%s] suc: %s", roothash, txhash))
				err = n.Cach.Put([]byte(Cach_prefix_report+roothash), []byte(fmt.Sprintf("%v", time.Now().Unix())))
				if err != nil {
					n.Log.Report("info", fmt.Sprintf("Report file [%s] suc, record failed: %v", roothash, err))
				}
				n.Log.Report("info", fmt.Sprintf("Report file [%s] suc, record suc", roothash))
				continue
			}
			n.Log.Report("err", fmt.Sprintf("Report file [%s] failed: %s", roothash, txhash))
		}

		roothashs, err = utils.Dirs(filepath.Join(n.Cli.Workspace(), n.Cli.FileDir))
		if err != nil {
			n.Log.Report("err", err.Error())
			continue
		}

		for _, v := range roothashs {
			roothash = filepath.Base(v)
			_, err = n.Cli.QueryFileMetadata(roothash)
			if err != nil {
				if err.Error() == chain.ERR_Empty {
					os.RemoveAll(v)
				}
				continue
			}
		}
		time.Sleep(configs.BlockInterval)
	}
}

func (n *Node) calcFileTag() {
	var roothash string
	var code uint32
	tees, err := n.Cli.QueryTeeInfoList()
	if err != nil {
		n.Log.Report("err", err.Error())
		return
	}
	roothashs, err := utils.Dirs(filepath.Join(n.Cli.FileDir))
	if err != nil {
		n.Log.Report("err", err.Error())
	}

	for _, v := range roothashs {
		roothash = filepath.Base(v)
		files, err := utils.DirFiles(filepath.Join(n.Cli.FileDir, roothash), 0)
		if err != nil {
			continue
		}
		for _, f := range files {
			_, err = os.Stat(filepath.Join(n.Cli.ServiceTagDir, filepath.Base(f)+".tag"))
			if err == nil {
				fmt.Println("Tag exist: ", filepath.Join(n.Cli.ServiceTagDir, filepath.Base(f)+".tag"))
				continue
			}
			for _, t := range tees {
				_ = t
				id, err := peer.Decode("12D3KooWAdyc4qPWFHsxMtXvSrm7CXNFhUmKPQdoXuKQXki69qBo")
				if err != nil {
					continue
				}
				code, err = n.Cli.CustomDataTagProtocol.TagReq(id, filepath.Base(f), "", 1024)
				if err != nil {
					fmt.Println("Tag req err:", err)
				}
				if code != 0 {
					continue
				}
				code, err = n.Cli.FileProtocol.FileReq(id, filepath.Base(f), pb.FileType_CustomData, f)
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
		err = os.MkdirAll(newDir, configs.DirMode)
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
