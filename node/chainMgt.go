/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"time"

	"github.com/CESSProject/cess-bucket/pkg/utils"
)

func (n *Node) chainMgt(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()
	var ok bool
	var err error
	tick := time.NewTicker(time.Minute)
	for {
		select {
		case <-tick.C:
			ok, err = n.NetListening()
			if !ok || err != nil {
				n.SetChainState(false)
				n.Reconnect()
			}
		case <-n.GetServiceTagCh():
		}
	}
}
