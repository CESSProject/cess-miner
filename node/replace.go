package node

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
)

// replaceMgr
func (n *Node) replaceMgr(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Log.Pnc(utils.RecoverError(err))
		}
	}()

	var err error
	var txhash string
	var count uint32
	var spacedir = filepath.Join(n.Cli.Workspace(), configs.SpaceDir)

	for {
		count, err = n.Cli.QueryPendingReplacements(n.Cfg.GetPublickey())
		if err != nil {
			n.Log.Replace("err", err.Error())
			time.Sleep(time.Minute)
			continue
		}

		if count == 0 {
			time.Sleep(time.Minute)
			continue
		}

		if count > MaxReplaceFiles {
			count = MaxReplaceFiles
		}
		files, err := SelectIdleFiles(spacedir, count)
		if err != nil {
			n.Log.Replace("err", err.Error())
			time.Sleep(time.Minute)
			continue
		}

		txhash, _, err = n.Cli.ReplaceFile(files)
		if err != nil {
			n.Log.Replace("err", err.Error())
			time.Sleep(configs.BlockInterval)
			continue
		}

		n.Log.Replace("info", fmt.Sprintf("Replace files: %v suc: [%s]", files, txhash))
		for i := 0; i < len(files); i++ {
			os.Remove(filepath.Join(spacedir, files[i]))
		}
	}
}

func SelectIdleFiles(dir string, count uint32) ([]string, error) {
	files, err := utils.DirFiles(dir, count)
	if err != nil {
		return nil, err
	}
	var result = make([]string, 0)
	for i := 0; i < len(files); i++ {
		result = append(result, filepath.Base(files[i]))
	}
	return result, nil
}
