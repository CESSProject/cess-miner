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

	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
)

type TeeRecord interface {
	// SaveTee saves or updates tee information
	SaveTee(workAccount, endPoint string, teeType uint8) error
	//
	GetTee(workAccount string) (TeeInfoType, error)
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

type TeeInfoType struct {
	EndPoint string
	Type     uint8
}

type TeeRecordType struct {
	lock                 *sync.RWMutex
	priorityTeeEndpoints []string
	teeList              map[string]TeeInfoType
}

var _ TeeRecord = (*TeeRecordType)(nil)

func NewTeeRecord() TeeRecord {
	return &TeeRecordType{
		lock:                 new(sync.RWMutex),
		priorityTeeEndpoints: make([]string, 0),
		teeList:              make(map[string]TeeInfoType, 10),
	}
}

// SaveTee saves or updates tee information
func (t *TeeRecordType) SaveTee(workAccount, endPoint string, teeType uint8) error {
	if workAccount == "" {
		return errors.New("work account is empty")
	}
	if endPoint == "" {
		return errors.New("endPoint is empty")
	}
	if teeType > pattern.TeeType_Marker {
		return errors.New("invalid tee type")
	}
	if utils.ContainsIpv4(endPoint) {
		endPoint = strings.TrimPrefix(endPoint, "http://")
	}
	var data = TeeInfoType{
		EndPoint: endPoint,
		Type:     teeType,
	}
	t.lock.Lock()
	t.teeList[workAccount] = data
	t.lock.Unlock()
	return nil
}

func (t *TeeRecordType) GetTee(workAccount string) (TeeInfoType, error) {
	t.lock.RLock()
	result, ok := t.teeList[workAccount]
	t.lock.RUnlock()
	if !ok {
		return TeeInfoType{}, errors.New("not found")
	}
	return result, nil
}

func (t *TeeRecordType) GetTeeWorkAccount(endpoint string) (string, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	for k, v := range t.teeList {
		if v.EndPoint == endpoint {
			return k, nil
		}
	}
	return "", errors.New("not found")
}

func (t *TeeRecordType) DeleteTee(workAccount string) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if _, ok := t.teeList[workAccount]; ok {
		delete(t.teeList, workAccount)
	}
}

func (t *TeeRecordType) GetAllTeeEndpoint() []string {
	var result = make([]string, 0)
	t.lock.RLock()
	defer t.lock.RUnlock()
	for _, v := range t.teeList {
		result = append(result, v.EndPoint)
	}
	return result
}

func (t *TeeRecordType) GetAllMarkerTeeEndpoint() []string {
	var result = make([]string, 0)
	t.lock.RLock()
	defer t.lock.RUnlock()
	for _, v := range t.teeList {
		if v.Type == pattern.TeeType_Full || v.Type == pattern.TeeType_Marker {
			result = append(result, v.EndPoint)
		}
	}
	return result
}

func (t *TeeRecordType) GetAllVerifierTeeEndpoint() []string {
	var result = make([]string, 0)
	t.lock.RLock()
	defer t.lock.RUnlock()
	for _, v := range t.teeList {
		if v.Type == pattern.TeeType_Full || v.Type == pattern.TeeType_Verifier {
			result = append(result, v.EndPoint)
		}
	}
	return result
}
