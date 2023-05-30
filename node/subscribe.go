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
	"github.com/CESSProject/sdk-go/core/event"
	"github.com/CESSProject/sdk-go/core/rule"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

func (n *Node) SubscribeNewHeads(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	for {

		if n.GetChainState() {

			sub, err := n.GetSubstrateAPI().RPC.Chain.SubscribeNewHeads()
			if err != nil {
				time.Sleep(rule.BlockInterval)
				continue
			}
			defer sub.Unsubscribe()

			for {
				head := <-sub.Chan()
				fmt.Printf("Chain is at block: #%v\n", head.Number)
				blockhash, err := n.GetSubstrateAPI().RPC.Chain.GetBlockHash(uint64(head.Number))
				if err != nil {
					continue
				}
				h, err := n.GetSubstrateAPI().RPC.State.GetStorageRaw(n.GetKeyEvents(), blockhash)
				if err != nil {
					continue
				}
				var events = event.EventRecords{}
				types.EventRecordsRaw(*h).DecodeEventRecords(n.GetMetadata(), &events)

				//TODO: Corresponding processing according to different events
			}
		}
	}
}
