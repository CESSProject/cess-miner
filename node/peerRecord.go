/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/AstaFrode/go-libp2p/core/peer"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
)

type PeerRecord interface {
	// SavePeer saves or updates peer information
	SavePeer(addr peer.AddrInfo) error
	//
	HasPeer(peerid string) bool
	//
	GetPeer(peerid string) (peer.AddrInfo, error)
	//
	GetAllPeerId() []string
	//
	BackupPeer(path string) error
	//
	LoadPeer(path string) error
}

type PeerRecordType struct {
	lock     *sync.RWMutex
	peerList map[string]peer.AddrInfo
}

var _ PeerRecord = (*PeerRecordType)(nil)

func NewPeerRecord() PeerRecord {
	return &PeerRecordType{
		lock:     new(sync.RWMutex),
		peerList: make(map[string]peer.AddrInfo, 100),
	}
}

func (p *PeerRecordType) SavePeer(addr peer.AddrInfo) error {
	if addr.ID.Pretty() == "" {
		return errors.New("peer id is empty")
	}

	if addr.Addrs == nil {
		return errors.New("peer addrs is nil")
	}

	p.lock.Lock()
	p.peerList[addr.ID.Pretty()] = addr
	p.lock.Unlock()
	return nil
}

func (p *PeerRecordType) HasPeer(peerid string) bool {
	p.lock.RLock()
	_, ok := p.peerList[peerid]
	p.lock.RUnlock()
	return ok
}

func (p *PeerRecordType) GetPeer(peerid string) (peer.AddrInfo, error) {
	p.lock.RLock()
	result, ok := p.peerList[peerid]
	p.lock.RUnlock()
	if !ok {
		return peer.AddrInfo{}, errors.New("not found")
	}
	return result, nil
}

func (p *PeerRecordType) GetAllPeerId() []string {
	var result = make([]string, len(p.peerList))
	p.lock.RLock()
	defer p.lock.RUnlock()
	var i int
	for k, _ := range p.peerList {
		result[i] = k
		i++
	}
	return result
}

func (p *PeerRecordType) BackupPeer(path string) error {
	p.lock.RLock()
	buf, err := json.Marshal(p.peerList)
	if err != nil {
		p.lock.RUnlock()
		return err
	}
	p.lock.RUnlock()
	err = sutils.WriteBufToFile(buf, path)
	return err
}

func (p *PeerRecordType) LoadPeer(path string) error {
	buf, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	oldPeer := p.peerList
	p.lock.Lock()
	err = json.Unmarshal(buf, &p.peerList)
	p.lock.Unlock()
	if err != nil {
		p.peerList = oldPeer
	}
	return err
}
