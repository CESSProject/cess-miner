package node

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/sdk-go/core/pattern"
	sutils "github.com/CESSProject/sdk-go/core/utils"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
)

func (n *Node) restoreMgt(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	n.Restore("info", ">>>>> Start restoreMgt")
	for {
		for n.GetChainState() {
			err := n.inspector()
			if err != nil {
				n.Restore("err", err.Error())
				time.Sleep(pattern.BlockInterval)
			} else {
				time.Sleep(time.Minute)
			}
		}
		time.Sleep(pattern.BlockInterval)
	}
}

func (n *Node) inspector() error {
	var (
		err      error
		roothash string
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
		fmeta, err = n.QueryFileMetadata(v)
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
						ok := n.fetchFile(v, string(fragment.Hash[:]), filepath.Join(n.GetDirs().FileDir, v, string(fragment.Hash[:])))
						if !ok {
							// report miss
						}
					}
				}
			}
		}
	}

	return nil
}

func (n *Node) restoreFragment(roothash string, segement pattern.SegmentInfo) {

}

func (n *Node) fetchFile(roothash, fragmentHash, path string) bool {
	var err error
	var ok bool
	var id peer.ID
	peers := n.GetAllTeePeerId()

	for _, v := range peers {
		id, err = peer.Decode(v)
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
