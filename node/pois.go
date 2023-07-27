/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"

	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/cess_pois/acc"
	"github.com/CESSProject/cess_pois/pois"
)

type Pois struct {
	*pois.Prover
	*acc.RsaKey
	pattern.ExpendersInfo
	front int64
	rear  int64
}

func (n *Node) InitPois(front, rear int64) error {
	var err error

	n.Pois = &Pois{
		Prover: new(pois.Prover),
		RsaKey: new(acc.RsaKey),
		front:  front,
		rear:   rear,
	}

	// k,n,d and key are params that needs to be negotiated with the verifier in advance.
	// minerID is storage node's account ID, and space is the amount of physical space available(MiB)
	n.Prover, err = pois.NewProver(int64(n.ExpendersInfo.K), int64(n.ExpendersInfo.N), int64(n.ExpendersInfo.D), n.GetSignatureAccPulickey(), 256)
	if err != nil {
		return err
	}

	//Please initialize prover for the first time
	err = n.Prover.Init(*n.Pois.RsaKey)
	if err != nil {
		return err
	}

	// If it is downtime recovery, call the recovery method.front and rear are read from minner info on chain
	err = n.Prover.Recovery(*n.Pois.RsaKey, front, rear)
	if err != nil {
		return err
	}

	// Run the idle file generation service, it returns a channel (recorded in the prover object),
	// insert the file ID into the channel to automatically generate idle files.
	// The number of threads started by default is pois.MaxProofThread(he number of files generation supported at the same time)
	n.Prover.RunIdleFileGenerationServer(pois.MaxProofThread)
	return nil
}

func (n *Node) pois() error {
	// Generate Idle Files
	if ok := n.Prover.GenerateFile(1); !ok {
		return fmt.Errorf("GenerateFile failed")
	}

	return nil
}
