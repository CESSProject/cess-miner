/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"time"

	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/cess_pois/acc"
	"github.com/CESSProject/cess_pois/pois"
	"github.com/pkg/errors"
)

type Pois struct {
	*pois.Prover
	*acc.RsaKey
	pattern.ExpendersInfo
	teePeerid string
	front     int64
	rear      int64
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
	var err error
	var commits []pois.Commit

	tickCheck := time.NewTicker(time.Minute)
	defer tickCheck.Stop()
	tickTimeout := time.NewTicker(time.Hour)
	defer tickTimeout.Stop()

	for {
		select {
		case <-tickCheck.C:
			commits, err = n.Prover.GetCommits(1)
			if err == nil {
				break
			}
		case <-tickTimeout.C:
			return errors.New("generate idle file timeout")
		}
		if commits != nil {
			break
		}
	}

	var chals [][]int64
	// TODO: send commits to tee and receive chall

	commitProofs, accProof, err := n.Prover.ProveCommitAndAcc(chals)
	if err != nil {
		return errors.Wrapf(err, "[ProveCommitAndAcc]")
	}
	if err == nil && commitProofs == nil && accProof == nil {
		// If the results are all nil, it means that other programs are updating the data of the prover object.
		return errors.New("other programs are updating the data of the prover object")
	}

	var ok bool
	var idleSignInfo pattern.IdleSignInfo
	var sign pattern.TeeSignature
	// TODO: send commitProofs and accProof to verifier and then wait for the response

	// If the challenge is failure, need to roll back the prover to the previous status,
	// this method will return whether the rollback is successful, and its parameter is also whether it is a delete operation be rolled back.
	if !ok {
		n.Prover.AccRollback(false)
		return nil
	}

	//
	txhash, err := n.CertIdleSpace(idleSignInfo, sign)
	if err != nil {
		n.Prover.AccRollback(false)
		return errors.Wrapf(err, "[CertIdleSpace]")
	}
	txhash = txhash

	// If the challenge is successful, update the prover status, fileNum is challenged files number,
	// the second parameter represents whether it is a delete operation, and the commit proofs should belong to the joining files, so it is false
	err = n.Prover.UpdateStatus(1, false)
	if err != nil {
		return errors.Wrapf(err, "[UpdateStatus]")
	}

	n.Prover.SetChallengeState(false)

	return nil
}
