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

// spaceMgr task will automatically help you complete file challenges.
// Apart from human influence, it ensures that you submit your certificates in a timely manner.
// It keeps running as a subtask.
func (n *Node) fileMgr(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Log.Pnc(utils.RecoverError(err))
		}
	}()
	var txhash string
	puk, err := n.Cfg.GetPublickey()
	if err != nil {
		n.Log.Log("err", err.Error())
		os.Exit(1)
	}
	addr, err := utils.EncodeToCESSAddr(puk)
	if err != nil {
		n.Log.Log("err", err.Error())
		os.Exit(1)
	}

	for {
		roothashs, err := utils.Dirs(filepath.Join(n.Cli.Workspace(), configs.TmpDir))
		if err != nil {
			n.Log.Log("err", err.Error())
			time.Sleep(time.Minute)
			continue
		}

		for _, v := range roothashs {
			var failfile bool
			var assignedFragmentHash = make([]string, 0)
			roothash := filepath.Base(v)

			_, err = n.Cli.QueryFile(roothash)
			if err == nil {
				continue
			}

			metadata, err := n.Cli.QueryStorageOrder(roothash)
			if err != nil {
				n.Log.Log("err", err.Error())
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
			n.Log.Log("info", fmt.Sprintf("Will report [%s], files: %v", roothash, assignedFragmentHash))
			for i := 0; i < len(assignedFragmentHash); i++ {
				fstat, err := os.Stat(filepath.Join(n.TmpDir, assignedFragmentHash[i]))
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
				n.Log.Log("err", err.Error())
				continue
			}
			n.Log.Log("info", fmt.Sprintf("Report file [%s] suc: %s", roothash, txhash))
		}

		// for i := 0; i < count; i++ {
		// 	spacePath, err = generateSpace_8MB(n.SpaceDir)
		// 	if err != nil {
		// 		n.Log.Log("err", err.Error())
		// 	}

		// 	txhash, err = n.Cli.SubmitIdleFile(configs.SIZE_1MiB*8, 0, 0, 0, puk, filepath.Base(spacePath))
		// 	if err != nil {
		// 		n.Log.Log("err", fmt.Sprintf("Submit idlefile: [%s] [%s] %v", txhash, filepath.Base(spacePath), err))
		// 		continue
		// 	}
		// 	n.Log.Log("info", fmt.Sprintf("Submit idlefile: %s %s", txhash, filepath.Base(spacePath)))
	}
}
