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
	"time"

	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/sdk-go/core/client"
	"github.com/CESSProject/sdk-go/core/rule"
	"github.com/decred/base58"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
)

// spaceMgt is a subtask for managing spaces
func (n *Node) spaceMgt(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Log.Pnc(utils.RecoverError(err))
		}
	}()

	var err error
	var spacePath string
	var tagPath string
	var txhash string
	var filehash string
	var blockheight uint32
	var teepuk []byte

	n.Log.Space("info", "Start spaceMgt task")

	timeout := time.NewTimer(time.Duration(time.Minute * 2))
	defer timeout.Stop()

	teelist, err := n.Cli.Chain.QueryTeeInfoList()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(len(teelist))
	fmt.Println(teelist)

	for {
		fmt.Println("2")
		_, err = n.GetAvailableTee()
		if err != nil {
			n.Log.Space("err", err.Error())
			time.Sleep(rule.BlockInterval)
			continue
		}
		fmt.Println("3")
		spacePath = ""
		tagPath = ""

		timeout.Reset(time.Duration(time.Minute * 2))
		for {
			select {
			case <-timeout.C:
				break
			case spacePath = <-n.Cli.GetIdleDataEvent():
			case tagPath = <-n.Cli.GetIdleTagEvent():
			}

			if tagPath != "" && spacePath != "" {
				break
			}
		}

		fmt.Println("receive tag: ", tagPath)
		fmt.Println("receive idlefile: ", spacePath)

		if tagPath == "" || spacePath == "" {
			n.Log.Space("err", spacePath)
			n.Log.Space("err", tagPath)
			continue
		}

		filehash, err = utils.CalcPathSHA256(spacePath)
		if err != nil {
			n.Log.Space("err", err.Error())
			os.Remove(spacePath)
			os.Remove(tagPath)
			continue
		}

		os.Rename(spacePath, filepath.Join(n.Cli.IdleDataDir, filehash))
		os.Rename(tagPath, filepath.Join(n.Cli.IdleTagDir, filehash+".tag"))

		for k := 0; k < len(teelist); k++ {
			pid := base58.Encode([]byte(string(teelist[k].PeerId[:])))
			if pid == "12D3KooWAdyc4qPWFHsxMtXvSrm7CXNFhUmKPQdoXuKQXki69qBo" {
				teepuk = teelist[k].ControllerAccount[:]
				fmt.Println("Found tee accpunt id: ", teepuk)
			}
		}

		var idlefile client.IdleFileMeta
		idlefile.BlockNum = 1024
		idlefile.BlockSize = 0
		idlefile.Hash = filehash
		idlefile.ScanSize = 0
		idlefile.Size = rule.SIZE_1MiB * 8
		idlefile.MinerAcc = n.Cfg.GetPublickey()
		txhash, err = n.Cli.SubmitIdleFile(teepuk, []client.IdleFileMeta{idlefile})
		fmt.Println("txhash:", txhash)
		fmt.Println("err:", err)
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
	// var code uint32
	// tees, err := n.Cli.QueryTeeInfoList()
	// if err != nil {
	// 	return peerid, err
	// }
	// fmt.Println(len(tees))
	// fmt.Println(tees)
	sign, err := n.Cli.Sign(n.Cli.PeerId)
	if err != nil {
		return peerid, err
	}
	fmt.Println(len(sign))
	fmt.Println(sign)
	// for _, v := range tees {
	// 	peerids := base58.Encode([]byte(string(v.PeerId[:])))
	// 	log.Println("found tee: ", peerids)
	// 	n.Cli.AddMultiaddrToPearstore("/ip4/221.122.79.3/tcp/10010/p2p/12D3KooWAdyc4qPWFHsxMtXvSrm7CXNFhUmKPQdoXuKQXki69qBo", time.Hour*999)
	// 	peerids = "12D3KooWAdyc4qPWFHsxMtXvSrm7CXNFhUmKPQdoXuKQXki69qBo"
	// 	code, err = n.Cli.IdleDataTagProtocol.IdleReq(peer.ID(peerids), 8*1024*1024, 2, sign)
	// 	if err != nil || code != 0 {
	// 		continue
	// 	}
	// }
	_, err = n.Cli.AddMultiaddrToPearstore("/ip4/221.122.79.3/tcp/10010/p2p/12D3KooWAdyc4qPWFHsxMtXvSrm7CXNFhUmKPQdoXuKQXki69qBo", time.Hour*999)
	if err != nil {
		return peerid, errors.Wrapf(err, "[AddMultiaddrToPearstore]")
	}
	//peerids := "12D3KooWAdyc4qPWFHsxMtXvSrm7CXNFhUmKPQdoXuKQXki69qBo"
	id, err := peer.Decode("12D3KooWAdyc4qPWFHsxMtXvSrm7CXNFhUmKPQdoXuKQXki69qBo")
	if err != nil {
		return peerid, errors.Wrapf(err, "[Decode]")
	}
	code, err := n.Cli.IdleDataTagProtocol.IdleReq(id, 8*1024*1024, 1024, n.Cfg.GetPublickey(), sign)
	fmt.Println(code, err)
	// if err != nil || code != 0 {
	// 	return peerid, err
	// }
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