/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/erasure"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/cess-go-sdk/core/sdk"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/p2p-go/core"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58"
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

func RestoreFiles(cli sdk.SDK, cace cache.Cache, l logger.Logger, fileDir string, ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			l.Pnc(utils.RecoverError(err))
		}
	}()

	err := RestoreLocalFiles(cli, l, cace, fileDir)
	if err != nil {
		l.Restore("err", err.Error())
		time.Sleep(pattern.BlockInterval)
	}

	err = RestoreOtherFiles(cli, l, fileDir)
	if err != nil {
		l.Restore("err", err.Error())
		time.Sleep(pattern.BlockInterval)
	}
}

func RestoreLocalFiles(cli sdk.SDK, l logger.Logger, cace cache.Cache, fileDir string) error {
	roothashes, err := utils.Dirs(fileDir)
	if err != nil {
		l.Restore("err", fmt.Sprintf("[Dir %v] %v", fileDir, err))
		return err
	}
	roothash := ""
	for _, v := range roothashes {
		roothash = filepath.Base(v)
		err = restoreFile(cli, l, fileDir, roothash)
		if err != nil {
			l.Restore("err", fmt.Sprintf("restoreFile: %v", fileDir, err))
		}
	}
	return nil
}

func RestoreOtherFiles(cli sdk.SDK, l logger.Logger, fileDir string) error {
	restoreOrderList, err := cli.QueryRestoralOrderList()
	if err != nil {
		l.Restore("err", fmt.Sprintf("[QueryRestoralOrderList] %v", err))
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
		rsOrder, err := cli.QueryRestoralOrder(string(v.FragmentHash[:]))
		if err != nil {
			l.Restore("err", fmt.Sprintf("[QueryRestoralOrder] %v", err))
			continue
		}

		latestBlock, err = cli.QueryBlockHeight("")
		if err != nil {
			l.Restore("err", fmt.Sprintf("[QueryBlockHeight] %v", err))
			continue
		}
		if latestBlock <= uint32(rsOrder.Deadline) {
			continue
		}

		_, err = cli.ClaimRestoralOrder(string(v.FragmentHash[:]))
		if err != nil {
			l.Restore("err", fmt.Sprintf("[ClaimRestoralOrder] %v", err))
			continue
		}

		// recover fragment
		err = restoreFragment(cli.GetSignatureAcc(), l, string(v.FileHash[:]), string(v.FragmentHash[:]), fileDir)
		if err != nil {
			l.Restore("err", fmt.Sprintf("[ClaimRestoralOrder] %v", err))
			recoveryFailedFilesLock.Lock()
			recoveryFailedFiles[string(v.FragmentHash[:])] = time.Now().Unix()
			recoveryFailedFilesLock.Unlock()
			continue
		}

		blockhash, err = cli.RestoralComplete(string(v.FragmentHash[:]))
		if err != nil {
			l.Restore("err", fmt.Sprintf("[RestoralComplete %s-%s] %v", string(v.FileHash[:]), string(v.FragmentHash[:]), err))
			return err
		}
		l.Restore("info", fmt.Sprintf("restoral complete: %v", blockhash))
	}
	return err
}

func restoreFile(cli sdk.SDK, l logger.Logger, fileDir string, fid string) error {
	metadata, err := cli.QueryFileMetadata(fid)
	if err != nil {
		time.Sleep(pattern.BlockInterval)
		return err
	}
	var chainRecord = make([]string, 0)
	for i := 0; i < len(metadata.SegmentList); i++ {
		for j := 0; j < len(metadata.SegmentList[i].FragmentList); j++ {
			if sutils.CompareSlice(metadata.SegmentList[i].FragmentList[j].Miner[:], cli.GetSignatureAccPulickey()) {
				chainRecord = append(chainRecord, string(metadata.SegmentList[i].FragmentList[j].Hash[:]))
			}
		}
	}
	for _, v := range chainRecord {
		fstat, err := os.Stat(filepath.Join(fileDir, fid, v))
		if err != nil {
			if !strings.Contains(err.Error(), "no such file") {
				continue
			}
		} else {
			if fstat.Size() == pattern.FragmentSize {
				continue
			}
		}

		_, ok := recoveryFailedFiles[v]
		if ok {
			continue
		}

		// recover fragment
		err = restoreFragment(cli.GetSignatureAcc(), l, fid, v, fileDir)
		if err == nil {
			continue
		}
		recoveryFailedFilesLock.Lock()
		recoveryFailedFiles[v] = time.Now().Unix()
		recoveryFailedFilesLock.Unlock()
		l.Restore("err", fmt.Sprintf("[RestoreFragment(%s.%s)] %v", fid, v, err))
		// report lost
		_, err = cli.GenerateRestoralOrder(fid, v)
		if err != nil {
			l.Restore("err", fmt.Sprintf("[GenerateRestoralOrder(%s.%s)] %v", fid, v, err))
			continue
		}
	}
	return nil
}

func restoreFragment(signAcc string, l logger.Logger, roothash, fragmentHash, fileDir string) error {
	var err error
	l.Restore("info", fmt.Sprintf("[%s] To restore the fragment: %s", roothash, fragmentHash))
	_, err = os.Stat(filepath.Join(fileDir, roothash))
	if err != nil {
		err = os.MkdirAll(filepath.Join(fileDir, roothash), configs.FileMode)
		if err != nil {
			l.Restore("err", fmt.Sprintf("[%s.%s] Error restoring fragment: [MkdirAll] %v", roothash, fragmentHash, err))
			return err
		}
	}
	if fragmentHash == core.ZeroFileHash_8M {
		err = os.WriteFile(filepath.Join(fileDir, roothash, fragmentHash), make([]byte, pattern.FragmentSize), os.ModePerm)
		if err != nil {
			l.Restore("err", fmt.Sprintf("[%s.%s] Error restoring fragment: %v", roothash, fragmentHash, err))
		} else {
			l.Restore("info", fmt.Sprintf("[%s.%s] Successfully restored fragment", roothash, fragmentHash))
		}
		return err
	}

	roothashes, err := utils.Dirs(fileDir)
	if err != nil {
		l.Restore("err", fmt.Sprintf("[Dir %v] %v", fileDir, err))
		return err
	}

	for _, v := range roothashes {
		_, err = os.Stat(filepath.Join(v, fragmentHash))
		if err == nil {
			return utils.CopyFile(filepath.Join(fileDir, roothash, fragmentHash), filepath.Join(v, fragmentHash))
		}
	}

	data, err := GetFragmentFromOss(fragmentHash, signAcc)
	if err == nil {
		return os.WriteFile(filepath.Join(fileDir, roothash, fragmentHash), data, os.ModePerm)
	}

	// fmeta, err := n.QueryFileMetadata(roothash)
	// if err != nil {
	// 	n.Restore("err", fmt.Sprintf("[QueryFileMetadata %v] %v", roothash, err))
	// 	return err
	// }

	// var id peer.ID
	// var miner pattern.MinerInfo
	// var canRestore int
	// var recoverList = make([]string, pattern.DataShards+pattern.ParShards)
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
	// 		err = n.ReadFileAction(id, roothash, string(v.Hash[:]), filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:])), pattern.FragmentSize)
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

func (n *Node) claimRestoreOrder() error {
	var err error
	val, _ := n.QueryPrefixKeyList(Cach_prefix_recovery)
	for _, v := range val {
		restoreOrder, err := n.QueryRestoralOrder(v)
		if err != nil {
			if err.Error() == pattern.ERR_Empty {
				n.Delete([]byte(Cach_prefix_recovery + v))
				continue
			}
			continue
		}

		b, err := n.Get([]byte(Cach_prefix_recovery + v))
		if err != nil {
			n.Restore("err", fmt.Sprintf("[Get %s] %v", v, err))
			n.Delete([]byte(Cach_prefix_recovery + v))
			continue
		}
		err = n.restoreAFragment(string(b), v, filepath.Join(n.GetDirs().FileDir, string(b), v))
		if err != nil {
			n.Restore("err", fmt.Sprintf("[restoreAFragment %s-%s] %v", string(b), v, err))
			continue
		}

		if !sutils.CompareSlice(restoreOrder.Miner[:], n.GetSignatureAccPulickey()) {
			n.Delete([]byte(Cach_prefix_recovery + v))
			continue
		}

		txhash, err := n.RestoralComplete(v)
		if err != nil {
			n.Restore("err", fmt.Sprintf("[RestoralComplete %s-%s] %v", string(b), v, err))
			continue
		}
		// err = n.calcFragmentTag(string(b), filepath.Join(n.GetDirs().FileDir, string(b), v))
		// if err != nil {
		// 	n.Restore("err", fmt.Sprintf("[calcFragmentTag %s-%s] %v", string(b), v, err))
		// }
		n.Restore("info", fmt.Sprintf("[RestoralComplete %s-%s] %s", string(b), v, txhash))
		n.Delete([]byte(Cach_prefix_recovery + v))
	}

	time.Sleep(time.Second * time.Duration(rand.Intn(200)+3))

	restoreOrderList, err := n.QueryRestoralOrderList()
	if err != nil {
		n.Restore("err", fmt.Sprintf("[QueryRestoralOrderList] %v", err))
		return err
	}
	// blockHeight, err := n.QueryBlockHeight("")
	// if err != nil {
	// 	n.Restore("err", fmt.Sprintf("[QueryBlockHeight] %v", err))
	// 	return err
	// }
	utils.RandSlice(restoreOrderList)
	for _, v := range restoreOrderList {
		time.Sleep(time.Second * time.Duration(utils.RandomInRange(1, 100)*6))
		rsOrder, err := n.QueryRestoralOrder(string(v.FragmentHash[:]))
		if err != nil {
			n.Restore("err", fmt.Sprintf("[QueryRestoralOrder] %v", err))
			continue
		}
		blockHeight, err := n.QueryBlockHeight("")
		if err != nil {
			n.Restore("err", fmt.Sprintf("[QueryBlockHeight] %v", err))
			continue
		}
		if (blockHeight + 30) <= uint32(rsOrder.Deadline) {
			continue
		}
		_, err = n.ClaimRestoralOrder(string(v.FragmentHash[:]))
		if err != nil {
			n.Restore("err", fmt.Sprintf("[ClaimRestoralOrder] %v", err))
			continue
		}
		n.Put([]byte(Cach_prefix_recovery+string(v.FragmentHash[:])), []byte(string(v.FileHash[:])))
		break
	}

	return nil
}

func (n *Node) restoreAFragment(roothash, framentHash, recoveryPath string) error {
	var err error
	var miner pattern.MinerInfo
	n.Restore("info", fmt.Sprintf("[%s] To restore the fragment: %s", roothash, framentHash))
	n.Restore("info", fmt.Sprintf("[%s] Restore path: %s", roothash, recoveryPath))
	_, err = os.Stat(filepath.Join(n.GetDirs().FileDir, roothash))
	if err != nil {
		err = os.MkdirAll(filepath.Join(n.GetDirs().FileDir, roothash), configs.FileMode)
		if err != nil {
			n.Restore("err", fmt.Sprintf("[%s.%s] Error restoring fragment: [MkdirAll] %v", roothash, framentHash, err))
			return err
		}
	}

	if framentHash == core.ZeroFileHash_8M {
		err = os.WriteFile(recoveryPath, make([]byte, pattern.FragmentSize), os.ModePerm)
		if err != nil {
			n.Restore("err", fmt.Sprintf("[%s.%s] Error restoring fragment: %v", roothash, framentHash, err))
		} else {
			n.Restore("info", fmt.Sprintf("[%s.%s] Successfully restored fragment", roothash, framentHash))
		}
		return err
	}

	roothashes, _ := utils.Dirs(n.GetDirs().FileDir)
	for _, v := range roothashes {
		_, err = os.Stat(filepath.Join(v, framentHash))
		if err == nil {
			n.Restore("info", fmt.Sprintf("[%s] found: %s", roothash, filepath.Join(v, framentHash)))
			err = utils.CopyFile(recoveryPath, filepath.Join(v, framentHash))
			if err == nil {
				n.Delete([]byte(Cach_prefix_MyLost + framentHash))
				n.Delete([]byte(Cach_prefix_recovery + framentHash))
				n.Restore("info", fmt.Sprintf("[%s] Restore the fragment: %s", roothash, framentHash))
				return nil
			}
		}
	}

	data, err := GetFragmentFromOss(framentHash, "")
	if err == nil && len(data) == pattern.FragmentSize {
		err = os.WriteFile(recoveryPath, data, os.ModePerm)
		if err == nil {
			return nil
		}
	}

	var canRestore int
	var dstSegement pattern.SegmentInfo
	fmeta, err := n.QueryFileMetadata(roothash)
	if err != nil {
		return err
	}
	for _, segement := range fmeta.SegmentList {
		for _, v := range segement.FragmentList {
			if string(v.Hash[:]) == framentHash {
				dstSegement = segement
				break
			}
		}
		if dstSegement.FragmentList != nil {
			break
		}
	}
	var recoverList = make([]string, len(dstSegement.FragmentList))
	n.Restore("info", fmt.Sprintf("[%s] locate to segment: %s", roothash, string(dstSegement.Hash[:])))
	n.Restore("info", fmt.Sprintf("[%s] segmen contains %d fragments:", roothash, len(dstSegement.FragmentList)))
	for k, v := range dstSegement.FragmentList {
		// if string(v.Hash[:]) == framentHash {
		// 	recoverList[k] = ""
		// 	continue
		// }
		_, err = os.Stat(filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:])))
		if err == nil {
			n.Restore("info", fmt.Sprintf("[%s] found a fragment: %s", roothash, string(v.Hash[:])))
			recoverList[k] = filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:]))
			canRestore++
			if canRestore >= int(len(dstSegement.FragmentList)*2/3) {
				break
			}
			continue
		}
		minerAcc, _ := sutils.EncodePublicKeyAsCessAccount(v.Miner[:])
		miner, err = n.QueryStorageMiner(v.Miner[:])
		if err != nil {
			n.Restore("err", fmt.Sprintf("[QueryStorageMiner %s]: %v", minerAcc, err))
			continue
		}

		peerid := base58.Encode([]byte(string(miner.PeerId[:])))

		// addr, err := n.GetPeer(peerid)
		// if err != nil {
		// 	n.Restore("err", fmt.Sprintf("Not found miner: %s, %s", minerAcc, peerid))
		// 	continue
		// }
		addr := peer.AddrInfo{}
		err = n.Connect(context.Background(), addr)
		if err != nil {
			n.Restore("err", fmt.Sprintf("Connect to miner failed: %s, %s, err: %v", minerAcc, peerid, err))
			continue
		}

		n.Restore("info", fmt.Sprintf("[%s] will read file from %s: %s", peerid, roothash, string(v.Hash[:])))
		err = n.ReadFileAction(addr.ID, roothash, string(v.Hash[:]), filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:])), pattern.FragmentSize)
		if err != nil {
			os.Remove(filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:])))
			n.Del("info", filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:])))
			n.Restore("err", fmt.Sprintf("[ReadFileAction] %v", err))
			continue
		}
		n.Restore("info", fmt.Sprintf("[%s] found a fragment: %s", roothash, string(v.Hash[:])))
		recoverList[k] = filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:]))
		canRestore++
		if canRestore >= int(len(dstSegement.FragmentList)*2/3) {
			break
		}
	}
	n.Restore("info", fmt.Sprintf("all found frgments: %v", recoverList))
	segmentpath := filepath.Join(n.GetDirs().FileDir, roothash, string(dstSegement.Hash[:]))
	if canRestore >= int(len(dstSegement.FragmentList)*2/3) {
		err = erasure.RSRestore(segmentpath, recoverList)
		if err != nil {
			os.Remove(segmentpath)
			n.Del("info", segmentpath)
			return err
		}
		_, err = erasure.ReedSolomon(segmentpath)
		if err != nil {
			return err
		}
		_, err = os.Stat(filepath.Join(n.GetDirs().FileDir, roothash, framentHash))
		if err != nil {
			return errors.New("recpvery failed")
		}
		n.Restore("info", fmt.Sprintf("[%s] restore fragment suc: %s", roothash, framentHash))
		n.Delete([]byte(Cach_prefix_MyLost + framentHash))
		n.Delete([]byte(Cach_prefix_recovery + framentHash))
		os.Remove(segmentpath)
		n.Del("info", segmentpath)
	} else {
		n.Restore("err", fmt.Sprintf("[%s] There are not enough fragments to recover the segment %s", roothash, string(dstSegement.Hash[:])))
		return errors.New("recpvery failed")
	}

	return nil
}

func (n *Node) claimNoExitOrder() error {
	var roothash string
	var fmeta pattern.FileMetadata
	var miner string
	var txhash string
	var restoralOrder pattern.RestoralOrderInfo
	var blockHeight uint32
	var roothashs = make(map[string]struct{}, 0)
	targetMiner, err := n.QueryPrefixKeyList(Cach_prefix_TargetMiner)
	if err == nil && len(targetMiner) > 0 {
		roothashList, err := utils.Dirs(n.GetDirs().FileDir)
		if err != nil {
			n.Restore("err", fmt.Sprintf("[Dirs] %v", err))
		}
		for _, v := range roothashList {
			roothashs[filepath.Base(v)] = struct{}{}
		}
		filelist, err := n.QueryPrefixKeyList(Cach_prefix_File)
		if err != nil {
			n.Restore("err", fmt.Sprintf("[QueryPrefixKeyList] %v", err))
		}
		for _, v := range filelist {
			roothashs[v] = struct{}{}
		}
		for v := range roothashs {
			roothash = v
			n.Restore("info", fmt.Sprintf("check file: %s", roothash))
			fmeta, err = n.QueryFileMetadata(roothash)
			if err != nil {
				n.Restore("err", fmt.Sprintf("[QueryFileMetadata] %v", err))
				continue
			}
			for _, segment := range fmeta.SegmentList {
				for _, fragment := range segment.FragmentList {
					miner, err = sutils.EncodePublicKeyAsCessAccount(fragment.Miner[:])
					if err != nil {
						n.Restore("err", fmt.Sprintf("[EncodePublicKeyAsCessAccount] %v", err))
						continue
					}
					if miner == targetMiner[0] {
						if ok, _ := n.Has([]byte(Cach_prefix_recovery + string(fragment.Hash[:]))); ok {
							continue
						}

						restoralOrder, err = n.QueryRestoralOrder(string(fragment.Hash[:]))
						if err != nil {
							if err.Error() != pattern.ERR_Empty {
								n.Restore("err", fmt.Sprintf("[QueryRestoralOrder] %v", err))
								continue
							}
						} else {
							blockHeight, err = n.QueryBlockHeight("")
							if err != nil {
								continue
							}
							if uint32(restoralOrder.Deadline) <= blockHeight {
								continue
							}
							time.Sleep(time.Second * time.Duration(rand.Intn(100)+3))
							restoralOrder, err = n.QueryRestoralOrder(string(fragment.Hash[:]))
							if err != nil {
								if err.Error() != pattern.ERR_Empty {
									n.Restore("err", fmt.Sprintf("[QueryRestoralOrder] %v", err))
									continue
								}
							} else {
								blockHeight, err = n.QueryBlockHeight("")
								if err != nil {
									continue
								}
								if uint32(restoralOrder.Deadline) <= blockHeight {
									continue
								}
							}
						}

						n.Restore("info", fmt.Sprintf("will claim restore order and fragment is: %s", string(fragment.Hash[:])))
						txhash, err = n.ClaimRestoralNoExistOrder(fragment.Miner[:], roothash, string(fragment.Hash[:]))
						if err != nil {
							n.Restore("err", fmt.Sprintf("[ClaimRestoralNoExistOrder] %v", err))
							continue
						}
						n.Restore("info", fmt.Sprintf("Claim exit miner [%s] restoral fragment [%s] order: %s", miner, string(fragment.Hash[:]), txhash))
						n.Put([]byte(Cach_prefix_recovery+string(fragment.Hash[:])), []byte(roothash))
						return nil
					}
				}
			}
		}
		n.Delete([]byte(Cach_prefix_TargetMiner + targetMiner[0]))
	}

	restoreTargetList, err := n.QueryRestoralTargetList()
	if err != nil {
		return errors.Wrapf(err, "[QueryRestoralTargetList]")
	}

	utils.RandSlice(restoreTargetList)

	for _, v := range restoreTargetList {
		minerAcc, err := sutils.EncodePublicKeyAsCessAccount(v.Miner[:])
		if err != nil {
			n.Restore("err", fmt.Sprintf("[EncodePublicKeyAsCessAccount] %v", err))
			continue
		}
		if v.ServiceSpace.CmpAbs(v.RestoredSpace.Int) < 1 {
			n.Delete([]byte(Cach_prefix_TargetMiner + minerAcc))
			continue
		}
		n.Restore("info", fmt.Sprintf("Found a exit miner: %s", minerAcc))
		n.Put([]byte(Cach_prefix_TargetMiner+minerAcc), nil)
		break
	}
	return nil
}

func calcFragmentTag(cli sdk.SDK, l logger.Logger, teeRecord *TeeRecord, ws *Workspace, fid, fragment string) error {
	buf, err := os.ReadFile(fragment)
	if err != nil {
		return err
	}
	if len(buf) != pattern.FragmentSize {
		return errors.New("invalid fragment size")
	}
	fragmentHash := filepath.Base(fragment)

	genTag, teePubkey, err := requestTeeTag(l, teeRecord, cli.GetSignatureAccPulickey(), fid, fragment, nil, nil)
	if err != nil {
		return err
	}

	if len(genTag.USig) != pattern.TeeSignatureLen {
		return fmt.Errorf("invalid USig length: %d", len(genTag.USig))
	}

	if len(genTag.Signature) != pattern.TeeSigLen {
		return fmt.Errorf("invalid genTag.Signature length: %d", len(genTag.Signature))
	}

	index := getTagsNumber(filepath.Join(ws.GetFileDir(), fid))

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
	l.Restore("info", fmt.Sprintf("Calc a service tag: %s", fmt.Sprintf("%s.tag", fragment)))
	return nil
}
