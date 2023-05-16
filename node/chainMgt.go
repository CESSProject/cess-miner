/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"time"

	"github.com/CESSProject/cess-bucket/pkg/utils"
)

func (n *Node) chainMgt(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Log.Pnc(utils.RecoverError(err))
		}
	}()
	var ok bool
	var err error
	var customTag string
	tick := time.NewTicker(time.Minute)
	for {
		select {
		case <-tick.C:
			ok, err = n.Cli.Chain.NetListening()
			if !ok || err != nil {
				n.Cli.Chain.SetChainState(false)
				n.Cli.Chain.Reconnect()
			}
		case customTag = <-n.Cli.GetServiceTagEvent():
			fmt.Println("Received custom tag: ", customTag)
		}
	}
}
