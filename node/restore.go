/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AstaFrode/go-libp2p/core/peer"
	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/erasure"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (n *Node) restoreMgt(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	chainSt := n.GetChainState()
	if !chainSt {
		return
	}

	minerSt := n.GetMinerState()
	if minerSt != pattern.MINER_STATE_POSITIVE &&
		minerSt != pattern.MINER_STATE_FROZEN {
		return
	}

	err := n.restoreFiles()
	if err != nil {
		n.Restore("err", err.Error())
		time.Sleep(pattern.BlockInterval)
	}

	err = n.claimNoExitOrder()
	if err != nil {
		n.Restore("err", err.Error())
		time.Sleep(pattern.BlockInterval)
	}

	err = n.claimRestoreOrder()
	if err != nil {
		n.Restore("err", err.Error())
		time.Sleep(pattern.BlockInterval)
	}
}

func (n *Node) restoreFiles() error {
	var (
		ok           bool
		recover      bool
		err          error
		roothash     string
		fragmentHash []byte
	)

	roothashes, err := utils.Dirs(n.GetDirs().FileDir)
	if err != nil {
		n.Restore("err", fmt.Sprintf("[Dir %v] %v", n.GetDirs().FileDir, err))
		roothashes, err = n.QueryPrefixKeyList(Cach_prefix_File)
		if err != nil {
			return errors.Wrapf(err, "[QueryPrefixKeyList]")
		}
	}

	for _, v := range roothashes {
		roothash = filepath.Base(v)
		recover = false
		ok, err = n.Has([]byte(Cach_prefix_File + roothash))
		if err != nil {
			n.Restore("err", fmt.Sprintf("[Cache.Has(%v)] %v", Cach_prefix_File+roothash, err))
			continue
		}
		if !ok {
			continue
		}
		fragmentHash, err = n.Get([]byte(Cach_prefix_File + roothash))
		if err != nil {
			n.Restore("err", fmt.Sprintf("[Cache.Get(%v)] %v", Cach_prefix_File+roothash, err))
			continue
		}

		fstat, err := os.Stat(filepath.Join(n.GetDirs().FileDir, roothash, string(fragmentHash)))
		if err != nil {
			if strings.Contains(err.Error(), configs.Err_file_not_fount) {
				recover = true
			} else {
				n.Restore("err", fmt.Sprintf("[os.Stat(%s.%s)] %v", roothash, string(fragmentHash), err))
				continue
			}
		}
		if fstat.Size() != pattern.FragmentSize {
			recover = true
		}
		if recover {
			// Try to recover yourself
			err = n.restoreFragment(roothashes, roothash, string(fragmentHash))
			if err != nil {
				n.Restore("err", fmt.Sprintf("[RestoreFragment(%s.%s)] %v", roothash, string(fragmentHash), err))
			}
		}
	}
	return nil
}

func (n *Node) restoreFragment(roothashes []string, roothash, fragmentHash string) error {
	var err error
	n.Restore("info", fmt.Sprintf("[%s] To restore the fragment: %s", roothash, fragmentHash))
	_, err = os.Stat(filepath.Join(n.GetDirs().FileDir, roothash))
	if err != nil {
		err = os.MkdirAll(filepath.Join(n.GetDirs().FileDir, roothash), pattern.DirMode)
		if err != nil {
			n.Restore("err", fmt.Sprintf("[%s.%s] Error restoring fragment: [MkdirAll] %v", roothash, fragmentHash, err))
			return err
		}
	}
	if fragmentHash == core.ZeroFileHash_16M {
		err = os.WriteFile(filepath.Join(n.GetDirs().FileDir, roothash, fragmentHash), make([]byte, pattern.FragmentSize), os.ModePerm)
		if err != nil {
			n.Restore("err", fmt.Sprintf("[%s.%s] Error restoring fragment: %v", roothash, fragmentHash, err))
		} else {
			n.Restore("info", fmt.Sprintf("[%s.%s] Successfully restored fragment", roothash, fragmentHash))
		}
		return err
	}

	for _, v := range roothashes {
		_, err = os.Stat(filepath.Join(v, fragmentHash))
		if err == nil {
			err = utils.CopyFile(filepath.Join(n.GetDirs().FileDir, roothash, fragmentHash), filepath.Join(v, fragmentHash))
			if err == nil {
				return nil
			}
		}
	}

	data, err := n.GetFragmentFromOss(fragmentHash)
	if err == nil {
		err = os.WriteFile(filepath.Join(n.GetDirs().FileDir, roothash, fragmentHash), data, os.ModePerm)
		if err == nil {
			return nil
		}
	}

	fmeta, err := n.QueryFileMetadata(roothash)
	if err != nil {
		n.Restore("err", fmt.Sprintf("[QueryFileMetadata %v] %v", roothash, err))
		return err
	}

	var id peer.ID
	var miner pattern.MinerInfo
	var canRestore int
	var recoverList = make([]string, pattern.DataShards+pattern.ParShards)
	for _, segment := range fmeta.SegmentList {
		for k, v := range segment.FragmentList {
			if !sutils.CompareSlice(v.Miner[:], n.GetSignaturePublickey()) {
				continue
			}
			if string(v.Hash[:]) == fragmentHash {
				recoverList[k] = ""
				continue
			}
			_, err = os.Stat(filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:])))
			if err == nil {
				n.Restore("info", fmt.Sprintf("[%s] found a fragment: %s", roothash, string(v.Hash[:])))
				recoverList[k] = filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:]))
				canRestore++
				if canRestore >= int(len(segment.FragmentList)*2/3) {
					break
				}
				continue
			}
			miner, err = n.QueryStorageMiner(v.Miner[:])
			if err != nil {
				n.Restore("err", fmt.Sprintf("[%s] QueryStorageMiner err: %v", roothash, err))
				continue
			}
			id, err = peer.Decode(base58.Encode([]byte(string(miner.PeerId[:]))))
			if err != nil {
				n.Restore("err", fmt.Sprintf("[%s] peer Decode err: %v", roothash, err))
				continue
			}
			addr, err := n.GetPeer(id.Pretty())
			if err != nil {
				n.Restore("err", fmt.Sprintf("[%s] not found peer: %v", roothash, id.Pretty()))
				continue
			}
			err = n.Connect(n.GetCtxQueryFromCtxCancel(), addr)
			if err != nil {
				n.Restore("err", fmt.Sprintf("[%s] Connect peer err: %v", roothash, err))
				continue
			}
			n.Restore("info", fmt.Sprintf("[%s] will read file from %s: %s", id.Pretty(), roothash, string(v.Hash[:])))
			err = n.ReadFileAction(id, roothash, string(v.Hash[:]), filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:])), pattern.FragmentSize)
			if err != nil {
				os.Remove(filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:])))
				n.Restore("err", fmt.Sprintf("[ReadFileAction] %v", err))
				continue
			}
			n.Restore("info", fmt.Sprintf("[%s] found a fragment: %s", roothash, string(v.Hash[:])))
			recoverList[k] = filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:]))
			canRestore++
			if canRestore >= int(len(segment.FragmentList)*2/3) {
				break
			}
		}
		n.Restore("info", fmt.Sprintf("all found frgments: %v", recoverList))
		segmentpath := filepath.Join(n.GetDirs().FileDir, roothash, string(segment.Hash[:]))
		if canRestore >= int(len(segment.FragmentList)*2/3) {
			err = erasure.RSRestore(segmentpath, recoverList)
			if err != nil {
				os.Remove(segmentpath)
				return err
			}
			_, err = erasure.ReedSolomon(segmentpath)
			if err != nil {
				return err
			}
			_, err = os.Stat(filepath.Join(n.GetDirs().FileDir, roothash, fragmentHash))
			if err != nil {
				return errors.New("recpvery failed")
			}
			n.Restore("info", fmt.Sprintf("[%s] restore fragment suc: %s", roothash, fragmentHash))
			os.Remove(segmentpath)
		} else {
			n.Restore("err", fmt.Sprintf("[%s] There are not enough fragments to recover the segment %s", roothash, string(segment.Hash[:])))
			return errors.New("recpvery failed")
		}
	}
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

		if !sutils.CompareSlice(restoreOrder.Miner[:], n.GetSignaturePublickey()) {
			n.Delete([]byte(Cach_prefix_recovery + v))
			continue
		}

		txhash, err := n.RestoralComplete(v)
		if err != nil {
			n.Restore("err", fmt.Sprintf("[RestoralComplete %s-%s] %v", string(b), v, err))
			continue
		}
		err = n.calcFragmentTag(string(b), filepath.Join(n.GetDirs().FileDir, string(b), v))
		if err != nil {
			n.Restore("err", fmt.Sprintf("[calcFragmentTag %s-%s] %v", string(b), v, err))
		}
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
		err = os.MkdirAll(filepath.Join(n.GetDirs().FileDir, roothash), pattern.DirMode)
		if err != nil {
			n.Restore("err", fmt.Sprintf("[%s.%s] Error restoring fragment: [MkdirAll] %v", roothash, framentHash, err))
			return err
		}
	}

	if framentHash == core.ZeroFileHash_16M {
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

	data, err := n.GetFragmentFromOss(framentHash)
	if err == nil {
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

		addr, err := n.GetPeer(peerid)
		if err != nil {
			n.Restore("err", fmt.Sprintf("Not found miner: %s, %s", minerAcc, peerid))
			continue
		}

		err = n.Connect(n.GetCtxQueryFromCtxCancel(), addr)
		if err != nil {
			n.Restore("err", fmt.Sprintf("Connect to miner failed: %s, %s, err: %v", minerAcc, peerid, err))
			continue
		}

		n.Restore("info", fmt.Sprintf("[%s] will read file from %s: %s", peerid, roothash, string(v.Hash[:])))
		err = n.ReadFileAction(addr.ID, roothash, string(v.Hash[:]), filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:])), pattern.FragmentSize)
		if err != nil {
			os.Remove(filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:])))
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

// func (n *Node) fetchFile(roothash, fragmentHash, path string) bool {
// 	var err error
// 	var ok bool
// 	var id peer.ID
// 	peers := n.GetAllPeerId()

// 	for _, v := range peers {
// 		id, err = peer.Decode(v)
// 		if err != nil {
// 			continue
// 		}
// 		addr, err := n.GetPeer(v)
// 		if err != nil {
// 			continue
// 		}
// 		err = n.Connect(n.GetCtxQueryFromCtxCancel(), addr)
// 		if err != nil {
// 			continue
// 		}
// 		err = n.ReadFileAction(id, roothash, fragmentHash, path, pattern.FragmentSize)
// 		if err != nil {
// 			continue
// 		}
// 		ok = true
// 		break
// 	}
// 	return ok
// }

func (n *Node) calcFragmentTag(fid, fragment string) error {
	buf, err := os.ReadFile(fragment)
	if err != nil {
		return err
	}
	if len(buf) != pattern.FragmentSize {
		return errors.New("invalid fragment size")
	}
	fragmentHash := filepath.Base(fragment)
	teeEndPoints := n.GetPriorityTeeList()
	teeEndPoints = append(teeEndPoints, n.GetAllMarkerTeeEndpoint()...)
	requestGenTag := &pb.RequestGenTag{
		FragmentData: buf[:pattern.FragmentSize],
		FragmentName: fragmentHash,
		CustomData:   "",
		FileName:     fid,
		MinerId:      n.GetSignatureAccPulickey(),
	}
	var dialOptions []grpc.DialOption
	var teeSign pattern.TeeSig
	for i := 0; i < len(teeEndPoints); i++ {
		teePubkey, err := n.GetTeeWorkAccount(teeEndPoints[i])
		if err != nil {
			n.Restore("info", fmt.Sprintf("[GetTee(%s)] %v", teeEndPoints[i], err))
			continue
		}
		n.Restore("info", fmt.Sprintf("[%s] Will calc file tag: %v", fid, fragmentHash))
		n.Restore("info", fmt.Sprintf("[%s] Will use tee: %v", fid, teeEndPoints[i]))
		if !strings.Contains(teeEndPoints[i], "443") {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		} else {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
		}
		genTag, err := n.RequestGenTag(
			teeEndPoints[i],
			requestGenTag,
			time.Duration(time.Minute*20),
			dialOptions,
			nil,
		)
		if err != nil {
			n.Restore("err", fmt.Sprintf("[RequestGenTag] %v", err))
			continue
		}

		if len(genTag.USig) != pattern.TeeSigLen {
			n.Restore("err", fmt.Sprintf("[RequestGenTag] invalid USig length: %d", len(genTag.USig)))
			continue
		}

		if len(genTag.Signature) != pattern.TeeSigLen {
			n.Restore("err", fmt.Sprintf("[RequestGenTag] invalid TagSigInfo length: %d", len(genTag.Signature)))
			continue
		}
		for j := 0; j < pattern.TeeSigLen; j++ {
			teeSign[j] = types.U8(genTag.Signature[j])
		}

		index := getTagsNumber(filepath.Join(n.GetDirs().FileDir, fid))

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
			n.Restore("err", fmt.Sprintf("[json.Marshal] err: %s", err))
			continue
		}
		ok, err := n.GetPodr2Key().VerifyAttest(genTag.Tag.T.Name, genTag.Tag.T.U, genTag.Tag.PhiHash, genTag.Tag.Attest, "")
		if err != nil {
			n.Restore("err", fmt.Sprintf("[VerifyAttest] err: %s", err))
			continue
		}
		if !ok {
			n.Restore("err", "VerifyAttest is false")
			continue
		}
		err = sutils.WriteBufToFile(buf, fmt.Sprintf("%s.tag", fragment))
		if err != nil {
			n.Restore("err", fmt.Sprintf("[WriteBufToFile] err: %s", err))
			continue
		}
		n.Restore("info", fmt.Sprintf("Calc a service tag: %s", fmt.Sprintf("%s.tag", fragment)))
		break
	}
	return nil
}
