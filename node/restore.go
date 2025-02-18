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
	"sync"
	"time"

	"github.com/CESSProject/cess-go-sdk/chain"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/pkg/utils"
	"github.com/pkg/errors"
)

var (
	recoveryFailedFilesLock *sync.Mutex
	recoveryFailedFiles     map[string]int64
)

func init() {
	recoveryFailedFilesLock = new(sync.Mutex)
	recoveryFailedFiles = make(map[string]int64, 0)
}

func (n *Node) RestoreFiles(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	if n.GetState() != chain.MINER_STATE_POSITIVE {
		return
	}

	err := n.RestoreLocalFiles()
	if err != nil {
		n.Restore("err", err.Error())
		time.Sleep(chain.BlockInterval)
	}

	err = n.RestoreOtherFiles()
	if err != nil {
		n.Restore("err", err.Error())
		time.Sleep(chain.BlockInterval)
	}
}

func (n *Node) RestoreLocalFiles() error {
	roothashes, err := utils.Dirs(n.GetFileDir())
	if err != nil {
		n.Restore("err", fmt.Sprintf("[Dir %v] %v", n.GetFileDir(), err))
		return err
	}
	roothash := ""
	for _, v := range roothashes {
		roothash = filepath.Base(v)
		err = n.restoreFile(roothash)
		if err != nil {
			n.Restore("err", fmt.Sprintf("restoreFile: %v err: %v", n.GetFileDir(), err))
		}
	}
	return nil
}

func (n *Node) RestoreOtherFiles() error {
	restoreOrderList, err := n.QueryAllRestoralOrder(-1)
	if err != nil {
		n.Restore("err", fmt.Sprintf("[QueryRestoralOrderList] %v", err))
		return err
	}

	utils.RandSlice(restoreOrderList)
	ok := false
	blockhash := ""
	var latestBlock uint32 = 0
	for _, v := range restoreOrderList {
		recoveryFailedFilesLock.Lock()
		_, ok = recoveryFailedFiles[string(v.FragmentHash[:])]
		recoveryFailedFilesLock.Unlock()
		if ok {
			continue
		}
		blockhash = ""
		rsOrder, err := n.QueryRestoralOrder(string(v.FragmentHash[:]), -1)
		if err != nil {
			n.Restore("err", fmt.Sprintf("[QueryRestoralOrder] %v", err))
			continue
		}

		latestBlock, err = n.QueryBlockNumber("")
		if err != nil {
			n.Restore("err", fmt.Sprintf("[QueryBlockHeight] %v", err))
			continue
		}
		if latestBlock <= uint32(rsOrder.Deadline) {
			continue
		}

		_, err = n.ClaimRestoralOrder(string(v.FragmentHash[:]))
		if err != nil {
			n.Restore("err", fmt.Sprintf("[ClaimRestoralOrder] %v", err))
			continue
		}

		// recover fragment
		err = n.restoreFragment(string(v.FileHash[:]), string(v.FragmentHash[:]))
		if err != nil {
			n.Restore("err", fmt.Sprintf("[ClaimRestoralOrder] %v", err))
			recoveryFailedFilesLock.Lock()
			recoveryFailedFiles[string(v.FragmentHash[:])] = time.Now().Unix()
			recoveryFailedFilesLock.Unlock()
			continue
		}

		blockhash, err = n.RestoralOrderComplete(string(v.FragmentHash[:]))
		if err != nil {
			n.Restore("err", fmt.Sprintf("[RestoralComplete %s-%s] %v", string(v.FileHash[:]), string(v.FragmentHash[:]), err))
			return err
		}
		n.Restore("info", fmt.Sprintf("restoral complete: %v", blockhash))
	}
	return err
}

func (n *Node) restoreFile(fid string) error {
	metadata, err := n.QueryFile(fid, -1)
	if err != nil {
		time.Sleep(chain.BlockInterval)
		return err
	}
	var chainRecord = make([]string, 0)
	for i := 0; i < len(metadata.SegmentList); i++ {
		for j := 0; j < len(metadata.SegmentList[i].FragmentList); j++ {
			if sutils.CompareSlice(metadata.SegmentList[i].FragmentList[j].Miner[:], n.GetSignatureAccPulickey()) {
				chainRecord = append(chainRecord, string(metadata.SegmentList[i].FragmentList[j].Hash[:]))
			}
		}
	}
	for _, v := range chainRecord {
		fstat, err := os.Stat(filepath.Join(n.GetFileDir(), fid, v))
		if err != nil {
			if !strings.Contains(err.Error(), "no such file") {
				continue
			}
		} else {
			if fstat.Size() == chain.FragmentSize {
				continue
			}
		}

		_, ok := recoveryFailedFiles[v]
		if ok {
			continue
		}

		// recover fragment
		err = n.restoreFragment(fid, v)
		if err == nil {
			continue
		}
		recoveryFailedFilesLock.Lock()
		recoveryFailedFiles[v] = time.Now().Unix()
		recoveryFailedFilesLock.Unlock()
		n.Restore("err", fmt.Sprintf("[RestoreFragment(%s.%s)] %v", fid, v, err))
		// report lost
		_, err = n.GenerateRestoralOrder(fid, v)
		if err != nil {
			n.Restore("err", fmt.Sprintf("[GenerateRestoralOrder(%s.%s)] %v", fid, v, err))
			continue
		}
	}
	return nil
}

func (n *Node) restoreFragment(roothash, fragmentHash string) error {
	var err error
	n.Restore("info", fmt.Sprintf("[%s] To restore the fragment: %s", roothash, fragmentHash))
	_, err = os.Stat(filepath.Join(n.GetFileDir(), roothash))
	if err != nil {
		err = os.MkdirAll(filepath.Join(n.GetFileDir(), roothash), configs.FileMode)
		if err != nil {
			n.Restore("err", fmt.Sprintf("[%s.%s] Error restoring fragment: [MkdirAll] %v", roothash, fragmentHash, err))
			return err
		}
	}
	if fragmentHash == chain.ZeroFileHash_8M {
		err = os.WriteFile(filepath.Join(n.GetFileDir(), roothash, fragmentHash), make([]byte, chain.FragmentSize), os.ModePerm)
		if err != nil {
			n.Restore("err", fmt.Sprintf("[%s.%s] Error restoring fragment: %v", roothash, fragmentHash, err))
		} else {
			n.Restore("info", fmt.Sprintf("[%s.%s] Successfully restored fragment", roothash, fragmentHash))
		}
		return err
	}

	roothashes, err := utils.Dirs(n.GetFileDir())
	if err != nil {
		n.Restore("err", fmt.Sprintf("[Dir %v] %v", n.GetFileDir(), err))
		return err
	}

	for _, v := range roothashes {
		_, err = os.Stat(filepath.Join(v, fragmentHash))
		if err == nil {
			return utils.CopyFile(filepath.Join(n.GetFileDir(), roothash, fragmentHash), filepath.Join(v, fragmentHash))
		}
	}

	// data, err := GetFragmentFromOss(fragmentHash, signAcc)
	// if err == nil {
	// 	return os.WriteFile(filepath.Join(fileDir, roothash, fragmentHash), data, os.ModePerm)
	// }

	// fmeta, err := n.QueryFileMetadata(roothash)
	// if err != nil {
	// 	n.Restore("err", fmt.Sprintf("[QueryFileMetadata %v] %v", roothash, err))
	// 	return err
	// }

	// var id peer.ID
	// var miner chain.MinerInfo
	// var canRestore int
	// var recoverList = make([]string, chain.DataShards+chain.ParShards)
	// for _, segment := range fmeta.SegmentList {
	// 	for k, v := range segment.FragmentList {
	// 		if !sutils.CompareSlice(v.Miner[:], n.GetSignatureAccPulickey()) {
	// 			continue
	// 		}
	// 		if string(v.Hash[:]) == fragmentHash {
	// 			recoverList[k] = ""
	// 			continue
	// 		}
	// 		_, err = os.Stat(filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:])))
	// 		if err == nil {
	// 			n.Restore("info", fmt.Sprintf("[%s] found a fragment: %s", roothash, string(v.Hash[:])))
	// 			recoverList[k] = filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:]))
	// 			canRestore++
	// 			if canRestore >= int(len(segment.FragmentList)*2/3) {
	// 				break
	// 			}
	// 			continue
	// 		}
	// 		miner, err = n.QueryStorageMiner(v.Miner[:])
	// 		if err != nil {
	// 			n.Restore("err", fmt.Sprintf("[%s] QueryStorageMiner err: %v", roothash, err))
	// 			continue
	// 		}
	// 		id, err = peer.Decode(base58.Encode([]byte(string(miner.PeerId[:]))))
	// 		if err != nil {
	// 			n.Restore("err", fmt.Sprintf("[%s] peer Decode err: %v", roothash, err))
	// 			continue
	// 		}
	// 		addr, err := n.GetPeer(id.String())
	// 		if err != nil {
	// 			n.Restore("err", fmt.Sprintf("[%s] not found peer: %v", roothash, id.String()))
	// 			continue
	// 		}
	// 		err = n.Connect(context.Background(), addr)
	// 		if err != nil {
	// 			n.Restore("err", fmt.Sprintf("[%s] Connect peer err: %v", roothash, err))
	// 			continue
	// 		}
	// 		n.Restore("info", fmt.Sprintf("[%s] will read file from %s: %s", id.String(), roothash, string(v.Hash[:])))
	// 		err = n.ReadFileAction(id, roothash, string(v.Hash[:]), filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:])), chain.FragmentSize)
	// 		if err != nil {
	// 			err = os.Remove(filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:])))
	// 			if err == nil {
	// 				n.Del("info", filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:])))
	// 			}
	// 			n.Restore("err", fmt.Sprintf("[ReadFileAction] %v", err))
	// 			continue
	// 		}
	// 		n.Restore("info", fmt.Sprintf("[%s] found a fragment: %s", roothash, string(v.Hash[:])))
	// 		recoverList[k] = filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:]))
	// 		canRestore++
	// 		if canRestore >= int(len(segment.FragmentList)*2/3) {
	// 			break
	// 		}
	// 	}
	// 	n.Restore("info", fmt.Sprintf("all found frgments: %v", recoverList))
	// 	segmentpath := filepath.Join(n.GetDirs().FileDir, roothash, string(segment.Hash[:]))
	// 	if canRestore >= int(len(segment.FragmentList)*2/3) {
	// 		err = erasure.RSRestore(segmentpath, recoverList)
	// 		if err != nil {
	// 			os.Remove(segmentpath)
	// 			n.Del("info", segmentpath)
	// 			return err
	// 		}
	// 		_, err = erasure.ReedSolomon(segmentpath)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		_, err = os.Stat(filepath.Join(n.GetDirs().FileDir, roothash, fragmentHash))
	// 		if err != nil {
	// 			return errors.New("recpvery failed")
	// 		}
	// 		n.Restore("info", fmt.Sprintf("[%s] restore fragment suc: %s", roothash, fragmentHash))
	// 		os.Remove(segmentpath)
	// 		n.Del("info", segmentpath)
	// 	} else {
	// 		n.Restore("err", fmt.Sprintf("[%s] There are not enough fragments to recover the segment %s", roothash, string(segment.Hash[:])))
	// 		return errors.New("recpvery failed")
	// 	}
	// }
	return nil
}

func (n *Node) calcFragmentTag(fid, fragment string) error {
	buf, err := os.ReadFile(fragment)
	if err != nil {
		return err
	}
	if len(buf) != chain.FragmentSize {
		return errors.New("invalid fragment size")
	}
	fragmentHash := filepath.Base(fragment)

	genTag, teePubkey, err := n.requestTeeTag(fid, fragment, nil, nil)
	if err != nil {
		return err
	}

	if len(genTag.USig) != chain.TeeSignatureLen {
		return fmt.Errorf("invalid USig length: %d", len(genTag.USig))
	}

	if len(genTag.Signature) != chain.TeeSigLen {
		return fmt.Errorf("invalid genTag.Signature length: %d", len(genTag.Signature))
	}

	index := getTagsNumber(filepath.Join(n.GetFileDir(), fid))

	var tfile = &TagfileType{
		Tag:          genTag.Tag,
		USig:         genTag.USig,
		Signature:    genTag.Signature,
		FragmentName: []byte(fragmentHash),
		TeeAccountId: []byte(teePubkey),
		Index:        uint16(index + 1),
	}
	buf, err = json.Marshal(tfile)
	if err != nil {
		return fmt.Errorf("json.Marshal: %v", err)
	}
	// ok, err := n.GetPodr2Key().VerifyAttest(genTag.Tag.T.Name, genTag.Tag.T.U, genTag.Tag.PhiHash, genTag.Tag.Attest, "")
	// if err != nil {
	// 	n.Restore("err", fmt.Sprintf("[VerifyAttest] err: %s", err))
	// 	continue
	// }
	// if !ok {
	// 	n.Restore("err", "VerifyAttest is false")
	// 	continue
	// }
	err = sutils.WriteBufToFile(buf, fmt.Sprintf("%s.tag", fragment))
	if err != nil {
		return fmt.Errorf("WriteBufToFile: %v", err)
	}
	n.Restore("info", fmt.Sprintf("Calc a service tag: %s", fmt.Sprintf("%s.tag", fragment)))
	return nil
}
