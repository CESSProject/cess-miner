/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"math/big"
	"runtime"
	"strings"

	"github.com/CESSProject/cess-go-sdk/chain"
	sconfig "github.com/CESSProject/cess-go-sdk/config"
	"github.com/CESSProject/cess-miner/pkg/utils"
	"github.com/CESSProject/cess_pois/acc"
	"github.com/CESSProject/cess_pois/pois"
	"github.com/CESSProject/p2p-go/out"
	"github.com/pkg/errors"
)

type Pois struct {
	*pois.Prover
	*acc.RsaKey
}

var minSpace = uint64(pois.FileSize * sconfig.SIZE_1MiB * acc.DEFAULT_ELEMS_NUM * 2)

func NewPOIS(poisDir, spaceDir, accDir string, expendersInfo chain.ExpendersInfo, register bool, front, rear, freeSpace, count int64, cpus int, key_n, key_g, signAccPulickey []byte) (*Pois, error) {
	var err error
	p := &Pois{}

	if len(key_n) != len(chain.PoISKey_N{}) {
		return p, errors.New("[NewPOIS] invalid key_n length")
	}

	if len(key_g) != len(chain.PoISKey_G{}) {
		return p, errors.New("[NewPOIS] invalid key_g length")
	}

	p.RsaKey = &acc.RsaKey{
		N: *new(big.Int).SetBytes(key_n),
		G: *new(big.Int).SetBytes(key_g),
	}
	cfg := pois.Config{
		AccPath:        poisDir,
		IdleFilePath:   spaceDir,
		ChallAccPath:   accDir,
		MaxProofThread: cpus,
	}

	// k,n,d and key are params that needs to be negotiated with the verifier in advance.
	// minerID is storage node's account ID, and space is the amount of physical space available(MiB)
	p.Prover, err = pois.NewProver(
		int64(expendersInfo.K),
		int64(expendersInfo.N),
		int64(expendersInfo.D),
		signAccPulickey,
		freeSpace,
		count,
	)
	if err != nil {
		return p, fmt.Errorf("new pois prover: %v", err)
	}
	if register {
		//Please initialize prover for the first time
		err = p.Prover.Init(*p.RsaKey, cfg)
		if err != nil {
			return p, fmt.Errorf("pois prover init: %v", err)
		}
	} else {
		// If it is downtime recovery, call the recovery method.front and rear are read from minner info on chain
		err = p.Prover.Recovery(*p.RsaKey, front, rear, cfg)
		if err != nil {
			if strings.Contains(err.Error(), "read element data") {
				num := 2
				m, err := utils.GetSysMemAvailable()
				cpuNum := runtime.NumCPU()
				if err == nil {
					m = m * 7 / 10 / (2 * 1024 * 1024 * 1024)
					if int(m) < cpuNum {
						cpuNum = int(m)
					}
					if cpuNum > num {
						num = cpuNum
					}
				}
				out.Tip(fmt.Sprintf("check and restore idle data, use %d cpus", num))
				err = p.Prover.CheckAndRestoreIdleData(front, rear, num)
				if err != nil {
					return p, fmt.Errorf("check and restore idle data: %v", err)
				}
				err = p.Prover.Recovery(*p.RsaKey, front, rear, cfg)
				if err != nil {
					return p, fmt.Errorf("pois prover recovery: %v", err)
				}
			} else {
				return p, fmt.Errorf("pois prover recovery: %v", err)
			}
		}
	}
	p.Prover.AccManager.GetSnapshot()
	return p, nil
}
