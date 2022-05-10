package proof

import (
	"cess-bucket/configs"
	"cess-bucket/internal/chain"
	"cess-bucket/internal/encryption"
	. "cess-bucket/internal/logger"
	api "cess-bucket/internal/proof/apiv1"
	"cess-bucket/internal/rpc"
	p "cess-bucket/internal/rpc/proto"
	"cess-bucket/tools"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/disk"
	"storj.io/common/base58"
)

type mountpathInfo struct {
	Path  string
	Total uint64
	Free  uint64
}

type RespSpacetagInfo struct {
	FileId string       `json:"fileId"`
	T      api.FileTagT `json:"file_tag_t"`
	Sigmas [][]byte     `json:"sigmas"`
}

type TagInfo struct {
	T      api.FileTagT `json:"file_tag_t"`
	Sigmas [][]byte     `json:"sigmas"`
}

type RespSpacefileInfo struct {
	FileId     string `json:"fileId"`
	BlockTotal uint32 `json:"blockTotal"`
	BlockIndex uint32 `json:"blockIndex"`
	BlockData  []byte `json:"blockData"`
}

// init
func Proof_Init() {
	configs.SpaceDir = filepath.Join(configs.MinerDataPath, configs.SpaceDir)
	configs.FilesDir = filepath.Join(configs.MinerDataPath, configs.FilesDir)
	_, err := os.Stat(configs.SpaceDir)
	if err != nil {
		if err = os.MkdirAll(configs.SpaceDir, os.ModeDir); err != nil {
			fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
			Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
			os.Exit(1)
		}
	}
	_, err = os.Stat(configs.FilesDir)
	if err != nil {
		if err = os.MkdirAll(configs.FilesDir, os.ModeDir); err != nil {
			fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
			Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
			os.Exit(1)
		}
	}

	path := filepath.Join(configs.MinerDataPath, configs.TmpltFileFolder)
	configs.TmpltFileFolder = path
	_, err = os.Stat(configs.TmpltFileFolder)
	if err != nil {
		err = os.MkdirAll(configs.TmpltFileFolder, os.ModeDir)
		if err != nil {
			fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
			Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
			os.Exit(1)
		}
	}

	tmpFile := filepath.Join(configs.TmpltFileFolder, configs.TmpltFileName)
	_, err = os.Create(tmpFile)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
		os.Exit(1)
	}
	configs.TmpltFileName = tmpFile
	deleteFailedSegment(configs.SpaceDir)
	spaceReasonable()
}

// Start the proof module
func Proof_Main() {
	go segmentVpa()
	go segmentVpb()
	go segmentVpc()
	go segmentVpd()
}

// segmentVpa is used to generate the porep of idle data segments.
// it also has a space management and space synchronization mechanism.
// normally it will run forever.
func segmentVpa() {
	var (
		err         error
		segsizeType uint8
		enableS     uint64
	)
	for {
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(10, 30)))
		deleteFailedSegment(configs.SpaceDir)
		enableS, err = calcAvailableSpace()
		if err != nil {
			Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
			continue
		}
		if enableS > 0 {
			if enableS > 512*1024*1024 {
				segsizeType = configs.SegMentType_512M
			} else {
				segsizeType = configs.SegMentType_8M
			}
			segmentId, randnum, err := chain.IntentSubmitToChain(
				configs.Confile.MinerData.TransactionPrK,
				chain.ChainTx_SegmentBook_IntentSubmit,
				segsizeType,
				configs.SegMentType_Idle,
				configs.MinerId_I,
				nil,
				nil,
			)
			if err != nil || randnum == 0 || segmentId == 0 {
				Err.Sugar().Errorf("[%v][%v][%v]", err, segmentId, randnum)
				continue
			}

			secid := SectorID{
				PeerID:    abi.ActorID(configs.MinerId_I),
				SectorNum: abi.SectorNumber(segmentId),
			}
			seed, err := tools.IntegerToBytes(randnum)
			if err != nil {
				Err.Sugar().Errorf("%v", err)
				continue
			}
			// Generate proof
			cid, prf, err := GenerateSenmentVpa(secid, seed, seed, abi.RegisteredSealProof(segsizeType))
			if err != nil {
				Err.Sugar().Errorf("%v", err)
				continue
			}
			sproof := ""
			for i := 0; i < len(prf); i++ {
				var tmp = fmt.Sprintf("%#02x", prf[i])
				sproof += tmp[2:]
			}
			// put the proof on the chain
			go func(t int64, segid uint64, prf, cids string) {
				for {
					_, errs := chain.SegmentSubmitToVpaOrVpb(
						configs.Confile.MinerData.TransactionPrK,
						chain.ChainTx_SegmentBook_SubmitToVpa,
						configs.MinerId_I,
						uint64(segid),
						[]byte(prf),
						[]byte(cids),
					)
					if errs == nil {
						Out.Sugar().Infof("[%v][%v][%v][%v]", chain.ChainTx_SegmentBook_SubmitToVpa, segid, prf, cids)
						return
					}
					if time.Since(time.Unix(t, 0)).Minutes() > 10.0 {
						Err.Sugar().Errorf("[%v][%v][%v][%v][%v]", chain.ChainTx_SegmentBook_SubmitToVpa, segid, prf, cids, err)
						return
					}
					time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
				}
			}(time.Now().Unix(), segmentId, sproof, cid.String())
		} else {
			Out.Sugar().Infof("Insufficient free space on the mounted disk or the maximum storage space has been reached.")
			time.Sleep(time.Minute)
		}
	}
}

// segmentVpb is used to generate the post of idle data segments.
// normally it will run forever.
func segmentVpb() {
	var (
		err                 error
		segsizetype         uint8
		postproofType       uint8
		randnum             uint32
		sealcid             string
		verifiedPorepData   []chain.IpostParaInfo
		segDeduplicationVpb sync.Map
	)
	for {
		time.Sleep(time.Minute * time.Duration(tools.RandomInRange(5, 30)))
		verifiedPorepData, err = chain.GetVpaPostOnChain(
			configs.Confile.MinerData.TransactionPrK,
			chain.State_SegmentBook,
			chain.SegmentBook_ConProofInfoA,
		)
		if err != nil {
			Err.Sugar().Errorf("%v", err)
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(30, 60)))
			continue
		}
		effictiveDir := make([]string, 0)
		for m := 0; m < len(verifiedPorepData); m++ {
			segsizetype := ""
			sizetypes := fmt.Sprintf("%v", verifiedPorepData[m].Size_type)
			switch sizetypes {
			case "8":
				segsizetype = configs.SegMentType_8M_S
			case "512":
				segsizetype = configs.SegMentType_512M_S
			}
			dir := segsizetype + "_" + fmt.Sprintf("%d", verifiedPorepData[m].Segment_id)
			effictiveDir = append(effictiveDir, dir)
		}
		localdir, _ := tools.WalkDir(configs.SpaceDir)
		ishave := false
		for _, v1 := range localdir {
			ishave = false
			for _, v2 := range effictiveDir {
				if v1 == v2 {
					ishave = true
					break
				}
			}
			if !ishave {
				os.RemoveAll(v1)
			}
		}

		if len(verifiedPorepData) == 0 {
			continue
		}

		for i := 0; i < len(verifiedPorepData); i++ {
			if _, ok := segDeduplicationVpb.Load(uint64(verifiedPorepData[i].Segment_id)); ok {
				continue
			}
			sealcid = ""
			sizetypes := fmt.Sprintf("%v", verifiedPorepData[i].Size_type)
			switch sizetypes {
			case "8":
				segsizetype = configs.SegMentType_8M
				postproofType = configs.SegMentType_8M_post
			case "512":
				segsizetype = configs.SegMentType_512M
				postproofType = configs.SegMentType_512M_post
			}
			randnum, err = chain.IntentSubmitPostToChain(
				configs.Confile.MinerData.TransactionPrK,
				chain.ChainTx_SegmentBook_IntentSubmitPost,
				uint64(verifiedPorepData[i].Segment_id),
				segsizetype,
				configs.SegMentType_Idle,
			)
			if err != nil || randnum == 0 {
				Err.Sugar().Errorf("%v", err)
				continue
			}

			secid := SectorID{
				PeerID:    abi.ActorID(verifiedPorepData[i].Peer_id),
				SectorNum: abi.SectorNumber(verifiedPorepData[i].Segment_id),
			}
			seed, err := tools.IntegerToBytes(randnum)
			if err != nil {
				Err.Sugar().Errorf("%v", err)
				continue
			}
			for j := 0; j < len(verifiedPorepData[i].Sealed_cid); j++ {
				temp := fmt.Sprintf("%c", verifiedPorepData[i].Sealed_cid[j])
				sealcid += temp
			}
			// Generate proof
			prf, err := generateSenmentVpb(secid, segsizetype, abi.RegisteredPoStProof(postproofType), []string{sealcid}, seed)
			if err != nil {
				Err.Sugar().Errorf("%v", err)
				continue
			}
			spostproof := ""
			for j := 0; j < len(prf[0].ProofBytes); j++ {
				var tmp = fmt.Sprintf("%#02x", prf[0].ProofBytes[j])
				spostproof += tmp[2:]
			}
			// put the proof on the chain
			go func(t int64, peerid, segid uint64, sprf string, cids types.Bytes) {
				segDeduplicationVpb.Store(segid, true)
				defer segDeduplicationVpb.Delete(segid)
				for {
					_, errs := chain.SegmentSubmitToVpaOrVpb(
						configs.Confile.MinerData.TransactionPrK,
						chain.ChainTx_SegmentBook_SubmitToVpb,
						peerid,
						segid,
						[]byte(sprf),
						cids,
					)
					if errs == nil {
						Out.Sugar().Infof("[%v][%v][%v]", chain.ChainTx_SegmentBook_SubmitToVpb, segid, sprf)
						return
					}
					if time.Since(time.Unix(t, 0)).Minutes() > 10.0 {
						Err.Sugar().Errorf("[%v][%v][%v][%v]", chain.ChainTx_SegmentBook_SubmitToVpb, segid, sprf, err)
						return
					}
					time.Sleep(time.Second * time.Duration(tools.RandomInRange(10, 20)))
				}
			}(time.Now().Unix(), uint64(verifiedPorepData[i].Peer_id), uint64(verifiedPorepData[i].Segment_id), spostproof, verifiedPorepData[i].Sealed_cid)
		}
	}
}

// segmentVpc is used to generate porep for service data segments.
// normally it will run forever.
func segmentVpc() {
	var (
		err                 error
		unsealedcidData     []chain.UnsealedCidInfo
		segDeduplicationVpc sync.Map
	)
	for {
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(10, 30)))
		unsealedcidData, err = chain.GetunsealcidOnChain(
			configs.Confile.MinerData.TransactionPrK,
			chain.State_SegmentBook,
			chain.SegmentBook_MinerHoldSlice,
		)
		if err != nil {
			Err.Sugar().Errorf("%v", err)
			continue
		}
		_, err = os.Stat(configs.FilesDir)
		if err != nil {
			err = os.MkdirAll(configs.FilesDir, os.ModeDir)
			if err != nil {
				Err.Sugar().Errorf("%v", err)
				continue
			}
		}
		if len(unsealedcidData) == 0 {
			continue
		}
		for i := 0; i < len(unsealedcidData); i++ {
			if _, ok := segDeduplicationVpc.Load(uint64(unsealedcidData[i].Segment_id)); ok {
				continue
			}
			shardhash := ""
			uncidstring := ""
			uncid := make([]string, 0)
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
				Err.Sugar().Errorf("%v", err)
				continue
			}
			fid := strings.Split(shardhash, ".")[0]

			filehashid := filepath.Join(configs.FilesDir, fid)
			_, err = os.Stat(filehashid)
			if err != nil {
				err = os.MkdirAll(filehashid, os.ModeDir)
				if err != nil {
					Err.Sugar().Errorf("%v", err)
					continue
				}
			}
			filesegid := filepath.Join(filehashid, fmt.Sprintf("%v", unsealedcidData[i].Segment_id))
			_, err = os.Stat(filesegid)
			if err == nil {
				os.RemoveAll(filesegid)
			}
			err = os.MkdirAll(filesegid, os.ModeDir)
			if err != nil {
				Err.Sugar().Errorf("%v", err)
				continue
			}
			filefullpath := filepath.Join(configs.FilesDir, fid, shardhash)
			// Generate proof
			sealcid, prf, err := generateSegmentVpc(filefullpath, filesegid, uint64(unsealedcidData[i].Segment_id), seed, uncid)
			if err != nil {
				Err.Sugar().Errorf("%v", err)
				continue
			}
			var sealedcid = make([]types.Bytes, len(sealcid))
			for m := 0; m < len(sealcid); m++ {
				sealedcid[m] = make(types.Bytes, 0)
				sealedcid[m] = append(sealedcid[m], types.NewBytes([]byte(sealcid[m].String()))...)
			}
			// put the proof on the chain
			go func(t int64, peerid, segid uint64, proof [][]byte, cids []types.Bytes, fileid string) {
				segDeduplicationVpc.Store(segid, true)
				defer segDeduplicationVpc.Delete(segid)
				for {
					_, errs := chain.SegmentSubmitToVpc(
						configs.Confile.MinerData.TransactionPrK,
						chain.ChainTx_SegmentBook_SubmitToVpc,
						peerid,
						segid,
						proof,
						cids,
						types.Bytes([]byte(fileid)),
					)
					if errs == nil {
						Out.Sugar().Infof("[%v][%v][%v]", chain.ChainTx_SegmentBook_SubmitToVpc, peerid, segid)
						return
					}
					if time.Since(time.Unix(t, 0)).Minutes() > 10.0 {
						Err.Sugar().Errorf("[%v][%v][%v][%v]", chain.ChainTx_SegmentBook_SubmitToVpc, peerid, segid, err)
						return
					}
					time.Sleep(time.Second * time.Duration(tools.RandomInRange(10, 20)))
				}
			}(time.Now().Unix(), uint64(unsealedcidData[i].Peer_id), uint64(unsealedcidData[i].Segment_id), prf, sealedcid, fid)
		}
	}
}

// segmentVpd is used to generate post for service data segments.
// normally it will run forever.
func segmentVpd() {
	var (
		err                 error
		randnum             uint32
		verifiedPorepData   []chain.FpostParaInfo
		segDeduplicationVpd sync.Map
	)
	for {
		time.Sleep(time.Minute * time.Duration(tools.RandomInRange(1, 5)))
		verifiedPorepData, err = chain.GetVpcPostOnChain(
			configs.Confile.MinerData.TransactionPrK,
			chain.State_SegmentBook,
			chain.SegmentBook_ConProofInfoC,
		)
		if err != nil {
			Err.Sugar().Errorf("%v", err)
			continue
		}
		if len(verifiedPorepData) == 0 {
			continue
		}
		for i := 0; i < len(verifiedPorepData); i++ {
			if _, ok := segDeduplicationVpd.Load(uint64(verifiedPorepData[i].Segment_id)); ok {
				continue
			}
			randnum, err = chain.IntentSubmitPostToChain(
				configs.Confile.MinerData.TransactionPrK,
				chain.ChainTx_SegmentBook_IntentSubmitPost,
				uint64(verifiedPorepData[i].Segment_id),
				configs.SegMentType_8M,
				configs.SegMentType_Service,
			)
			if err != nil || randnum == 0 {
				Err.Sugar().Errorf("[%v][%v]", err, randnum)
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(30, 60)))
				continue
			}

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
				Err.Sugar().Errorf("%v", err)
				continue
			}

			hash := ""
			for j := 0; j < len(verifiedPorepData[i].Hash); j++ {
				temp := fmt.Sprintf("%c", verifiedPorepData[i].Hash[j])
				hash += temp
			}
			fid := strings.Split(hash, ".")[0]
			filehashid := filepath.Join(configs.FilesDir, fid)
			_, err = os.Stat(filehashid)
			if err != nil {
				Err.Sugar().Errorf("%v", err)
				continue
			}
			filesegid := filepath.Join(filehashid, fmt.Sprintf("%v", verifiedPorepData[i].Segment_id))
			_, err = os.Stat(filesegid)
			if err != nil {
				Err.Sugar().Errorf("%v", err)
				continue
			}
			cachepath := filepath.Join(filesegid, configs.Cache)
			_, err = os.Stat(cachepath)
			if err != nil {
				Err.Sugar().Errorf("%v", err)
				continue
			}
			// Generate proof
			postprf, err := generateSenmentVpd(filesegid, cachepath, uint64(verifiedPorepData[i].Segment_id), seed, sealcid)
			if err != nil {
				Err.Sugar().Errorf("%v", err)
				continue
			}
			var proof = make([][]byte, len(postprf))
			for j := 0; j < len(postprf); j++ {
				proof[j] = make([]byte, 0)
				proof[j] = append(proof[j], postprf[j].ProofBytes...)
			}
			// put the proof on the chain
			go func(t int64, peerid, segid uint64, prf [][]byte, cids []types.Bytes, fileid string) {
				segDeduplicationVpd.Store(segid, true)
				defer segDeduplicationVpd.Delete(segid)
				for {
					_, errs := chain.SegmentSubmitToVpd(
						configs.Confile.MinerData.TransactionPrK,
						chain.ChainTx_SegmentBook_SubmitToVpd,
						peerid,
						segid,
						prf,
						cids,
						types.Bytes([]byte(fileid)),
					)
					if errs == nil {
						Out.Sugar().Infof("[%v][%v][%v]", chain.ChainTx_SegmentBook_SubmitToVpd, peerid, segid)
						return
					}
					if time.Since(time.Unix(t, 0)).Minutes() > 10.0 {
						Err.Sugar().Errorf("[%v][%v][%v][%v]", chain.ChainTx_SegmentBook_SubmitToVpd, peerid, segid, err)
						return
					}
					time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 20)))
				}
			}(time.Now().Unix(), uint64(verifiedPorepData[i].Peer_id), uint64(verifiedPorepData[i].Segment_id), proof, verifiedPorepData[i].Sealed_cid, fid)
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
		Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
		os.Exit(1)
	}

	sspace := configs.Confile.MinerData.StorageSpace * configs.Space_1GB
	mountP, err := getMountPathInfo(configs.Confile.MinerData.MountedPath)
	if err != nil {
		Err.Sugar().Errorf("%v", err)
		os.Exit(1)
	}
	if mountP.Total < sspace {
		Err.Sugar().Errorf("[%v] The storage space cannot be greater than the total hard disk space", configs.MinerId_S)
		os.Exit(1)
	}
	if (sspace + configs.Space_1GB) < configs.MinerUseSpace {
		Err.Sugar().Errorf("[%v] You cannot reduce your storage space", configs.MinerId_S)
		os.Exit(1)
	}
	if sspace > configs.MinerUseSpace {
		enableSpace := sspace - configs.MinerUseSpace
		if (enableSpace > mountP.Free) || ((mountP.Free - enableSpace) < configs.Space_1GB*20) {
			Err.Sugar().Errorf("[%v] Please reserve at least 20GB of space for your disk", configs.MinerId_S)
			os.Exit(1)
		}
	}
}

// Calculate available space
func calcAvailableSpace() (uint64, error) {
	var err error
	configs.MinerUseSpace, err = tools.DirSize(configs.MinerDataPath)
	if err != nil {
		return 0, err
	}
	sspace := configs.Confile.MinerData.StorageSpace * configs.Space_1GB
	mountP, err := getMountPathInfo(configs.Confile.MinerData.MountedPath)
	if err != nil {
		return 0, err
	}
	if sspace <= configs.MinerUseSpace {
		return 0, nil
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

// Delete failed data segment
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
				Err.Sugar().Infof("Remove [%v] suc", dirs[i])
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

//
func processingSpace() {
	var (
		err     error
		count   uint8
		enableS uint64
		req     p.SpaceTagReq
		addr    string
	)
	addr, err = chain.GetAddressFromPrk(configs.Confile.MinerData.TransactionPrK)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
	prk, err := encryption.GetRSAPrivateKey(filepath.Join(configs.MinerDataPath, configs.PrivateKeyfile))
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
	sign, err := encryption.CalcSign([]byte(addr), prk)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	schds, _ := chain.GetSchedulerInfo()

	for {
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
		//deleteFailedSegment(configs.SpaceDir)
		count++
		if count%100 == 0 {
			count = 0
			schds, _ = chain.GetSchedulerInfo()
		}
		if len(schds) == 0 {
			continue
		}
		enableS, err = calcAvailableSpace()
		if err != nil {
			Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
			continue
		}
		if enableS > uint64(8*configs.Space_1MB) {
			req.SizeMb = 8
			req.WalletAddress = addr
			req.Fileid = ""
			req.BlockIndex = 0
			req.Sign = sign

			req_b, err := proto.Marshal(&req)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}

			var client *rpc.Client
			for i, schd := range schds {
				wsURL := "ws://" + string(base58.Decode(string(schd.Ip)))
				ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
				client, err = rpc.DialWebsocket(ctx, wsURL, "")
				if err != nil {
					Err.Sugar().Errorf("[%v] %v", wsURL, err)
					if (i + 1) == len(schds) {
						Err.Sugar().Errorf("All scheduler not working")
					}
				} else {
					break
				}
			}
			if err != nil {
				continue
			}
			resp, err := rpc.WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_Space, req_b)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			var respInfo RespSpacetagInfo
			err = json.Unmarshal(resp, &respInfo)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			var tagInfo TagInfo
			tagfilepath := filepath.Join(configs.SpaceDir, respInfo.FileId)
			_, err = os.Stat(tagfilepath)
			if err != nil {
				err = os.MkdirAll(tagfilepath, os.ModeDir)
				if err != nil {
					Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
					continue
				}
			}
			tagfilename := respInfo.FileId + ".tag"
			tagfilefullpath := filepath.Join(tagfilepath, tagfilename)
			tagInfo.T = respInfo.T
			tagInfo.Sigmas = respInfo.Sigmas
			ft, err := os.OpenFile(tagfilefullpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			tag, err := json.Marshal(tagInfo)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				ft.Close()
				continue
			}
			ft.Write(tag)
			err = ft.Sync()
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				ft.Close()
				continue
			}
			ft.Close()

			// req file
			req.SizeMb = 0
			req.WalletAddress = addr
			req.Fileid = respInfo.FileId
			req.BlockIndex = 0
			req.Sign = sign

			req_b, err = proto.Marshal(&req)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			resp, err = rpc.WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_Space, req_b)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			var respspacefile RespSpacefileInfo
			err = json.Unmarshal(resp, &respspacefile)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			spacefilename := respInfo.FileId + ".space"
			spacefilefullpath := filepath.Join(tagfilepath, spacefilename)
			spacefile, err := os.OpenFile(spacefilefullpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_APPEND, os.ModePerm)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			spacefile.Write(respspacefile.BlockData)
			for j := 1; j < int(respspacefile.BlockTotal); j++ {
				req.BlockIndex = uint32(j)
				req_b, err = proto.Marshal(&req)
				if err != nil {
					Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
					spacefile.Close()
					os.Remove(spacefilefullpath)
					break
				}
				respi, err := rpc.WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_Space, req_b)
				if err != nil {
					Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
					spacefile.Close()
					os.Remove(spacefilefullpath)
					break
				}
				var respspacefilei RespSpacefileInfo
				err = json.Unmarshal(respi, &respspacefilei)
				if err != nil {
					Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
					spacefile.Close()
					os.Remove(spacefilefullpath)
					break
				}
				spacefile.Write(respspacefilei.BlockData)
			}
			_, err = os.Stat(spacefilefullpath)
			if err != nil {
				continue
			}
			err = spacefile.Sync()
			if err != nil {
				spacefile.Close()
				os.Remove(spacefilefullpath)
				continue
			}
			spacefile.Close()
		}
	}
}

//
func processingChallenges() {
	var (
		err         error
		filedir     string
		filename    string
		tagfilename string
		chlng       []chain.ChallengesInfo
	)
	for {
		time.Sleep(time.Minute * time.Duration(tools.RandomInRange(1, 5)))
		chlng, err = chain.GetChallengesById(configs.MinerId_I)
		if err != nil {
			continue
		}
		for i := 0; i < len(chlng); i++ {
			if chlng[i].File_type == 1 {
				filedir = filepath.Join(configs.SpaceDir, string(chlng[i].File_id))
				filename = string(chlng[i].File_id) + ".space"
			} else {
				filedir = filepath.Join(configs.FilesDir, string(chlng[i].File_id))
				filename = string(chlng[i].File_id) + ".cess"
			}
			tagfilename = string(chlng[i].File_id) + ".tag"
			_, err = os.Stat(filepath.Join(filedir, filename))
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", filedir, err)
				continue
			}
			tmp := make(map[int]*big.Int, len(chlng[i].Block_list))
			for j := 0; j < len(chlng[i].Block_list); j++ {
				tmp[int(chlng[i].Block_list[j])] = new(big.Int).SetBytes(chlng[i].Random[j])
			}
			//TODO: query SharedParams from chain
			qSlice, err := api.PoDR2ChallengeGenerateFromChain(tmp, "")
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", filedir, err)
				continue
			}
			_ = qSlice
			_ = tagfilename
		}
	}
}
