/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"errors"
	"strings"
	"sync"

	"github.com/CESSProject/cess-go-sdk/chain"
)

type TeeRecorder interface {
	// SaveTee saves or updates tee information
	SaveTee(workAccount, endPoint string, teeType uint8) error
	//
	GetTee(workAccount string) (TeeInfo, error)
	//
	GetTeeWorkAccount(endpoint string) (string, error)
	//
	DeleteTee(workAccount string)
	//
	GetAllTeeEndpoint() []string
	//
	GetAllMarkerTeeEndpoint() []string
	//
	GetAllVerifierTeeEndpoint() []string
}

type TeeInfo struct {
	EndPoint string
	Type     uint8
}

type TeeRecord struct {
	lock                 *sync.RWMutex
	priorityTeeEndpoints []string
	teeList              map[string]TeeInfo
}

var _ TeeRecorder = (*TeeRecord)(nil)

func NewTeeRecord() *TeeRecord {
	return &TeeRecord{
		lock:                 new(sync.RWMutex),
		priorityTeeEndpoints: make([]string, 0),
		teeList:              make(map[string]TeeInfo, 10),
	}
}

// SaveTee saves or updates tee information
func (t *TeeRecord) SaveTee(workAccount, endPoint string, teeType uint8) error {
	if workAccount == "" {
		return errors.New("work account is empty")
	}
	if endPoint == "" {
		return errors.New("endPoint is empty")
	}
	if teeType > chain.TeeType_Marker {
		return errors.New("invalid tee type")
	}
	var teeEndPoint string

	if strings.HasPrefix(endPoint, "http://") {
		teeEndPoint = strings.TrimPrefix(endPoint, "http://")
		teeEndPoint = strings.TrimSuffix(teeEndPoint, "/")
		if !strings.Contains(teeEndPoint, ":") {
			teeEndPoint = teeEndPoint + ":80"
		}
	} else if strings.HasPrefix(endPoint, "https://") {
		teeEndPoint = strings.TrimPrefix(endPoint, "https://")
		teeEndPoint = strings.TrimSuffix(teeEndPoint, "/")
		if !strings.Contains(teeEndPoint, ":") {
			teeEndPoint = teeEndPoint + ":443"
		}
	} else {
		if !strings.Contains(endPoint, ":") {
			teeEndPoint = endPoint + ":80"
		} else {
			teeEndPoint = endPoint
		}
	}

	var data = TeeInfo{
		EndPoint: teeEndPoint,
		Type:     teeType,
	}
	t.lock.Lock()
	t.teeList[workAccount] = data
	t.lock.Unlock()
	return nil
}

func (t *TeeRecord) GetTee(workAccount string) (TeeInfo, error) {
	t.lock.RLock()
	result, ok := t.teeList[workAccount]
	t.lock.RUnlock()
	if !ok {
		return TeeInfo{}, errors.New("not found")
	}
	return result, nil
}

func (t *TeeRecord) GetTeeWorkAccount(endpoint string) (string, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	for k, v := range t.teeList {
		if v.EndPoint == endpoint {
			return k, nil
		}
	}
	return "", errors.New("not found")
}

func (t *TeeRecord) DeleteTee(workAccount string) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if _, ok := t.teeList[workAccount]; ok {
		delete(t.teeList, workAccount)
	}
}

func (t *TeeRecord) GetAllTeeEndpoint() []string {
	var result = make([]string, 0)
	t.lock.RLock()
	defer t.lock.RUnlock()
	result = t.priorityTeeEndpoints
	for _, v := range t.teeList {
		if v.EndPoint == "" {
			continue
		}
		result = append(result, v.EndPoint)
	}
	return result
}

func (t *TeeRecord) GetAllMarkerTeeEndpoint() []string {
	var result = make([]string, 0)
	t.lock.RLock()
	defer t.lock.RUnlock()
	for _, v := range t.teeList {
		if v.Type == chain.TeeType_Full || v.Type == chain.TeeType_Marker {
			if v.EndPoint == "" {
				continue
			}
			result = append(result, v.EndPoint)
		}
	}
	return result
}

func (t *TeeRecord) GetAllVerifierTeeEndpoint() []string {
	var result = make([]string, 0)
	t.lock.RLock()
	defer t.lock.RUnlock()
	for _, v := range t.teeList {
		if v.Type == chain.TeeType_Full || v.Type == chain.TeeType_Verifier {
			result = append(result, v.EndPoint)
		}
	}
	return result
}
