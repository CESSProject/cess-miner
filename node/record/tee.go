/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package record

import (
	"encoding/hex"
	"errors"
	"strings"
	"sync"

	"github.com/CESSProject/cess-go-sdk/chain"
)

type TeeRecorder interface {
	// SaveTee saves or updates tee information
	SaveTee(pubkeyhex, endPoint string, teeType uint8) error
	//
	GetTee(pubkeyhex string) (TeeInfo, error)
	//
	GetTeePubkeyHexByEndpoint(endpoint string) (string, error)
	//
	GetTeePubkeyByEndpoint(endpoint string) ([]byte, error)
	//
	DeleteTee(pubkeyhex string)
	//
	GetAllTeePubkeyHex() []string
	//
	GetAllTeeEndpoint() []string
	//
	GetAllMarkerTeeEndpoint() []string
	//
	GetAllVerifierTeeEndpoint() []string
	//
	Length() int
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

func NewTeeRecord() TeeRecorder {
	return &TeeRecord{
		lock:                 new(sync.RWMutex),
		priorityTeeEndpoints: make([]string, 0),
		teeList:              make(map[string]TeeInfo, 10),
	}
}

// SaveTee saves or updates tee information
func (t *TeeRecord) SaveTee(pubkeyhex, endPoint string, teeType uint8) error {
	if pubkeyhex == "" {
		return errors.New("publickey is empty")
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
	t.teeList[pubkeyhex] = data
	t.lock.Unlock()
	return nil
}

func (t *TeeRecord) Length() int {
	t.lock.RLock()
	length := len(t.teeList)
	t.lock.RUnlock()
	return length
}

func (t *TeeRecord) GetTee(pubkeyhex string) (TeeInfo, error) {
	t.lock.RLock()
	result, ok := t.teeList[pubkeyhex]
	t.lock.RUnlock()
	if !ok {
		return TeeInfo{}, errors.New("not found")
	}
	return result, nil
}

func (t *TeeRecord) GetTeePubkeyHexByEndpoint(endpoint string) (string, error) {
	pubkeyHex := ""
	t.lock.RLock()
	for k, v := range t.teeList {
		if v.EndPoint == endpoint {
			pubkeyHex = k
			break
		}
	}
	t.lock.RUnlock()
	if pubkeyHex == "" {
		return "", errors.New("not found")
	}
	return pubkeyHex, nil
}

func (t *TeeRecord) GetTeePubkeyByEndpoint(endpoint string) ([]byte, error) {
	pubkeyHex := ""
	t.lock.RLock()
	for k, v := range t.teeList {
		if v.EndPoint == endpoint {
			pubkeyHex = k
			break
		}
	}
	t.lock.RUnlock()
	if pubkeyHex == "" {
		return nil, errors.New("not found")
	}
	return hex.DecodeString(pubkeyHex)
}

func (t *TeeRecord) DeleteTee(pubkeyhex string) {
	t.lock.Lock()
	if _, ok := t.teeList[pubkeyhex]; ok {
		delete(t.teeList, pubkeyhex)
	}
	t.lock.Unlock()
}

func (t *TeeRecord) GetAllTeeEndpoint() []string {
	var result = make([]string, 0)
	t.lock.RLock()
	result = t.priorityTeeEndpoints
	for _, v := range t.teeList {
		result = append(result, v.EndPoint)
	}
	t.lock.RUnlock()
	return result
}

func (t *TeeRecord) GetAllTeePubkeyHex() []string {
	var index int
	t.lock.RLock()
	var result = make([]string, len(t.teeList))
	for k := range t.teeList {
		result[index] = k
		index++
	}
	t.lock.RUnlock()
	return result
}

func (t *TeeRecord) GetAllMarkerTeeEndpoint() []string {
	var result = make([]string, 0)
	t.lock.RLock()
	for _, v := range t.teeList {
		if v.Type == chain.TeeType_Full || v.Type == chain.TeeType_Marker {
			result = append(result, v.EndPoint)
		}
	}
	t.lock.RUnlock()
	return result
}

func (t *TeeRecord) GetAllVerifierTeeEndpoint() []string {
	var result = make([]string, 0)
	t.lock.RLock()
	for _, v := range t.teeList {
		if v.Type == chain.TeeType_Full || v.Type == chain.TeeType_Verifier {
			result = append(result, v.EndPoint)
		}
	}
	t.lock.RUnlock()
	return result
}
