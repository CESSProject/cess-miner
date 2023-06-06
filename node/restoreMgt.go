package node

import (
	"os"
	"path/filepath"

	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/sdk-go/core/pattern"
	sutils "github.com/CESSProject/sdk-go/core/utils"
	"github.com/libp2p/go-libp2p/core/peer"
)

func (n *Node) restoreMgt(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()
	n.inspector()
}

func (n *Node) inspector() error {
	roothashList, err := n.QueryPrefixKeyList(Cach_prefix_metadata)
	if err != nil {
		return err
	}
	for _, v := range roothashList {
		fmeta, err := n.QueryFileMetadata(v)
		if err != nil {
			continue
		}
		for _, segement := range fmeta.SegmentList {
			for _, fragment := range segement.FragmentList {
				if sutils.CompareSlice(fragment.Miner[:], n.GetStakingPublickey()) {
					_, err = os.Stat(filepath.Join(n.GetDirs().FileDir, v, string(fragment.Hash[:])))
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
	peers := n.GetAllPeer()

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
