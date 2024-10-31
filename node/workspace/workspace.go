/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CESSProject/cess-go-sdk/chain"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/node/common"
	out "github.com/CESSProject/cess-miner/pkg/fout"
	"github.com/CESSProject/cess-miner/pkg/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

const (
	fileDir       = "file"
	reportDir     = "report"
	tmpDir        = "tmp"
	dbDir         = "db"
	logDir        = "log"
	spaceDir      = "space"
	poisDir       = "pois"
	accDir        = "acc"
	randomDir     = "random"
	idle_proof    = "idle_proof"
	service_proof = "service_proof"
)

type Workspace interface {
	Build() error
	RemoveAndBuild() error
	GetRootDir() string
	GetFileDir() string
	GetReportDir() string
	GetTmpDir() string
	GetDbDir() string
	GetLogDir() string
	GetSpaceDir() string
	GetPoisDir() string
	GetPoisAccDir() string
	GetChallRndomDir() string
	GetPodr2Key() string
	GetIdleProve() string
	GetServiceProve() string
	SaveIdleProve(idleProofRecord common.IdleProofInfo) error
	LoadIdleProve() (common.IdleProofInfo, error)
	SaveServiceProve(serviceProofRecord common.ServiceProofInfo) error
	LoadServiceProve() (common.ServiceProofInfo, error)
	SaveChallRandom(
		challStart uint32,
		randomIndexList []types.U32,
		randomList []chain.Random,
	) error
}

type workspace struct {
	rootDir       string
	fileDir       string
	reportDir     string
	tmpDir        string
	dbDir         string
	logDir        string
	spaceDir      string
	poisDir       string
	accDir        string
	randomDir     string
	podr2_rsa_pub string
	idle_prove    string
	service_prove string
}

var _ Workspace = (*workspace)(nil)

func NewWorkspace(ws string) Workspace {
	return &workspace{rootDir: ws}
}

func (w *workspace) Check() error {
	dirfreeSpace, err := utils.GetDirFreeSpace(w.rootDir)
	if err != nil {
		return fmt.Errorf("check workspace: %v", err)
	}

	if dirfreeSpace < chain.SIZE_1GiB*32 {
		out.Warn("Your free space in workspace is less than 32GiB and cannot generate idle data")
	}
	return nil
}

func (w *workspace) RemoveAndBuild() error {
	if w.rootDir == "" {
		return fmt.Errorf("Please initialize the workspace first")
	}
	w.idle_prove = filepath.Join(w.rootDir, idle_proof)
	w.service_prove = filepath.Join(w.rootDir, service_proof)
	w.fileDir = filepath.Join(w.rootDir, fileDir)
	w.reportDir = filepath.Join(w.rootDir, reportDir)
	w.tmpDir = filepath.Join(w.rootDir, tmpDir)
	w.dbDir = filepath.Join(w.rootDir, dbDir)
	w.logDir = filepath.Join(w.rootDir, logDir)
	w.spaceDir = filepath.Join(w.rootDir, spaceDir)
	w.accDir = filepath.Join(w.rootDir, accDir)
	w.poisDir = filepath.Join(w.rootDir, poisDir)
	w.randomDir = filepath.Join(w.rootDir, randomDir)

	err := os.RemoveAll(w.fileDir)
	if err != nil {
		return err
	}
	err = os.RemoveAll(w.reportDir)
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

	os.Remove(filepath.Join(w.rootDir, "idle_prove"))
	os.Remove(filepath.Join(w.rootDir, "service_prove"))
	os.Remove(filepath.Join(w.rootDir, "podr2_rsa.pub"))
	os.Remove(filepath.Join(w.rootDir, "peer_record"))
	os.Remove(w.podr2_rsa_pub)
	os.Remove(w.idle_prove)
	os.Remove(w.service_prove)

	err = os.MkdirAll(w.fileDir, configs.FileMode)
	if err != nil {
		return err
	}

	err = os.MkdirAll(w.reportDir, configs.FileMode)
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

func (w *workspace) Build() error {
	if w.rootDir == "" {
		return fmt.Errorf("Please initialize the workspace first")
	}

	os.Remove(filepath.Join(w.rootDir, "idle_prove"))
	os.Remove(filepath.Join(w.rootDir, "service_prove"))
	os.Remove(filepath.Join(w.rootDir, "podr2_rsa.pub"))
	os.Remove(filepath.Join(w.rootDir, "peer_record"))
	w.idle_prove = filepath.Join(w.rootDir, idle_proof)
	w.service_prove = filepath.Join(w.rootDir, service_proof)

	w.logDir = filepath.Join(w.rootDir, logDir)
	if err := os.MkdirAll(w.logDir, configs.FileMode); err != nil {
		return err
	}

	w.dbDir = filepath.Join(w.rootDir, dbDir)
	if err := os.MkdirAll(w.dbDir, configs.FileMode); err != nil {
		return err
	}

	w.accDir = filepath.Join(w.rootDir, accDir)
	if err := os.MkdirAll(w.accDir, configs.FileMode); err != nil {
		return err
	}

	w.poisDir = filepath.Join(w.rootDir, poisDir)
	if err := os.MkdirAll(w.poisDir, configs.FileMode); err != nil {
		return err
	}

	w.randomDir = filepath.Join(w.rootDir, randomDir)
	if err := os.MkdirAll(w.randomDir, configs.FileMode); err != nil {
		return err
	}

	w.spaceDir = filepath.Join(w.rootDir, spaceDir)
	if err := os.MkdirAll(w.spaceDir, configs.FileMode); err != nil {
		return err
	}

	w.fileDir = filepath.Join(w.rootDir, fileDir)
	if err := os.MkdirAll(w.fileDir, configs.FileMode); err != nil {
		return err
	}

	w.reportDir = filepath.Join(w.rootDir, reportDir)
	if err := os.MkdirAll(w.reportDir, configs.FileMode); err != nil {
		return err
	}

	w.tmpDir = filepath.Join(w.rootDir, tmpDir)
	os.RemoveAll(w.tmpDir)
	if err := os.MkdirAll(w.tmpDir, configs.FileMode); err != nil {
		return err
	}
	return nil
}

func (w *workspace) GetRootDir() string {
	return w.rootDir
}
func (w *workspace) GetFileDir() string {
	return w.fileDir
}
func (w *workspace) GetReportDir() string {
	return w.reportDir
}
func (w *workspace) GetTmpDir() string {
	return w.tmpDir
}
func (w *workspace) GetDbDir() string {
	return w.dbDir
}
func (w *workspace) GetLogDir() string {
	return w.logDir
}
func (w *workspace) GetSpaceDir() string {
	return w.spaceDir
}
func (w *workspace) GetPoisDir() string {
	return w.poisDir
}
func (w *workspace) GetPoisAccDir() string {
	return w.accDir
}
func (w *workspace) GetChallRndomDir() string {
	return w.randomDir
}
func (w *workspace) GetChallRandomDir() string {
	return w.randomDir
}
func (w *workspace) GetPodr2Key() string {
	return w.podr2_rsa_pub
}
func (w *workspace) GetIdleProve() string {
	return w.idle_prove
}
func (w *workspace) GetServiceProve() string {
	return w.service_prove
}

func (w *workspace) SaveIdleProve(idleProofRecord common.IdleProofInfo) error {
	buf, err := json.Marshal(&idleProofRecord)
	if err != nil {
		return err
	}
	return sutils.WriteBufToFile(buf, w.idle_prove)
}

func (w *workspace) LoadIdleProve() (common.IdleProofInfo, error) {
	var result common.IdleProofInfo
	buf, err := os.ReadFile(w.idle_prove)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(buf, &result)
	return result, err
}

func (w *workspace) SaveServiceProve(serviceProofRecord common.ServiceProofInfo) error {
	buf, err := json.Marshal(&serviceProofRecord)
	if err != nil {
		return err
	}
	return sutils.WriteBufToFile(buf, w.service_prove)
}

func (w *workspace) LoadServiceProve() (common.ServiceProofInfo, error) {
	var result common.ServiceProofInfo
	buf, err := os.ReadFile(w.service_prove)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(buf, &result)
	return result, err
}

func (w *workspace) SaveChallRandom(
	challStart uint32,
	randomIndexList []types.U32,
	randomList []chain.Random,
) error {
	randfilePath := filepath.Join(w.GetChallRndomDir(), fmt.Sprintf("random.%d", challStart))
	fstat, err := os.Stat(randfilePath)
	if err == nil && fstat.Size() > 0 {
		return nil
	}
	var rd common.RandomList
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
