package node

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
)

// spaceMgr task will automatically help you complete file challenges.
// Apart from human influence, it ensures that you submit your certificates in a timely manner.
// It keeps running as a subtask.
func (n *Node) spaceMgr(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Log.Pnc(utils.RecoverError(err))
		}
	}()

	//utils.DirSize(n.Cli.Workspace())
	var err error
	var count = 128 * 10
	var spacePath string
	var txhash string
	puk, err := n.Cfg.GetPublickey()
	if err != nil {
		n.Log.Log("err", err.Error())
		os.Exit(1)
	}

	for i := 0; i < count; i++ {
		spacePath, err = generateSpace_8MB(n.SpaceDir)
		if err != nil {
			n.Log.Log("err", err.Error())
		}

		txhash, err = n.Cli.SubmitIdleFile(configs.SIZE_1MiB*8, 0, 0, 0, puk, filepath.Base(spacePath))
		if err != nil {
			n.Log.Log("err", fmt.Sprintf("Submit idlefile: [%s] [%s] %v", txhash, filepath.Base(spacePath), err))
			continue
		}
		n.Log.Log("info", fmt.Sprintf("Submit idlefile: %s %s", txhash, filepath.Base(spacePath)))
	}
}

func generateSpace_8MB(dir string) (string, error) {
	fpath := filepath.Join(dir, fmt.Sprintf("%v", time.Now().UnixNano()))
	defer os.Remove(fpath)
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0)
	if err != nil {
		return "", err
	}

	for i := uint64(0); i < 2048; i++ {
		f.WriteString(utils.RandStr(4095) + "\n")
	}
	err = f.Sync()
	if err != nil {
		os.Remove(fpath)
		return "", err
	}
	f.Close()

	hash, err := utils.CalcFileHash(fpath)
	if err != nil {
		return "", err
	}

	hashpath := filepath.Join(dir, hash)
	err = os.Rename(fpath, hashpath)
	if err != nil {
		return "", err
	}
	return hashpath, nil
}
