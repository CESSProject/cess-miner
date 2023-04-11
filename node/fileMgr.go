package node

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
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
	var txhash string
	var roothash string
	var ok bool
	var failfile bool
	puk, err := n.Cfg.GetPublickey()
	if err != nil {
		n.Log.Report("err", err.Error())
		os.Exit(1)
	}
	addr, err := utils.EncodeToCESSAddr(puk)
	if err != nil {
		n.Log.Report("err", err.Error())
		os.Exit(1)
	}

	for {
		roothashs, err := utils.Dirs(filepath.Join(n.Cli.Workspace(), configs.TmpDir))
		if err != nil {
			n.Log.Report("err", err.Error())
			time.Sleep(time.Minute)
			continue
		}

		for _, v := range roothashs {
			failfile = false
			var assignedFragmentHash = make([]string, 0)
			roothash = filepath.Base(v)
			ok, _ = n.Cach.Has([]byte(Cach_prefix_report + roothash))
			if ok {
				continue
			}

			_, err = n.Cli.QueryFile(roothash)
			if err == nil {
				continue
			}

			n.Log.Report("info", fmt.Sprintf("Will report %s", roothash))

			metadata, err := n.Cli.QueryStorageOrder(roothash)
			if err != nil {
				n.Log.Report("err", err.Error())
				continue
			}

			for i := 0; i < len(metadata.AssignedMiner); i++ {
				assignedAddr, _ := utils.EncodeToCESSAddr(metadata.AssignedMiner[i].Account[:])
				if addr == assignedAddr {
					for j := 0; j < len(metadata.AssignedMiner[i].Hash); j++ {
						assignedFragmentHash = append(assignedFragmentHash, string(metadata.AssignedMiner[i].Hash[j][:]))
					}
				}
			}
			n.Log.Report("info", fmt.Sprintf("Query [%s], files: %v", roothash, assignedFragmentHash))
			for i := 0; i < len(assignedFragmentHash); i++ {
				fstat, err := os.Stat(filepath.Join(n.Cli.Workspace(), configs.TmpDir, roothash, assignedFragmentHash[i]))
				if err != nil || fstat.Size() != rule.FragmentSize {
					failfile = true
					break
				}
			}
			if failfile {
				continue
			}

			txhash, _, err = n.Cli.ReportFile([]string{roothash})
			if err != nil {
				n.Log.Report("err", err.Error())
				continue
			}
			n.Log.Report("info", fmt.Sprintf("Report file [%s] suc: %s", roothash, txhash))
			n.Cach.Put([]byte(Cach_prefix_report+roothash), nil)
		}
		time.Sleep(configs.BlockInterval)
	}
}
