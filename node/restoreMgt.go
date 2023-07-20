package node

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/erasure"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
)

func (n *Node) restoreMgt(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	n.Restore("info", ">>>>> start restoreMgt <<<<<")
	for {
		for n.GetChainState() {
			time.Sleep(time.Minute)
			minerInfo, err := n.QueryStorageMiner(n.GetStakingPublickey())
			if err != nil {
				time.Sleep(time.Minute)
				continue
			}

			if string(minerInfo.State) != "positive" {
				continue
			}

			err = n.inspector()
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
		time.Sleep(pattern.BlockInterval)
	}
}

func (n *Node) inspector() error {
	var (
		err      error
		roothash string
		txhash   string
		fmeta    pattern.FileMetadata
	)

	roothashes, err := utils.Dirs(n.GetDirs().FileDir)
	if err != nil {
		n.Restore("err", fmt.Sprintf("[Dir %v] %v", n.GetDirs().FileDir, err))
		roothashes, err = n.QueryPrefixKeyList(Cach_prefix_metadata)
		if err != nil {
			return errors.Wrapf(err, "[QueryPrefixKeyList]")
		}
	}

	for _, v := range roothashes {
		roothash = filepath.Base(v)
		fmeta, err = n.QueryFileMetadata(roothash)
		if err != nil {
			if err.Error() == pattern.ERR_Empty {
				os.RemoveAll(v)
				continue
			}
			n.Restore("err", fmt.Sprintf("[QueryFileMetadata %v] %v", roothash, err))
			continue
		}
		for _, segment := range fmeta.SegmentList {
			for _, fragment := range segment.FragmentList {
				if sutils.CompareSlice(fragment.Miner[:], n.GetStakingPublickey()) {
					_, err = os.Stat(filepath.Join(n.GetDirs().FileDir, roothash, string(fragment.Hash[:])))
					if err != nil {
						err = n.restoreFragment(roothashes, roothash, string(fragment.Hash[:]), segment)
						if err != nil {
							os.Remove(filepath.Join(n.GetDirs().FileDir, roothash, string(fragment.Hash[:])))
							n.Restore("err", fmt.Sprintf("[restoreFragment %v] %v", roothash, err))
							if ok, err := n.Has([]byte(Cach_prefix_MyLost + string(fragment.Hash[:]))); !ok {
								txhash, err = n.GenerateRestoralOrder(roothash, string(fragment.Hash[:]))
								if err != nil {
									n.Restore("err", fmt.Sprintf("[GenerateRestoralOrder %v] %v", roothash, err))
								} else {
									n.Put([]byte(Cach_prefix_MyLost+string(fragment.Hash[:])), nil)
									n.Restore("info", fmt.Sprintf("[GenerateRestoralOrder %v-%v] %v", roothash, string(fragment.Hash[:]), txhash))
								}
							}
							continue
						}
						n.Delete([]byte(Cach_prefix_MyLost + string(fragment.Hash[:])))
						n.Delete([]byte(Cach_prefix_recovery + string(fragment.Hash[:])))
					}
				}
			}
		}
	}

	return nil
}

func (n *Node) restoreFragment(roothashes []string, roothash, framentHash string, segement pattern.SegmentInfo) error {
	var err error
	var id peer.ID
	var miner pattern.MinerInfo
	n.Restore("info", fmt.Sprintf("[%s] To restore the fragment: %s", roothash, framentHash))

	_, err = os.Stat(filepath.Join(n.GetDirs().FileDir, roothash))
	if err != nil {
		os.MkdirAll(filepath.Join(n.GetDirs().FileDir, roothash), pattern.DirMode)
	}

	for _, v := range roothashes {
		_, err = os.Stat(filepath.Join(v, framentHash))
		if err == nil {
			err = utils.CopyFile(filepath.Join(n.GetDirs().FileDir, roothash, framentHash), filepath.Join(v, framentHash))
			if err == nil {
				return nil
			}
		}
	}
	var canRestore int
	var recoverList = make([]string, len(segement.FragmentList))
	for k, v := range segement.FragmentList {
		if string(v.Hash[:]) == framentHash {
			recoverList[k] = ""
			continue
		}
		_, err = os.Stat(filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:])))
		if err == nil {
			n.Restore("info", fmt.Sprintf("[%s] found a fragment: %s", roothash, string(v.Hash[:])))
			recoverList[k] = filepath.Join(n.GetDirs().FileDir, roothash, string(v.Hash[:]))
			canRestore++
			if canRestore >= int(len(segement.FragmentList)*2/3) {
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
		addr, ok := n.GetPeer(id.Pretty())
		if !ok {
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
		if canRestore >= int(len(segement.FragmentList)*2/3) {
			break
		}
	}
	n.Restore("info", fmt.Sprintf("all found frgments: %v", recoverList))
	segmentpath := filepath.Join(n.GetDirs().FileDir, roothash, string(segement.Hash[:]))
	if canRestore >= int(len(segement.FragmentList)*2/3) {
		err = n.RedundancyRecovery(segmentpath, recoverList)
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
		os.Remove(segmentpath)
	} else {
		n.Restore("err", fmt.Sprintf("[%s] There are not enough fragments to recover the segment %s", roothash, string(segement.Hash[:])))
		return errors.New("recpvery failed")
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

		if !sutils.CompareSlice(restoreOrder.Miner[:], n.GetStakingPublickey()) {
			continue
		}

		txhash, err := n.RestoralComplete(v)
		if err != nil {
			n.Restore("err", fmt.Sprintf("[RestoralComplete %s-%s] %v", string(b), v, err))
			continue
		}
		n.Restore("info", fmt.Sprintf("[RestoralComplete %s-%s] %s", string(b), v, txhash))
		n.Delete([]byte(Cach_prefix_recovery + v))
	}

	restoreOrderList, err := n.QueryRestoralOrderList()
	if err != nil {
		n.Restore("err", fmt.Sprintf("[QueryRestoralOrderList] %v", err))
		return err
	}
	blockHeight, err := n.QueryBlockHeight("")
	if err != nil {
		n.Restore("err", fmt.Sprintf("[QueryBlockHeight] %v", err))
		return err
	}
	for _, v := range restoreOrderList {
		if blockHeight <= uint32(v.Deadline) {
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
		os.MkdirAll(filepath.Join(n.GetDirs().FileDir, roothash), pattern.DirMode)
	}
	roothashes, err := utils.Dirs(n.GetDirs().FileDir)
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

		addr, ok := n.GetPeer(peerid)
		if !ok {
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
		err = n.RedundancyRecovery(segmentpath, recoverList)
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
	var ok bool
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
		for v, _ := range roothashs {
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
						if ok, err = n.Has([]byte(Cach_prefix_recovery + string(fragment.Hash[:]))); ok {
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

func (n *Node) fetchFile(roothash, fragmentHash, path string) bool {
	var err error
	var ok bool
	var id peer.ID
	peers := n.GetAllPeerId()

	for _, v := range peers {
		id, err = peer.Decode(v)
		if err != nil {
			continue
		}
		addr, ok := n.GetPeer(v)
		if !ok {
			continue
		}
		err = n.Connect(n.GetCtxQueryFromCtxCancel(), addr)
		if err != nil {
			continue
		}
		err = n.ReadFileAction(id, roothash, fragmentHash, path, pattern.FragmentSize)
		if err != nil {
			continue
		}
		ok = true
		break
	}
	return ok
}
