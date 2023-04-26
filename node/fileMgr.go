package node

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/sdk-go/core/chain"
	"github.com/CESSProject/sdk-go/core/rule"
)

// fileMgr
func (n *Node) fileMgr(ch chan<- bool) {
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

	for {
		roothashs, err := utils.Dirs(filepath.Join(n.Cli.Workspace(), rule.TempDir))
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
							err = RenameDir(filepath.Join(n.Cli.Workspace(), rule.TempDir, roothash), filepath.Join(n.Cli.Workspace(), rule.FileDir, roothash))
							if err != nil {
								n.Log.Report("err", err.Error())
								continue
							}
							n.Cach.Delete([]byte(Cach_prefix_report + roothash))
							n.Cach.Put([]byte(Cach_prefix_metadata+roothash), nil)
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
						err = RenameDir(filepath.Join(n.Cli.Workspace(), rule.TempDir, roothash), filepath.Join(n.Cli.Workspace(), rule.FileDir, roothash))
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

			for i := 0; i < len(assignedFragmentHash); i++ {
				fstat, err := os.Stat(filepath.Join(n.Cli.Workspace(), rule.TempDir, roothash, assignedFragmentHash[i]))
				if err != nil || fstat.Size() != rule.FragmentSize {
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

		roothashs, err = utils.Dirs(filepath.Join(n.Cli.Workspace(), rule.FileDir))
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
