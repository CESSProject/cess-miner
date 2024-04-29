/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/cess-go-sdk/core/sdk"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TagfileType struct {
	Tag          *pb.Tag `protobuf:"bytes,1,opt,name=tag,proto3" json:"tag,omitempty"`
	USig         []byte  `protobuf:"bytes,2,opt,name=u_sig,json=uSig,proto3" json:"u_sig,omitempty"`
	Signature    []byte  `protobuf:"bytes,3,opt,name=signature,proto3" json:"signature,omitempty"`
	FragmentName []byte  `protobuf:"bytes,4,opt,name=fragment_name,json=fragmentName,proto3" json:"fragment_name,omitempty"`
	TeeAccountId []byte  `protobuf:"bytes,5,opt,name=tee_account_id,json=teeAccountId,proto3" json:"tee_account_id,omitempty"`
	Index        uint16  `protobuf:"bytes,6,opt,name=index,json=index,proto3" json:"index,omitempty"`
}

func CalcTag(cli sdk.SDK, cace cache.Cache, l logger.Logger, ws *Workspace, r *RunningState, teeRecord *TeeRecord, ch chan<- bool) {
	r.SetCalcTagFlag(true)
	defer func() {
		ch <- true
		r.SetCalcTagFlag(false)
		if err := recover(); err != nil {
			l.Pnc(utils.RecoverError(err))
		}
	}()

	roothashs, err := utils.Dirs(ws.GetFileDir())
	if err != nil {
		l.Stag("err", fmt.Sprintf("[Dirs(%s)] %v", ws.GetFileDir(), err))
		return
	}

	l.Stag("info", fmt.Sprintf("[roothashs] %v", roothashs))

	for _, fileDir := range roothashs {
		err = calc_tag(cli, cace, l, teeRecord, fileDir)
		if err != nil {
			l.Stag("err", fmt.Sprintf("[%s] [calc_tag] %v", filepath.Base(fileDir), roothashs))
		}
		time.Sleep(time.Minute)
	}
}

func calc_tag(cli sdk.SDK, cace cache.Cache, l logger.Logger, teeRecord *TeeRecord, file string) error {
	var ok bool
	var isReportTag bool
	var err error
	var tagPath string
	var fragments, tags []string

	fid := filepath.Base(file)
	l.Stag("info", fmt.Sprintf("[%s] Start calc file tag", fid))

	reportedFileLock.Lock()
	_, ok = reportedFile[fid]
	reportedFileLock.Unlock()
	if !ok {
		l.Stag("info", fmt.Sprintf("[%s] file not report", fid))
		return nil
	}

	ok, _ = cace.Has([]byte(Cach_prefix_Tag + fid))
	if ok {
		l.Stag("info", fmt.Sprintf("[%s] the file's tag already report", fid))
		return nil
	}

	fragments, err = getAllFragment(file)
	if err != nil {
		l.Stag("err", fmt.Sprintf("[getAllFragment(%s)] %v", fid, err))
		return nil
	}

	if len(fragments) == 0 {
		return nil
	}

	if err = checkFragmentsSize(fragments); err != nil {
		l.Stag("err", fmt.Sprintf("[checkFragmentsSize(%s)] %v", fid, err))
		return nil
	}

	for i := 0; i < len(fragments); i++ {
		tags, err = getFragmentTags(file)
		if err != nil {
			l.Stag("err", fmt.Sprintf("[getFragmentTags(%s)] %v", fid, err))
			return nil
		}

		latestSig, digest, maxIndex, err := calcRequestDigest(l, filepath.Base(fragments[i]), tags)
		if err != nil {
			l.Stag("err", fmt.Sprintf("[calcRequestDigest(%s)] %v", fid, err))
			return nil
		}
		_, err = os.Stat(fragments[i])
		if err != nil {
			l.Stag("err", fmt.Sprintf("[%s] [os.Stat(%s)] %v", fid, fragments[i], err))
			return nil
		}

		tagPath = (fragments[i] + ".tag")
		l.Stag("info", fmt.Sprintf("[%s] Check this file tag: %v", fid, tagPath))
		fstat, err := os.Stat(tagPath)
		if err == nil {
			if fstat.Size() < configs.MinTagFileSize {
				l.Stag("err", fmt.Sprintf("[%s] The file's tag size: %d < %d", fid, fstat.Size(), configs.MinTagFileSize))
				os.Remove(tagPath)
				l.Del("info", tagPath)
			} else {
				l.Stag("info", fmt.Sprintf("[%s] The file's tag already calced", fid))
				time.Sleep(time.Second)
				continue
			}
		} else {
			l.Stag("err", fmt.Sprintf("[%s] The file's tag stat err: %v", fid, err))
		}

		isreport, err := calcTheFragmentTag(l, teeRecord, cli.GetSignatureAccPulickey(), fid, fragments[i], maxIndex, latestSig, digest)
		if err != nil {
			l.Stag("err", fmt.Sprintf("[%s] [calcFragmentTag] %v", fid, err))
			return nil
		}
		l.Stag("info", fmt.Sprintf("Calc a service tag: %s", fmt.Sprintf("%s.tag", fragments[i])))
		if isreport {
			isReportTag = isreport
		}
	}

	if !isReportTag {
		fmeta, err := cli.QueryFileMetadata(fid)
		if err != nil {
			l.Stag("err", fmt.Sprintf("[%s] [QueryFileMetadata] %v", fid, err))
			return nil
		}
		for _, segment := range fmeta.SegmentList {
			for _, fragment := range segment.FragmentList {
				if sutils.CompareSlice(fragment.Miner[:], cli.GetSignatureAccPulickey()) {
					if fragment.Tag.HasValue() {
						err = cace.Put([]byte(Cach_prefix_Tag+fid), nil)
						if err != nil {
							l.Stag("err", fmt.Sprintf("[%s] [Cache.Put] %v", fid, err))
						}
						return nil
					}
					isReportTag = true
					break
				}
			}
			if isReportTag {
				break
			}
		}
	}

	if !checkAllFragmentTag(fragments) {
		l.Stag("err", fmt.Sprintf("[%s] [checkAllFragmentTag] failed", fid))
		return nil
	}

	tags, err = getFragmentTags(file)
	if err != nil {
		l.Stag("err", fmt.Sprintf("[getFragmentTags(%s)] %v", fid, err))
		return nil
	}

	txhash, err := reportFileTag(cli, l, cace, fid, tags)
	if err != nil {
		for k := 0; k < len(tags); k++ {
			os.Remove(tags[k])
			l.Del("info", tags[k])
		}
		l.Stag("err", fmt.Sprintf("[%s] [reportFileTag] %v", fid, err))
	} else {
		cace.Put([]byte(Cach_prefix_Tag+fid), nil)
		l.Stag("info", fmt.Sprintf("[%s] [reportFileTag] %v", fid, txhash))
	}
	return nil
}

func calcTheFragmentTag(l logger.Logger, teeRecord *TeeRecord, signPublicKey []byte, fid, fragmentFile string, maxIndex uint16, lastSign []byte, digest []*pb.DigestInfo) (bool, error) {
	var err error
	var isReportTag bool
	var teeSign pattern.TeeSig
	var genTag pb.GenTagMsg
	var teePubkey string
	var fragmentHash = filepath.Base(fragmentFile)

	genTag, teePubkey, err = requestTeeTag(l, teeRecord, signPublicKey, fid, fragmentFile, lastSign, digest)
	if err != nil {
		return false, fmt.Errorf("requestTeeTag: %v", err)
	}

	if len(genTag.USig) != pattern.TeeSignatureLen {
		return false, fmt.Errorf("invalid USig length: %d", len(genTag.USig))
	}

	if len(genTag.Signature) != pattern.TeeSigLen {
		return false, fmt.Errorf("invalid Tag.Signature length: %d", len(genTag.Signature))
	}
	for k := 0; k < pattern.TeeSigLen; k++ {
		teeSign[k] = types.U8(genTag.Signature[k])
	}

	var tfile = &TagfileType{
		Tag:          genTag.Tag,
		USig:         genTag.USig,
		Signature:    genTag.Signature,
		FragmentName: []byte(fragmentHash),
		TeeAccountId: []byte(teePubkey),
		Index:        (maxIndex + 1),
	}
	buf, err := json.Marshal(tfile)
	if err != nil {
		return false, fmt.Errorf("json.Marshal: %v", err)
	}

	// ok, err := n.GetPodr2Key().VerifyAttest(genTag.Tag.T.Name, genTag.Tag.T.U, genTag.Tag.PhiHash, genTag.Tag.Attest, "")
	// if err != nil {
	// 	n.Stag("err", fmt.Sprintf("[VerifyAttest] err: %s", err))
	// 	continue
	// }
	// if !ok {
	// 	n.Stag("err", "VerifyAttest is false")
	// 	continue
	// }

	err = sutils.WriteBufToFile(buf, fmt.Sprintf("%s.tag", fragmentFile))
	if err != nil {
		return false, fmt.Errorf("WriteBufToFile: %v", err)
	}
	isReportTag = true
	return isReportTag, nil
}

func requestTeeTag(l logger.Logger, teeRecord *TeeRecord, signPubkey []byte, fid, fragmentFile string, lastSign []byte, digest []*pb.DigestInfo) (pb.GenTagMsg, string, error) {
	var err error
	var teePubkey string
	var tagInfo pb.GenTagMsg
	teeEndPoints := teeRecord.GetAllMarkerTeeEndpoint()

	l.Stag("info", fmt.Sprintf("[%s] To calc the fragment tag: %v", fid, filepath.Base(fragmentFile)))
	for j := 0; j < len(teeEndPoints); j++ {
		l.Stag("info", fmt.Sprintf("[%s] Will use tee: %v", fid, teeEndPoints[j]))
		teePubkey, err = teeRecord.GetTeeWorkAccount(teeEndPoints[j])
		if err != nil {
			l.Stag("err", fmt.Sprintf("[GetTeeWorkAccount(%s)] %v", teeEndPoints[j], err))
			continue
		}
		tagInfo, err = callTeeTag(l, signPubkey, teeEndPoints[j], fid, fragmentFile, lastSign, digest)
		if err != nil {
			l.Stag("err", fmt.Sprintf("[callTeeTag(%s)] %v", teeEndPoints[j], err))
			continue
		}
		return tagInfo, teePubkey, nil
	}
	return tagInfo, teePubkey, err
}

func callTeeTag(l logger.Logger, signPubkey []byte, teeEndPoint, fid, fragmentFile string, lastSign []byte, digest []*pb.DigestInfo) (pb.GenTagMsg, error) {
	var dialOptions []grpc.DialOption
	if !strings.Contains(teeEndPoint, "443") {
		dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	} else {
		dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
	}
	conn, err := grpc.NewClient(teeEndPoint, dialOptions...)
	if err != nil {
		return pb.GenTagMsg{}, fmt.Errorf("grpc.Dial(%s): %v", teeEndPoint, err)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Minute*20))
	defer cancel()
	stream, err := pb.NewPodr2ApiClient(conn).RequestGenTag(ctx)
	if err != nil {
		return pb.GenTagMsg{}, fmt.Errorf("RequestGenTag: %v", err)
	}
	fragmentHash := filepath.Base(fragmentFile)
	buf, err := os.ReadFile(fragmentFile)
	if err != nil {
		return pb.GenTagMsg{}, fmt.Errorf("ReadFile: %v", err)
	}
	l.Stag("info", fmt.Sprintf("Will request first to %s", teeEndPoint))
	err = stream.Send(&pb.RequestGenTag{
		FragmentData:     make([]byte, 0),
		FragmentName:     fragmentHash,
		CustomData:       "",
		FileName:         fid,
		MinerId:          signPubkey,
		TeeDigestList:    make([]*pb.DigestInfo, 0),
		LastTeeSignature: make([]byte, 0)})
	if err != nil {
		return pb.GenTagMsg{}, fmt.Errorf("first send: %v", err)
	}
	l.Stag("info", fmt.Sprintf("Will recv first result from %s", teeEndPoint))
	ok, err := reciv_signal(stream)
	if err != nil {
		return pb.GenTagMsg{}, err
	}
	l.Stag("info", fmt.Sprintf("Recv first result is: %v", ok))
	if !ok {
		return pb.GenTagMsg{}, errors.New("reciv_signal: false")
	}
	l.Stag("info", fmt.Sprintf("Will request second to %s", teeEndPoint))
	err = stream.Send(&pb.RequestGenTag{
		FragmentData:     buf,
		FragmentName:     fragmentHash,
		CustomData:       "",
		FileName:         fid,
		MinerId:          signPubkey,
		TeeDigestList:    digest,
		LastTeeSignature: lastSign,
	})
	if err != nil {
		return pb.GenTagMsg{}, fmt.Errorf("second send: %v", err)
	}
	l.Stag("info", fmt.Sprintf("Will recv second result from %s", teeEndPoint))
	tag, err := reciv_tag(stream)
	if err != nil {
		return pb.GenTagMsg{}, err
	}
	l.Stag("info", "Recv second result suc")
	err = stream.CloseSend()
	if err != nil {
		l.Stag("err", fmt.Sprintf(" stream.Close: %v", err))
	}
	return tag, nil
}

func reciv_signal(stream pb.Podr2Api_RequestGenTagClient) (bool, error) {
	req, err := stream.Recv()
	if err != nil {
		return false, err
	}
	return req.GetProcessing(), nil
}
func reciv_tag(stream pb.Podr2Api_RequestGenTagClient) (pb.GenTagMsg, error) {
	req, err := stream.Recv()
	if err != nil {
		return pb.GenTagMsg{}, err
	}
	return *req.GetMsg(), nil
}

func getAllFragment(path string) ([]string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, errors.New("not dir")
	}
	files, err := utils.DirFiles(path, 0)
	if err != nil {
		return nil, err
	}
	var fragments []string
	for i := 0; i < len(files); i++ {
		if len(filepath.Base(files[i])) == pattern.FileHashLen {
			fragments = append(fragments, files[i])
		}
	}
	return fragments, nil
}

func getFragmentTags(path string) ([]string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, errors.New("not dir")
	}
	files, err := utils.DirFiles(path, 0)
	if err != nil {
		return nil, err
	}
	var tags []string
	for i := 0; i < len(files); i++ {
		if strings.Contains(files[i], ".tag") {
			tags = append(tags, files[i])
		}
	}
	return tags, nil
}

func checkFragmentsSize(fragments []string) error {
	for _, v := range fragments {
		fsata, err := os.Stat(v)
		if err != nil {
			return err
		}
		if fsata.Size() != pattern.FragmentSize {
			return errors.New("size error")
		}
	}
	return nil
}

func checkAllFragmentTag(fragments []string) bool {
	var err error
	for _, v := range fragments {
		_, err = os.Stat(v + ".tag")
		if err != nil {
			return false
		}
	}
	return true
}

func calcRequestDigest(l logger.Logger, fragment string, tags []string) ([]byte, []*pb.DigestInfo, uint16, error) {
	if len(tags) == 0 {
		return nil, nil, 0, nil
	}
	var maxIndex uint16
	var latestSig []byte
	var digest = make([]*pb.DigestInfo, len(tags))
	l.Stag("info", fmt.Sprintf("will check fragment tag: %s", fragment))
	l.Stag("info", fmt.Sprintf("can check fragment tags: %v", tags))
	for _, v := range tags {
		l.Stag("info", fmt.Sprintf("check fragment tag: %v", v))
		if strings.Contains(v, fragment) {
			continue
		}
		buf, err := os.ReadFile(v)
		if err != nil {
			return nil, nil, 0, err
		}
		var tag = &TagfileType{}
		err = json.Unmarshal(buf, tag)
		if err != nil {
			os.Remove(v)
			l.Del("info", v)
			return nil, nil, 0, err
		}
		l.Stag("info", fmt.Sprintf("tag index: %d", tag.Index))
		if tag.Index == 0 {
			msg := fmt.Sprintf("[%s] invalid tag.Index: %d ", fragment, tag.Index)
			return nil, nil, 0, errors.New(msg)
		}
		if tag.Index > maxIndex {
			maxIndex = tag.Index
			l.Stag("info", fmt.Sprintf("lastest tag sin: %d", tag.Index))
			latestSig = tag.Signature
		}
		var dig = &pb.DigestInfo{
			FragmentName: tag.FragmentName,
			TeeAccountId: tag.TeeAccountId,
		}
		digest[tag.Index-1] = dig
	}
	if len(tags) == 0 {
		digest = nil
		latestSig = nil
		maxIndex = 0
	}
	return latestSig, digest, maxIndex, nil
}

func getTagsNumber(path string) int {
	var count int
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	if !st.IsDir() {
		return 0
	}
	files, err := utils.DirFiles(path, 0)
	if err != nil {
		return 0
	}
	for i := 0; i < len(files); i++ {
		if strings.Contains(files[i], ".tag") {
			count++
		}
	}
	return count
}

func reportFileTag(cli sdk.SDK, l logger.Logger, cace cache.Cache, fid string, tags []string) (string, error) {
	var onChainFlag bool
	var err error
	var blocknumber uint32
	var txhash string
	var tagSigInfo pattern.TagSigInfo
	var latestSig []byte
	var fmeta pattern.FileMetadata
	for j := 0; j < pattern.FileHashLen; j++ {
		tagSigInfo.Filehash[j] = types.U8(fid[j])
	}
	var digest = make([]*pb.DigestInfo, len(tags))
	for _, v := range tags {
		buf, err := os.ReadFile(v)
		if err != nil {
			return txhash, err
		}
		var tag = &TagfileType{}
		err = json.Unmarshal(buf, tag)
		if err != nil {
			os.Remove(v)
			l.Del("info", v)
			return txhash, err
		}
		if tag.Index == 0 {
			msg := fmt.Sprintf("[%s] invalid tag.Index: %d ", fid, tag.Index)
			return "", errors.New(msg)
		}
		if int(tag.Index) == len(tags) {
			latestSig = tag.Signature
		}
		if int(tag.Index) > len(tags) {
			msg := fmt.Sprintf("[%s] invalid tag.Index: %d maxIndex: %d", fid, tag.Index, len(tags))
			return "", errors.New(msg)
		}
		var dig = &pb.DigestInfo{
			FragmentName: tag.FragmentName,
			TeeAccountId: tag.TeeAccountId,
		}
		digest[tag.Index-1] = dig
	}

	tagSigInfo.Miner = types.AccountID(cli.GetSignatureAccPulickey())
	tagSigInfo.Digest = make([]pattern.DigestInfo, len(digest))
	for j := 0; j < len(digest); j++ {
		tagSigInfo.Digest[j].Fragment, _ = sutils.BytesToFileHash(digest[j].FragmentName)
		tagSigInfo.Digest[j].TeePubkey, _ = sutils.BytesToWorkPublickey(digest[j].TeeAccountId)
	}
	l.Stag("info", fmt.Sprintf("[%s] Will report file tag", fid))
	for j := 0; j < 10; j++ {
		txhash, err = cli.ReportTagCalculated(latestSig, tagSigInfo)
		if err != nil || txhash == "" {
			l.Stag("err", fmt.Sprintf("[%s] ReportTagCalculated: %s %v", fid, txhash, err))
			time.Sleep(pattern.BlockInterval * 2)
			fmeta, err = cli.QueryFileMetadata(fid)
			if err == nil {
				for _, segment := range fmeta.SegmentList {
					for _, fragment := range segment.FragmentList {
						if sutils.CompareSlice(fragment.Miner[:], cli.GetSignatureAccPulickey()) {
							if fragment.Tag.HasValue() {
								ok, block := fragment.Tag.Unwrap()
								if ok {
									onChainFlag = true
									for k := 0; k < len(tags); k++ {
										fragmentHash := filepath.Base(tags[k])
										fragmentHash = strings.TrimSuffix(fragmentHash, ".tag")
										err = cace.Put([]byte(Cach_prefix_Tag+fid+"."+fragmentHash), []byte(fmt.Sprintf("%d", block)))
										if err != nil {
											l.Stag("err", fmt.Sprintf("[Cache.Put(%s, %s)] %v", Cach_prefix_Tag+fid+"."+fragmentHash, fmt.Sprintf("%d", block), err))
										} else {
											l.Stag("info", fmt.Sprintf("[Cache.Put(%s, %s)]", Cach_prefix_Tag+fid+"."+fragmentHash, fmt.Sprintf("%d", block)))
										}
									}
									break
								} else {
									onChainFlag = false
								}
							}
						}
					}
					if onChainFlag {
						break
					}
				}
			}
			if onChainFlag {
				break
			}
			l.Stag("err", err.Error())
			if (j + 1) >= 10 {
				break
			}
			time.Sleep(time.Minute)
			continue
		}
		onChainFlag = true
		l.Stag("info", fmt.Sprintf("[%s] ReportTagCalculated: %s", fid, txhash))
		blocknumber, err = cli.QueryBlockHeight(txhash)
		if err != nil {
			l.Stag("err", fmt.Sprintf("[QueryBlockHeight(%s)] %v", txhash, err))
			break
		}
		for k := 0; k < len(tags); k++ {
			fragmentHash := filepath.Base(tags[k])
			fragmentHash = strings.TrimSuffix(fragmentHash, ".tag")
			err = cace.Put([]byte(Cach_prefix_Tag+fid+"."+fragmentHash), []byte(fmt.Sprintf("%d", blocknumber)))
			if err != nil {
				l.Stag("err", fmt.Sprintf("[Cache.Put(%s, %s)] %v", Cach_prefix_Tag+fid+"."+fragmentHash, fmt.Sprintf("%d", blocknumber), err))
			} else {
				l.Stag("info", fmt.Sprintf("[Cache.Put(%s, %s)]", Cach_prefix_Tag+fid+"."+fragmentHash, fmt.Sprintf("%d", blocknumber)))
			}
		}
		break
	}
	return txhash, err
}
