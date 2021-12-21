package proof

import (
	"fmt"
	"os"
	"path/filepath"
	"storage-mining/configs"
	"storage-mining/internal/logger"

	"github.com/filecoin-project/go-state-types/abi"
	prf "github.com/filecoin-project/specs-actors/actors/runtime/proof"
	cid "github.com/ipfs/go-cid"
	"github.com/pkg/errors"
)

//Generate Segment Porep
func GenerateSenmentVpa(sectorId SectorID, seed abi.InteractiveSealRandomness, ticket abi.SealRandomness, sealProofType abi.RegisteredSealProof) (cid.Cid, []byte, error) {
	defer func() {
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	segPath := fmt.Sprintf("%v_%v", sealProofType, sectorId.SectorNum)
	path := filepath.Join(configs.MinerDataPath, configs.SegmentData, segPath)

	_, err := os.Stat(path)
	if err == nil {
		err = os.RemoveAll(path)
		if err != nil {
			return cid.Cid{}, nil, errors.Wrapf(err, "Remove %v err", path)
		}
	}
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return cid.Cid{}, nil, errors.Wrapf(err, "Mkdir %v err", path)
	}

	cachePath := filepath.Join(path, configs.Cache)
	err = os.MkdirAll(cachePath, os.ModePerm)
	if err != nil {
		return cid.Cid{}, nil, errors.Wrapf(err, "Mkdir %v err", cachePath)
	}

	sealedCID, proof := GetPoRepForIdle(sectorId, seed, ticket, sealProofType, configs.TmpltFileName, path, cachePath)
	if (proof == nil) || (sealedCID == cid.Cid{}) {
		return cid.Cid{}, nil, errors.Wrap(err, "PoRepForIdle is nil")
	}
	return sealedCID, proof, nil
}

//Generate Segment Post
func generateSenmentVpb(sectorId SectorID, postProofType abi.RegisteredPoStProof, sealedCIDsStr []string, randomness []byte) ([]prf.PoStProof, error) {
	defer func() {
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	segPath := fmt.Sprintf("%v_%v", SealProofType, sectorId.SectorNum)
	path := filepath.Join(configs.MinerDataPath, configs.SegmentData, segPath)

	_, err := os.Stat(path)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("[%v] not found, %v", path, err)
		return nil, errors.Wrap(err, "os.Stat(path) err")
	}
	cachePath := filepath.Join(path, configs.Cache)
	_, err = os.Stat(cachePath)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("[%v] not found, %v", cachePath, err)
		return nil, errors.Wrap(err, "os.Stat(cachePath) err")
	}
	var sealedCIDs = make([]cid.Cid, 0)
	for i := 0; i < len(sealedCIDsStr); i++ {
		tmp, err := cid.Parse(sealedCIDsStr[i])
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[%v] cid parse err, %v", sealedCIDsStr[i], err)
			return nil, errors.Wrap(err, "cid.Parse err")
		}
		sealedCIDs = append(sealedCIDs, tmp)
	}

	postProof, faultySectorsl, err := GetPoSt(sectorId, postProofType, sealedCIDs, randomness, path, cachePath)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("[C%v] GetPoSt err, %v", sectorId.PeerID, err)
		return nil, errors.Wrap(err, "GetPoSt err")
	}
	if faultySectorsl != nil {
		logger.ErrLogger.Sugar().Errorf("Failed sealedcid: %v", faultySectorsl)
		return nil, errors.Wrap(err, "GetPoSt err II")
	}
	return postProof, nil
}

//Generate file Porep
func generateSegmentVpc(file, filesegpath string, segid uint64, rand []byte, uncid []string) ([]cid.Cid, [][]byte, error) {
	var err error
	defer func() {
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	cachefilepath := filepath.Join(filesegpath, configs.Cache)
	if err = os.MkdirAll(cachefilepath, os.ModePerm); err != nil {
		logger.ErrLogger.Sugar().Errorf("%v", err)
		return nil, nil, err
	}

	var unsealedCids = make([]cid.Cid, 0)
	for i := 0; i < len(uncid); i++ {
		tmp, err := cid.Parse(uncid[i])
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("Parse cid error:%v", err)
			return nil, nil, err
		}
		unsealedCids = append(unsealedCids, tmp)
	}

	secid := SectorID{
		PeerID:    abi.ActorID(configs.MinerId_I),
		SectorNum: abi.SectorNumber(segid),
	}

	sealedCIDs, proofs := GetPoRep(secid, rand, rand, abi.RegisteredSealProof(configs.FileSealProof), unsealedCids, file, filesegpath, cachefilepath)
	if sealedCIDs == nil || proofs == nil {
		return nil, nil, errors.New("file porep failed")
	}
	return sealedCIDs, proofs, nil
}

//Generate file Post
func generateSenmentVpd(sealpath, cachePath string, segid uint64, rand []byte, sealcid []string) ([]prf.PoStProof, error) {
	defer func() {
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	_, err := os.Stat(sealpath)
	if err != nil {
		return nil, err
	}
	_, err = os.Stat(cachePath)
	if err != nil {
		return nil, err
	}
	var sealedCIDs = make([]cid.Cid, 0)
	for i := 0; i < len(sealcid); i++ {
		tmp, err := cid.Parse(sealcid[i])
		if err != nil {
			return nil, err
		}
		sealedCIDs = append(sealedCIDs, tmp)
	}
	secid := SectorID{
		PeerID:    abi.ActorID(configs.MinerId_I),
		SectorNum: abi.SectorNumber(segid),
	}
	proofsWwl, faultySectorsl, err := GetPoSt(secid, abi.RegisteredPoStProof(configs.FilePostProof), sealedCIDs, rand, sealpath, cachePath)
	if err != nil {
		return nil, err
	}
	if faultySectorsl != nil {
		return nil, errors.New("gen file post failed")
	}
	return proofsWwl, nil
}
