package node

import (
	"github.com/CESSProject/cess-bucket/pkg/utils"
)

func (n *Node) restoreMgt(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

}
