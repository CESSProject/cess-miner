/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

func (n *Node) TaskMgt() {
	var (
		ch_chainMgt      = make(chan bool, 1)
		ch_spaceMgt      = make(chan bool, 1)
		ch_fileMgt       = make(chan bool, 1)
		ch_replaceMgr    = make(chan bool, 1)
		ch_challengeMgt  = make(chan bool, 1)
		ch_stagMgt       = make(chan bool, 1)
		ch_restoreMgt    = make(chan bool, 1)
		ch_discoverMgt   = make(chan bool, 1)
		ch_parseBlockMgt = make(chan bool, 1)
	)

	go n.chainMgt(ch_chainMgt)
	go n.spaceMgt(ch_spaceMgt)
	go n.fileMgt(ch_fileMgt)
	go n.replaceMgr(ch_replaceMgr)
	go n.challengeMgt(ch_challengeMgt)
	go n.stagMgt(ch_stagMgt)
	go n.restoreMgt(ch_restoreMgt)
	go n.discoverMgt(ch_discoverMgt)
	// go n.parseBlockMgt(ch_parseBlockMgt)

	for {
		select {
		case <-ch_chainMgt:
			go n.chainMgt(ch_chainMgt)
		case <-ch_spaceMgt:
			go n.spaceMgt(ch_spaceMgt)
		case <-ch_fileMgt:
			go n.fileMgt(ch_fileMgt)
		case <-ch_replaceMgr:
			go n.replaceMgr(ch_replaceMgr)
		case <-ch_challengeMgt:
			go n.challengeMgt(ch_challengeMgt)
		case <-ch_stagMgt:
			go n.stagMgt(ch_stagMgt)
		case <-ch_restoreMgt:
			go n.restoreMgt(ch_restoreMgt)
		case <-ch_discoverMgt:
			go n.discoverMgt(ch_discoverMgt)
		case <-ch_parseBlockMgt:
			go n.parseBlockMgt(ch_parseBlockMgt)
		}
	}
}
