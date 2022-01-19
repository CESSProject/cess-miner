package proof

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"storage-mining/configs"
	"storage-mining/internal/chain"
	"storage-mining/internal/logger"
	"storage-mining/tools"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/disk"
)

type mountpathInfo struct {
	Path  string
	Total uint64
	Free  uint64
}

func Proof_Init() {
	path := filepath.Join(configs.MinerDataPath, configs.TmpltFileFolder)
	configs.TmpltFileFolder = path
	_, err := os.Stat(configs.TmpltFileFolder)
	if err != nil {
		err = os.MkdirAll(configs.TmpltFileFolder, os.ModePerm)
		if err != nil {
			fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
			logger.ErrLogger.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
			os.Exit(configs.Exit_CreateFolder)
		}
	}

	tmpFile := filepath.Join(configs.TmpltFileFolder, configs.TmpltFileName)
	_, err = os.Create(tmpFile)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		logger.ErrLogger.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
		os.Exit(configs.Exit_CreateFile)
	}
	configs.TmpltFileName = tmpFile
	deleteFailedSegment(filepath.Join(configs.MinerDataPath, configs.SegmentData))
	spaceReasonable()
}

func Proof_Main() {
	go segmentVpa()
	go segmentVpb()
	go segmentVpc()
	go segmentVpd()
}

func segmentVpa() {
	var (
		err         error
		ok          bool
		segType     uint8
		segsizeType uint8
		segmentNum  uint32
		enableS     uint64
		segmentPath = ""
	)
	segType = 1
	segmentPath = filepath.Join(configs.MinerDataPath, configs.SegmentData)
	for range time.Tick(time.Second) {
		deleteFailedSegment(filepath.Join(configs.MinerDataPath, configs.SegmentData))
		enableS, err = getEnableSpace()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
		}
		if enableS > 0 {
			segmentNum, err = getSegmentNumForTypeOne(segmentPath, configs.SegMentType_8M_S)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
				continue
			}
			if segmentNum >= 100 {
				segsizeType = configs.SegMentType_512M
			} else {
				segsizeType = configs.SegMentType_8M
			}

			segmentId, randnum, err := chain.IntentSubmitToChain(
				configs.Confile.MinerData.IdAccountPhraseOrSeed,
				configs.ChainTx_SegmentBook_IntentSubmit,
				segsizeType,
				segType,
				configs.MinerId_I,
				nil,
				nil,
				nil,
			)
			if err != nil || randnum == 0 || segmentId == 0 {
				logger.ErrLogger.Sugar().Errorf("[%v][%v][%v]", err, segmentId, randnum)
				continue
			}
			// porepRandData, err = chain.GetSeedNumOnChain(
			// 	configs.Confile.MinerData.IdAccountPhraseOrSeed,
			// 	configs.ChainModule_SegmentBook,
			// 	configs.ChainModule_SegmentBook_ParamSetA,
			// )
			// if err != nil {
			// 	logger.ErrLogger.Sugar().Errorf("%v", err)
			// 	continue
			// }

			secid := SectorID{
				PeerID:    abi.ActorID(configs.MinerId_I),
				SectorNum: abi.SectorNumber(segmentId),
			}
			seed, err := tools.IntegerToBytes(randnum)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
				continue
			}
			var cid cid.Cid
			var prf []byte
			cid, prf, err = GenerateSenmentVpa(secid, seed, seed, abi.RegisteredSealProof(segsizeType))
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
				continue
			}
			sproof := ""
			for i := 0; i < len(prf); i++ {
				var tmp = fmt.Sprintf("%#02x", prf[i])
				sproof += tmp[2:]
			}
			ok, err = chain.SegmentSubmitToVpaOrVpb(
				configs.Confile.MinerData.IdAccountPhraseOrSeed,
				configs.ChainTx_SegmentBook_SubmitToVpa,
				configs.MinerId_I,
				uint64(segmentId),
				[]byte(sproof),
				[]byte(cid.String()),
			)
			if !ok || err != nil {
				logger.ErrLogger.Sugar().Errorf("[%v][%v][%v][%v][%v]", configs.ChainTx_SegmentBook_SubmitToVpa, segmentId, sproof, cid.String(), err)
			} else {
				logger.InfoLogger.Sugar().Infof("[%v][%v][%v][%v]", configs.ChainTx_SegmentBook_SubmitToVpa, segmentId, sproof, cid.String())
			}
		} else {
			time.Sleep(time.Minute * 10)
		}
	}
}

func segmentVpb() {
	var (
		err           error
		ok            bool
		segsizetype   uint8
		postproofType uint8
		segType       uint8
		randnum       uint32
		sealcid       string
	)
	segType = 1
	tk := time.NewTicker(time.Minute)
	for range tk.C {
		var verifiedPorepData []chain.IpostParaInfo
		verifiedPorepData, err = chain.GetVpaPostOnChain(
			configs.Confile.MinerData.IdAccountPhraseOrSeed,
			configs.ChainModule_SegmentBook,
			configs.ChainModule_SegmentBook_ConProofInfoA,
		)
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("%v", err)
			tk.Reset(time.Minute)
			continue
		} else {
			tk.Reset(time.Minute * time.Duration(configs.Vpb_SubmintPeriod))
		}
		if len(verifiedPorepData) == 0 {
			tk.Reset(time.Minute)
		}
		for i := 0; i < len(verifiedPorepData); i++ {
			sealcid = ""
			sizetypes := fmt.Sprintf("%v", verifiedPorepData[i].Size_type)
			switch sizetypes {
			case "8":
				segsizetype = 1
				postproofType = 6
			case "512":
				segsizetype = 2
				postproofType = 7
			}
			randnum, err = chain.IntentSubmitPostToChain(
				configs.Confile.MinerData.IdAccountPhraseOrSeed,
				configs.ChainTx_SegmentBook_IntentSubmitPost,
				uint64(verifiedPorepData[i].Segment_id),
				segsizetype,
				segType,
			)
			if err != nil || randnum == 0 {
				logger.ErrLogger.Sugar().Errorf("%v", err)
				continue
			}
			// postRandData, err = chain.GetSeedNumOnChain(
			// 	configs.Confile.MinerData.IdAccountPhraseOrSeed,
			// 	configs.ChainModule_SegmentBook,
			// 	configs.ChainModule_SegmentBook_ParamSetB,
			// )
			// if err != nil {
			// 	logger.ErrLogger.Sugar().Errorf("%v", err)
			// 	continue
			// }

			secid := SectorID{
				PeerID:    abi.ActorID(verifiedPorepData[i].Peer_id),
				SectorNum: abi.SectorNumber(verifiedPorepData[i].Segment_id),
			}
			seed, err := tools.IntegerToBytes(randnum)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
				continue
			}
			for j := 0; j < len(verifiedPorepData[i].Sealed_cid); j++ {
				temp := fmt.Sprintf("%c", verifiedPorepData[i].Sealed_cid[j])
				sealcid += temp
			}
			prf, err := generateSenmentVpb(secid, segsizetype, abi.RegisteredPoStProof(postproofType), []string{sealcid}, seed)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
				continue
			}
			spostproof := ""
			for j := 0; j < len(prf[0].ProofBytes); j++ {
				var tmp = fmt.Sprintf("%#02x", prf[0].ProofBytes[j])
				spostproof += tmp[2:]
			}

			ok, err = chain.SegmentSubmitToVpaOrVpb(
				configs.Confile.MinerData.IdAccountPhraseOrSeed,
				configs.ChainTx_SegmentBook_SubmitToVpb,
				uint64(verifiedPorepData[i].Peer_id),
				uint64(verifiedPorepData[i].Segment_id),
				[]byte(spostproof),
				verifiedPorepData[i].Sealed_cid,
			)
			if !ok || err != nil {
				logger.ErrLogger.Sugar().Errorf("[%v][%v][%v][%v][%v]", configs.ChainTx_SegmentBook_SubmitToVpb, verifiedPorepData[i].Segment_id, spostproof, sealcid, err)
			} else {
				logger.InfoLogger.Sugar().Infof("[%v][%v][%v][%v]", configs.ChainTx_SegmentBook_SubmitToVpb, verifiedPorepData[i].Segment_id, spostproof, sealcid)
			}
		}
	}
}

func segmentVpc() {
	var (
		err error
		ok  bool
	)
	fileSegPath := filepath.Join(configs.MinerDataPath, configs.FileData)
	tk := time.NewTicker(time.Second)
	for range tk.C {
		var unsealedcidData []chain.UnsealedCidInfo
		unsealedcidData, err = chain.GetunsealcidOnChain(
			configs.Confile.MinerData.IdAccountPhraseOrSeed,
			configs.ChainModule_SegmentBook,
			configs.ChainModule_SegmentBook_MinerHoldSlice,
		)
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("%v", err)
			time.Sleep(time.Minute)
			continue
		}
		_, err = os.Stat(fileSegPath)
		if err != nil {
			err = os.MkdirAll(fileSegPath, os.ModePerm)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
				continue
			}
		}
		if len(unsealedcidData) == 0 {
			time.Sleep(time.Minute)
		}
		for i := 0; i < len(unsealedcidData); i++ {
			hash := ""
			shardhash := ""
			uncidstring := ""
			uncid := make([]string, 0)
			for j := 0; j < len(unsealedcidData[i].Hash); j++ {
				temp := fmt.Sprintf("%c", unsealedcidData[i].Hash[j])
				hash += temp
			}
			for j := 0; j < len(unsealedcidData[i].Shardhash); j++ {
				temp := fmt.Sprintf("%c", unsealedcidData[i].Shardhash[j])
				shardhash += temp
			}
			for j := 0; j < len(unsealedcidData[i].Uncid); j++ {
				uncidstring = ""
				for k := 0; k < len(unsealedcidData[i].Uncid[j]); k++ {
					temp := fmt.Sprintf("%c", unsealedcidData[i].Uncid[j][k])
					uncidstring += temp
				}
				uncid = append(uncid, uncidstring)
			}
			seed, err := tools.IntegerToBytes(unsealedcidData[i].Rand)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
				continue
			}

			filehashid := filepath.Join(fileSegPath, fmt.Sprintf("%v", hash))
			_, err = os.Stat(filehashid)
			if err != nil {
				err = os.MkdirAll(filehashid, os.ModePerm)
				if err != nil {
					logger.ErrLogger.Sugar().Errorf("%v", err)
					continue
				}
			}
			filesegid := filepath.Join(filehashid, fmt.Sprintf("%v", unsealedcidData[i].Segment_id))
			_, err = os.Stat(filesegid)
			if err == nil {
				os.RemoveAll(filesegid)
			}
			err = os.MkdirAll(filesegid, os.ModePerm)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
				continue
			}
			filefullpath := ""
			if hash == shardhash {
				filefullpath = filepath.Join(configs.Confile.FileSystem.DfsInstallPath, "files", hash, hash+".cess")
			} else {
				filefullpath = filepath.Join(configs.Confile.FileSystem.DfsInstallPath, "files", hash, shardhash)
			}
			sealcid, prf, err := generateSegmentVpc(filefullpath, filesegid, uint64(unsealedcidData[i].Segment_id), seed, uncid)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
				continue
			}
			var sealedcid = make([]types.Bytes, len(sealcid))
			for m := 0; m < len(sealcid); m++ {
				sealedcid[m] = make(types.Bytes, 0)
				sealedcid[m] = append(sealedcid[m], types.NewBytes([]byte(sealcid[m].String()))...)
			}

			ok, err = chain.SegmentSubmitToVpc(
				configs.Confile.MinerData.IdAccountPhraseOrSeed,
				configs.ChainTx_SegmentBook_SubmitToVpc,
				uint64(unsealedcidData[i].Peer_id),
				uint64(unsealedcidData[i].Segment_id),
				prf,
				sealedcid,
			)
			if !ok || err != nil {
				logger.ErrLogger.Sugar().Errorf("[%v][%v][%v][%v][%v]", configs.ChainTx_SegmentBook_SubmitToVpc, unsealedcidData[i].Segment_id, prf, sealcid, err)
			} else {
				logger.InfoLogger.Sugar().Infof("[%v][%v][%v][%v]", configs.ChainTx_SegmentBook_SubmitToVpc, unsealedcidData[i].Segment_id, prf, sealcid)
			}
		}
	}
}

func segmentVpd() {
	var (
		err         error
		ok          bool
		segType     uint8
		segsizetype uint8
		randnum     uint32
		// postRandData chain.ParamInfo
	)
	segsizetype = 1
	segType = 2
	tk := time.NewTicker(time.Minute * time.Duration(configs.Vpd_SubmintPeriod))

	for range tk.C {
		var verifiedPorepData []chain.FpostParaInfo
		verifiedPorepData, err = chain.GetVpcPostOnChain(
			configs.Confile.MinerData.IdAccountPhraseOrSeed,
			configs.ChainModule_SegmentBook,
			configs.ChainModule_SegmentBook_ConProofInfoC,
		)
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("%v", err)
			tk.Reset(time.Minute)
			continue
		} else {
			tk.Reset(time.Minute * time.Duration(configs.Vpd_SubmintPeriod))
		}
		if len(verifiedPorepData) == 0 {
			tk.Reset(time.Minute)
		}
		for i := 0; i < len(verifiedPorepData); i++ {
			randnum, err = chain.IntentSubmitPostToChain(
				configs.Confile.MinerData.IdAccountPhraseOrSeed,
				configs.ChainTx_SegmentBook_IntentSubmitPost,
				uint64(verifiedPorepData[i].Segment_id),
				segsizetype,
				segType,
			)
			if err != nil || randnum == 0 {
				logger.ErrLogger.Sugar().Errorf("[%v][%v]", err, randnum)
				continue
			}
			// postRandData, err = chain.GetSeedNumOnChain(
			// 	configs.Confile.MinerData.IdAccountPhraseOrSeed,
			// 	configs.ChainModule_SegmentBook,
			// 	configs.ChainModule_SegmentBook_ParamSetD,
			// )
			// if err != nil {
			// 	logger.ErrLogger.Sugar().Errorf("%v", err)
			// 	continue
			// }

			sealcidstring := ""
			sealcid := make([]string, 0)
			for j := 0; j < len(verifiedPorepData[i].Sealed_cid); j++ {
				sealcidstring = ""
				for k := 0; k < len(verifiedPorepData[i].Sealed_cid[j]); k++ {
					temp := fmt.Sprintf("%c", verifiedPorepData[i].Sealed_cid[j][k])
					sealcidstring += temp
				}
				sealcid = append(sealcid, sealcidstring)
			}
			seed, err := tools.IntegerToBytes(randnum)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
				continue
			}

			fileSegPath := filepath.Join(configs.MinerDataPath, configs.FileData)
			_, err = os.Stat(fileSegPath)
			if err != nil {
				err = os.MkdirAll(fileSegPath, os.ModePerm)
				if err != nil {
					logger.ErrLogger.Sugar().Errorf("%v", err)
					continue
				}
			}
			hash := ""
			for j := 0; j < len(verifiedPorepData[i].Hash); j++ {
				temp := fmt.Sprintf("%c", verifiedPorepData[i].Hash[j])
				hash += temp
			}
			filehashid := filepath.Join(fileSegPath, fmt.Sprintf("%v", hash))
			_, err = os.Stat(filehashid)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
				continue
			}
			filesegid := filepath.Join(filehashid, fmt.Sprintf("%v", verifiedPorepData[i].Segment_id))
			_, err = os.Stat(filesegid)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
				continue
			}
			cachepath := filepath.Join(filesegid, configs.Cache)
			_, err = os.Stat(cachepath)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
				continue
			}
			postprf, err := generateSenmentVpd(filesegid, cachepath, uint64(verifiedPorepData[i].Segment_id), seed, sealcid)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
				continue
			}
			var proof = make([][]byte, len(postprf))
			for j := 0; j < len(postprf); j++ {
				proof[j] = make([]byte, 0)
				proof[j] = append(proof[j], postprf[j].ProofBytes...)
			}
			ok, err = chain.SegmentSubmitToVpd(
				configs.Confile.MinerData.IdAccountPhraseOrSeed,
				configs.ChainTx_SegmentBook_SubmitToVpd,
				uint64(verifiedPorepData[i].Peer_id),
				uint64(verifiedPorepData[i].Segment_id),
				proof,
				verifiedPorepData[i].Sealed_cid,
			)
			if !ok || err != nil {
				logger.ErrLogger.Sugar().Errorf("[%v][%v][%v][%v][%v]", configs.ChainTx_SegmentBook_SubmitToVpd, verifiedPorepData[i].Segment_id, proof, sealcid, err)
			} else {
				logger.InfoLogger.Sugar().Infof("[%v][%v][%v][%v]", configs.ChainTx_SegmentBook_SubmitToVpd, verifiedPorepData[i].Segment_id, proof, sealcid)
			}
		}
	}
}

func getMountPathInfo(mountpath string) (mountpathInfo, error) {
	var mp mountpathInfo
	pss, err := disk.Partitions(false)
	if err != nil {
		return mp, errors.Wrap(err, "disk.Partitions err")
	}

	for _, ps := range pss {
		us, err := disk.Usage(ps.Mountpoint)
		if err != nil {
			continue
		}
		if us.Total < configs.Space_1GB {
			continue
		} else {
			if us.Path == mountpath {
				mp.Path = us.Path
				mp.Free = us.Free
				mp.Total = us.Total
				return mp, nil
			}
		}
	}
	return mp, errors.New("Mount path not found or total space less than 1TB")
}

func spaceReasonable() {
	var err error
	configs.MinerUseSpace, err = tools.DirSize(configs.MinerDataPath)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
		os.Exit(configs.Exit_DirSizeError)
	}

	sspace := configs.Confile.MinerData.StorageSpace * configs.Space_1GB
	mountP, err := getMountPathInfo(configs.Confile.MinerData.MountedPath)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("%v", err)
		os.Exit(configs.Exit_CreateFile)
	}
	if mountP.Total < sspace {
		logger.ErrLogger.Sugar().Errorf("[%v] The storage space cannot be greater than the total hard disk space", configs.MinerId_S)
		os.Exit(configs.Exit_SspaceInvalid)
	}
	if (sspace + configs.Space_1GB) < configs.MinerUseSpace {
		logger.ErrLogger.Sugar().Errorf("[%v] You cannot reduce your storage space", configs.MinerId_S)
		os.Exit(configs.Exit_ReduceStorageSpace)
	}
	if sspace > configs.MinerUseSpace {
		enableSpace := sspace - configs.MinerUseSpace
		if (enableSpace > mountP.Free) || ((mountP.Free - enableSpace) < configs.Space_1GB*20) {
			logger.ErrLogger.Sugar().Errorf("[%v] Please reserve at least 20GB of space for your disk", configs.MinerId_S)
			os.Exit(configs.Exit_FreeSpaceInvalid)
		}
	}
}

func getEnableSpace() (uint64, error) {
	var err error
	configs.MinerUseSpace, err = tools.DirSize(configs.MinerDataPath)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
		return 0, err
	}

	sspace := configs.Confile.MinerData.StorageSpace * configs.Space_1GB
	mountP, err := getMountPathInfo(configs.Confile.MinerData.MountedPath)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("%v", err)
		return 0, err
	}
	enableSpace := sspace - configs.MinerUseSpace
	if (enableSpace < mountP.Free) && ((mountP.Free - enableSpace) >= configs.Space_1GB*20) {
		return enableSpace, nil
	}
	return 0, nil
}

func getSegmentNumForTypeOne(segmentpath, segtype string) (uint32, error) {
	var (
		err   error
		count uint32
	)
	_, err = os.Stat(segmentpath)
	if err != nil {
		return 0, nil
	}
	fileInfoList, err := ioutil.ReadDir(segmentpath)
	if err != nil {
		return 0, err
	}
	for i := range fileInfoList {
		if fileInfoList[i].IsDir() {
			if fileInfoList[i].Name()[:1] == segtype {
				count++
			}
		}
	}
	return count, nil
}

func deleteFailedSegment(path string) {
	var (
		err error
	)
	dirs, _ := getChildDirs(path)
	for i := 0; i < len(dirs); i++ {
		_, err = os.Stat(dirs[i] + "/tmp")
		if err == nil {
			err = os.RemoveAll(dirs[i])
			if err == nil {
				logger.InfoLogger.Sugar().Infof("Remove [%v] suc", dirs[i])
			}
		}
	}
}

func getChildDirs(filePath string) ([]string, error) {
	dirs := make([]string, 0)
	f, err := os.Stat(filePath)
	if err != nil {
		return dirs, err
	}
	if !f.IsDir() {
		return dirs, errors.New("Not a dir")
	}
	files, err := ioutil.ReadDir(filePath)
	if err != nil {
		return dirs, err
	} else {
		for _, v := range files {
			if v.IsDir() {
				path := filepath.Join(filePath, v.Name())
				dirs = append(dirs, path)
			}
		}
	}
	return dirs, nil
}
