/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CESSProject/cess-go-sdk/chain"
	sconfig "github.com/CESSProject/cess-go-sdk/config"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/pkg/utils"
	"github.com/CESSProject/p2p-go/out"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

const (
	fileDir       = "file"
	tmpDir        = "tmp"
	dbDir         = "db"
	logDir        = "log"
	spaceDir      = "space"
	poisDir       = "pois"
	accDir        = "acc"
	randomDir     = "random"
	peer_record   = "peer_record"
	podr2_rsa_pub = "podr2_rsa.pub"
	idle_prove    = "idle_prove"
	service_prove = "service_prove"
)

type Workspacer interface {
	Build(rootDir string) error
	RemoveAndBuild(rootDir string) error
	GetRootDir() string
	GetFileDir() string
	GetTmpDir() string
	GetDbDir() string
	GetLogDir() string
	GetSpaceDir() string
	GetPoisDir() string
	GetPoisAccDir() string
	GetChallRndomDir() string
	GetPeerRecord() string
	GetPodr2Key() string
	GetIdleProve() string
	GetServiceProve() string
	SaveRsaPublicKey(pub []byte) error
	LoadRsaPublicKey() ([]byte, error)
	SaveIdleProve(idleProofRecord idleProofInfo) error
	LoadIdleProve() (idleProofInfo, error)
	SaveServiceProve(serviceProofRecord serviceProofInfo) error
	LoadServiceProve() (serviceProofInfo, error)
	SaveChallRandom(
		challStart uint32,
		randomIndexList []types.U32,
		randomList []chain.Random,
	) error
}

type Workspace struct {
	rootDir       string
	fileDir       string
	tmpDir        string
	dbDir         string
	logDir        string
	spaceDir      string
	poisDir       string
	accDir        string
	randomDir     string
	peer_record   string
	podr2_rsa_pub string
	idle_prove    string
	service_prove string
}

var _ Workspacer = (*Workspace)(nil)

func NewWorkspace() *Workspace {
	return &Workspace{}
}

func (w *Workspace) Check() error {
	dirfreeSpace, err := utils.GetDirFreeSpace(w.rootDir)
	if err != nil {
		return fmt.Errorf("check workspace: %v", err)
	}

	if dirfreeSpace < sconfig.SIZE_1GiB*32 {
		out.Warn("Your free space in workspace is less than 32GiB and cannot generate idle data")
	}
	return nil
}

func (w *Workspace) RemoveAndBuild(rootDir string) error {
	w.rootDir = rootDir
	w.peer_record = filepath.Join(rootDir, peer_record)
	w.podr2_rsa_pub = filepath.Join(rootDir, podr2_rsa_pub)
	w.idle_prove = filepath.Join(rootDir, idle_prove)
	w.service_prove = filepath.Join(rootDir, service_prove)
	w.fileDir = filepath.Join(rootDir, fileDir)
	w.tmpDir = filepath.Join(rootDir, tmpDir)
	w.dbDir = filepath.Join(rootDir, dbDir)
	w.logDir = filepath.Join(rootDir, logDir)
	w.spaceDir = filepath.Join(rootDir, spaceDir)
	w.accDir = filepath.Join(rootDir, accDir)
	w.poisDir = filepath.Join(rootDir, poisDir)
	w.randomDir = filepath.Join(rootDir, randomDir)

	err := os.RemoveAll(w.fileDir)
	if err != nil {
		return err
	}
	err = os.RemoveAll(w.tmpDir)
	if err != nil {
		return err
	}
	err = os.RemoveAll(w.dbDir)
	if err != nil {
		return err
	}
	err = os.RemoveAll(w.logDir)
	if err != nil {
		return err
	}
	err = os.RemoveAll(w.spaceDir)
	if err != nil {
		return err
	}
	err = os.RemoveAll(w.poisDir)
	if err != nil {
		return err
	}
	err = os.RemoveAll(w.accDir)
	if err != nil {
		return err
	}
	err = os.RemoveAll(w.randomDir)
	if err != nil {
		return err
	}

	os.Remove(w.peer_record)
	os.Remove(w.podr2_rsa_pub)
	os.Remove(w.idle_prove)
	os.Remove(w.service_prove)

	err = os.MkdirAll(w.fileDir, configs.FileMode)
	if err != nil {
		return err
	}

	err = os.MkdirAll(w.tmpDir, configs.FileMode)
	if err != nil {
		return err
	}

	err = os.MkdirAll(w.dbDir, configs.FileMode)
	if err != nil {
		return err
	}

	err = os.MkdirAll(w.logDir, configs.FileMode)
	if err != nil {
		return err
	}

	err = os.MkdirAll(w.spaceDir, configs.FileMode)
	if err != nil {
		return err
	}

	err = os.MkdirAll(w.accDir, configs.FileMode)
	if err != nil {
		return err
	}

	err = os.MkdirAll(w.poisDir, configs.FileMode)
	if err != nil {
		return err
	}

	return os.MkdirAll(w.randomDir, configs.FileMode)
}

func (w *Workspace) Build(rootDir string) error {
	w.rootDir = rootDir
	w.peer_record = filepath.Join(rootDir, peer_record)
	w.podr2_rsa_pub = filepath.Join(rootDir, podr2_rsa_pub)
	w.idle_prove = filepath.Join(rootDir, idle_prove)
	w.service_prove = filepath.Join(rootDir, service_prove)

	w.logDir = filepath.Join(rootDir, logDir)
	if err := os.MkdirAll(w.logDir, configs.FileMode); err != nil {
		return err
	}

	w.dbDir = filepath.Join(rootDir, dbDir)
	if err := os.MkdirAll(w.dbDir, configs.FileMode); err != nil {
		return err
	}

	w.accDir = filepath.Join(rootDir, accDir)
	if err := os.MkdirAll(w.accDir, configs.FileMode); err != nil {
		return err
	}

	w.poisDir = filepath.Join(rootDir, poisDir)
	if err := os.MkdirAll(w.poisDir, configs.FileMode); err != nil {
		return err
	}

	w.randomDir = filepath.Join(rootDir, randomDir)
	if err := os.MkdirAll(w.randomDir, configs.FileMode); err != nil {
		return err
	}

	w.spaceDir = filepath.Join(rootDir, spaceDir)
	if err := os.MkdirAll(w.spaceDir, configs.FileMode); err != nil {
		return err
	}

	w.fileDir = filepath.Join(rootDir, fileDir)
	if err := os.MkdirAll(w.fileDir, configs.FileMode); err != nil {
		return err
	}

	w.tmpDir = filepath.Join(rootDir, tmpDir)
	if err := os.MkdirAll(w.tmpDir, configs.FileMode); err != nil {
		return err
	}
	return nil
}

func (w *Workspace) GetRootDir() string {
	return w.rootDir
}
func (w *Workspace) GetFileDir() string {
	return w.fileDir
}
func (w *Workspace) GetTmpDir() string {
	return w.tmpDir
}
func (w *Workspace) GetDbDir() string {
	return w.dbDir
}
func (w *Workspace) GetLogDir() string {
	return w.logDir
}
func (w *Workspace) GetSpaceDir() string {
	return w.spaceDir
}
func (w *Workspace) GetPoisDir() string {
	return w.poisDir
}
func (w *Workspace) GetPoisAccDir() string {
	return w.accDir
}
func (w *Workspace) GetChallRndomDir() string {
	return w.randomDir
}
func (w *Workspace) GetChallRandomDir() string {
	return w.randomDir
}
func (w *Workspace) GetPeerRecord() string {
	return w.peer_record
}
func (w *Workspace) GetPodr2Key() string {
	return w.podr2_rsa_pub
}
func (w *Workspace) GetIdleProve() string {
	return w.idle_prove
}
func (w *Workspace) GetServiceProve() string {
	return w.service_prove
}
func (w *Workspace) SaveRsaPublicKey(pub []byte) error {
	if len(pub) == 0 {
		return errors.New("empty rsa public key")
	}
	return os.WriteFile(w.podr2_rsa_pub, pub, os.ModePerm)
}
func (w *Workspace) LoadRsaPublicKey() ([]byte, error) {
	buf, err := os.ReadFile(w.podr2_rsa_pub)
	if err != nil {
		return nil, err
	}
	_, err = x509.ParsePKCS1PublicKey(buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (w *Workspace) SaveIdleProve(idleProofRecord idleProofInfo) error {
	buf, err := json.Marshal(&idleProofRecord)
	if err != nil {
		return err
	}
	return sutils.WriteBufToFile(buf, w.idle_prove)
}

func (w *Workspace) LoadIdleProve() (idleProofInfo, error) {
	var result idleProofInfo
	buf, err := os.ReadFile(w.idle_prove)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(buf, &result)
	return result, err
}

func (w *Workspace) SaveServiceProve(serviceProofRecord serviceProofInfo) error {
	buf, err := json.Marshal(&serviceProofRecord)
	if err != nil {
		return err
	}
	return sutils.WriteBufToFile(buf, w.service_prove)
}

func (w *Workspace) LoadServiceProve() (serviceProofInfo, error) {
	var result serviceProofInfo
	buf, err := os.ReadFile(w.service_prove)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(buf, &result)
	return result, err
}

func (w *Workspace) SaveChallRandom(
	challStart uint32,
	randomIndexList []types.U32,
	randomList []chain.Random,
) error {
	randfilePath := filepath.Join(w.GetChallRndomDir(), fmt.Sprintf("random.%d", challStart))
	fstat, err := os.Stat(randfilePath)
	if err == nil && fstat.Size() > 0 {
		return nil
	}
	var rd RandomList
	rd.Index = make([]uint32, len(randomIndexList))
	rd.Random = make([][]byte, len(randomIndexList))
	for i := 0; i < len(randomIndexList); i++ {
		rd.Index[i] = uint32(randomIndexList[i])
		rd.Random[i] = make([]byte, len(randomList[i]))
		for j := 0; j < len(randomList[i]); j++ {
			rd.Random[i][j] = byte(randomList[i][j])
		}
	}
	buff, err := json.Marshal(&rd)
	if err != nil {
		return fmt.Errorf("[json.Marshal] %v", err)
	}

	f, err := os.OpenFile(randfilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("[OpenFile] %v", err)
	}
	defer f.Close()
	_, err = f.Write(buff)
	if err != nil {
		return fmt.Errorf("[Write] %v", err)
	}
	return f.Sync()
}
