/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

func (n *Node) TaskMgt() {
	var (
		ch_chainMgt     = make(chan bool, 1)
		ch_spaceMgr     = make(chan bool, 1)
		ch_fileMgr      = make(chan bool, 1)
		ch_replaceMgr   = make(chan bool, 1)
		ch_challengeMgr = make(chan bool, 1)
	)

	go n.chainMgt(ch_chainMgt)
	go n.spaceMgt(ch_spaceMgr)
	go n.fileMgt(ch_fileMgr)
	go n.replaceMgr(ch_replaceMgr)
	go n.challengeMgr(ch_challengeMgr)

	for {
		select {
		case <-ch_chainMgt:
			go n.chainMgt(ch_chainMgt)
		case <-ch_spaceMgr:
			go n.spaceMgt(ch_spaceMgr)
		case <-ch_fileMgr:
			go n.fileMgt(ch_fileMgr)
		case <-ch_replaceMgr:
			go n.replaceMgr(ch_replaceMgr)
		case <-ch_challengeMgr:
			go n.challengeMgr(ch_challengeMgr)
		}
	}
}
