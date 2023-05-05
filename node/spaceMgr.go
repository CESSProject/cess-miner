package node

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/sdk-go/core/rule"
	"github.com/libp2p/go-libp2p/core/peer"
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

	var count = 128 * 10
	var spacePath string
	var tagPath string
	var txhash string
	var blockheight uint32

	utils.U64ToBytes(n.PeerIndex)
	timeout := time.NewTicker(time.Duration(time.Minute * 2))
	defer timeout.Stop()
	for i := 0; i < count; i++ {
		// spacePath, err = generateSpace_8MB(n.SpaceDir)
		// if err != nil {
		// 	n.Log.Space("err", err.Error())
		// }

		_, err := n.GetAvailableTee()
		if err != nil {
			time.Sleep(rule.BlockInterval)
			continue
		}
		spacePath = ""
		tagPath = ""

		timeout.Reset(time.Duration(time.Minute * 2))
		for {
			select {
			case <-timeout.C:
				break
			case spacePath = <-n.Cli.GetIdleDataEvent():
			case tagPath = <-n.Cli.GetTagEvent():
			}

			if tagPath != "" && spacePath != "" {
				break
			}
		}

		if tagPath == "" || spacePath == "" {
			continue
		}

		txhash, err = n.Cli.SubmitIdleFile(rule.SIZE_1MiB*8, 0, 0, 0, n.Cfg.GetPublickey(), filepath.Base(spacePath))
		if err != nil {
			if txhash != "" {
				err = n.Cach.Put([]byte(fmt.Sprintf("%s%s", Cach_prefix_idle, filepath.Base(spacePath))), []byte(fmt.Sprintf("%s", txhash)))
				if err != nil {
					n.Log.Space("err", fmt.Sprintf("Record idlefile [%s] failed [%v]", filepath.Base(spacePath), err))
					continue
				}
			}
			n.Log.Space("err", fmt.Sprintf("Submit idlefile [%s] err [%s] %v", filepath.Base(spacePath), txhash, err))
			continue
		}

		blockheight, err = n.Cli.QueryBlockHeight(txhash)
		if err != nil {
			err = n.Cach.Put([]byte(fmt.Sprintf("%s%s", Cach_prefix_idle, filepath.Base(spacePath))), []byte(fmt.Sprintf("%s", txhash)))
			if err != nil {
				n.Log.Space("err", fmt.Sprintf("Record idlefile [%s] failed [%v]", filepath.Base(spacePath), err))
			}
			continue
		}

		err = n.Cach.Put([]byte(fmt.Sprintf("%s%s", Cach_prefix_idle, filepath.Base(spacePath))), []byte(fmt.Sprintf("%d", blockheight)))
		if err != nil {
			n.Log.Space("err", fmt.Sprintf("Record idlefile [%s] failed [%v]", filepath.Base(spacePath), err))
			continue
		}

		n.Log.Space("info", fmt.Sprintf("Submit idlefile [%s] suc [%s]", filepath.Base(spacePath), txhash))
	}
}

func (n *Node) GetAvailableTee() (peer.ID, error) {
	var peerid peer.ID
	var code uint32
	tees, err := n.Cli.QueryTeeInfoList()
	if err != nil {
		return peerid, err
	}

	sign, err := n.Cli.Sign(utils.U64ToBytes(n.PeerIndex))
	if err != nil {
		return peerid, err
	}

	for _, v := range tees {
		peerid, err = peer.IDFromBytes([]byte(string(v.PeerId[:])))
		if err != nil {
			continue
		}
		code, err = n.Cli.IdleDataTagProtocol.IdleReq(peerid, 8*1024*1024, 2, n.PeerIndex, sign)
		if err != nil || code != 0 {
			continue
		}
	}
	return peerid, err
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
