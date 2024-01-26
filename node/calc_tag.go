/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
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

func (n *Node) calcTag(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	chainSt := n.GetChainState()
	if !chainSt {
		return
	}

	minerSt := n.GetMinerState()
	if minerSt != pattern.MINER_STATE_POSITIVE &&
		minerSt != pattern.MINER_STATE_FROZEN {
		return
	}

	var ok bool
	var onChainFlag bool
	var blocknumber uint32
	var txhash string
	var fid string
	var fragmentHash string
	var dialOptions []grpc.DialOption
	var teeSign pattern.TeeSig
	var tagSigInfo pattern.TagSigInfo

	n.SetCalcTagFlag(true)
	defer n.SetCalcTagFlag(false)

	roothashs, err := utils.Dirs(n.GetDirs().FileDir)
	if err != nil {
		n.Stag("err", fmt.Sprintf("[Dirs(%s)] %v", n.GetDirs().FileDir, err))
		return
	}

	teeEndPoints := n.GetPriorityTeeList()
	teeEndPoints = append(teeEndPoints, n.GetAllMarkerTeeEndpoint()...)

	for _, fileDir := range roothashs {
		fid = filepath.Base(fileDir)
		ok, err = n.Has([]byte(Cach_prefix_File + fid))
		if err == nil {
			if !ok {
				continue
			}
		} else {
			n.Report("err", err.Error())
			time.Sleep(time.Second)
			continue
		}

		fragments, tags, err := getFragmentAndTag(fileDir)
		if err != nil {
			n.Stag("err", fmt.Sprintf("[getFragmentAndTag(%s)] %v", fid, err))
			time.Sleep(time.Second)
			continue
		}

		if err = checkFragmentsSize(fragments); err != nil {
			n.Stag("err", fmt.Sprintf("[checkFragmentsSize(%s)] %v", fid, err))
			time.Sleep(time.Second)
			continue
		}

		for _, f := range fragments {
			if _, err = os.Stat(filepath.Join(f, ".tag")); err == nil {
				continue
			}
			latestSig, digest, maxIndex, err := calcRequestDigest(filepath.Base(f), tags)
			if err != nil {
				break
			}
			buf, err := os.ReadFile(f)
			if err != nil {
				break
			}
			fragmentHash = filepath.Base(f)
			var requestGenTag = &pb.RequestGenTag{
				FragmentData:     buf,
				FragmentName:     fragmentHash,
				CustomData:       "",
				FileName:         fid,
				MinerId:          n.GetSignatureAccPulickey(),
				TeeDigestList:    digest,
				LastTeeSignature: latestSig,
			}
			for i := 0; i < len(teeEndPoints); i++ {
				onChainFlag = false
				teePubkey, err := n.GetTeeWorkAccount(teeEndPoints[i])
				if err != nil {
					n.Stag("err", fmt.Sprintf("[GetTeeWorkAccount(%s)] %v", teeEndPoints[i], err))
					continue
				}
				n.Stag("info", fmt.Sprintf("[%s] Will calc file tag: %v", fid, fragmentHash))
				n.Stag("info", fmt.Sprintf("[%s] Will use tee: %v", fid, teeEndPoints[i]))
				if !strings.Contains(teeEndPoints[i], "443") {
					dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
				} else {
					dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
				}
				genTag, err := n.RequestGenTag(
					teeEndPoints[i],
					requestGenTag,
					time.Duration(time.Minute*20),
					dialOptions,
					nil,
				)
				if err != nil {
					n.Stag("err", fmt.Sprintf("[RequestGenTag] %v", err))
					continue
				}

				if len(genTag.USig) != pattern.TeeSignatureLen {
					n.Stag("err", fmt.Sprintf("[RequestGenTag] invalid USig length: %d", len(genTag.USig)))
					continue
				}

				if len(genTag.Signature) != pattern.TeeSigLen {
					n.Stag("err", fmt.Sprintf("[RequestGenTag] invalid TagSigInfo length: %d", len(genTag.Signature)))
					continue
				}
				for k := 0; k < pattern.TeeSigLen; k++ {
					teeSign[k] = types.U8(genTag.Signature[k])
				}

				var tfile = &TagfileType{
					Tag:          genTag.Tag,
					USig:         genTag.USig,
					Signature:    genTag.Signature,
					FragmentName: []byte(fragmentHash),
					TeeAccountId: []byte(string(teePubkey[:])),
					Index:        (maxIndex + 1),
				}
				buf, err = json.Marshal(tfile)
				if err != nil {
					n.Stag("err", fmt.Sprintf("[json.Marshal] err: %s", err))
					continue
				}
				ok, err := n.GetPodr2Key().VerifyAttest(genTag.Tag.T.Name, genTag.Tag.T.U, genTag.Tag.PhiHash, genTag.Tag.Attest, "")
				if err != nil {
					n.Stag("err", fmt.Sprintf("[VerifyAttest] err: %s", err))
					continue
				}
				if !ok {
					n.Stag("err", "VerifyAttest is false")
					continue
				}
				err = sutils.WriteBufToFile(buf, fmt.Sprintf("%s.tag", f))
				if err != nil {
					n.Stag("err", fmt.Sprintf("[WriteBufToFile] err: %s", err))
					continue
				}

				n.Stag("info", fmt.Sprintf("Calc a service tag: %s", fmt.Sprintf("%s.tag", f)))

				for j := 0; j < pattern.FileHashLen; j++ {
					tagSigInfo.Filehash[j] = types.U8(fid[j])
				}
				tagSigInfo.Miner = types.AccountID(n.GetSignatureAccPulickey())
				digest = append(digest, &pb.DigestInfo{
					FragmentName: []byte(fragmentHash),
					TeeAccountId: []byte(teePubkey),
				})
				tagSigInfo.Digest = make([]pattern.DigestInfo, len(digest))
				for j := 0; j < len(digest); j++ {
					tagSigInfo.Digest[j].Fragment = utils.BytesToFileHash(digest[j].FragmentName)
					tagSigInfo.Digest[j].TeePubkey = utils.BytesToWorkPublickey(digest[j].TeeAccountId)
				}
				n.Stag("info", fmt.Sprintf("Will report tag: %s.%s", fid, fragmentHash))
				var teeSignBytes = make(types.Bytes, len(teeSign))
				for j := 0; j < len(teeSign); j++ {
					teeSignBytes[j] = byte(teeSign[j])
				}
				for j := 0; j < 10; j++ {
					txhash, err = n.ReportTagCalculated(teeSignBytes, tagSigInfo)
					if err != nil || txhash == "" {
						n.Stag("err", fmt.Sprintf("ReportTagCalculated[%s.%s]: [%s] %v", fid, fragmentHash, txhash, err))
						time.Sleep(pattern.BlockInterval)
						fmeta, err := n.QueryFileMetadata(fid)
						if err == nil {
							for _, segment := range fmeta.SegmentList {
								for _, fragment := range segment.FragmentList {
									if sutils.CompareSlice(fragment.Miner[:], n.GetSignatureAccPulickey()) {
										if fragment.Tag.HasValue() {
											ok, block := fragment.Tag.Unwrap()
											if ok {
												onChainFlag = true
												err = n.Put([]byte(Cach_prefix_Tag+fid+"."+fragmentHash), []byte(fmt.Sprintf("%d", block)))
												if err != nil {
													n.Stag("err", fmt.Sprintf("[Cache.Put(%s, %s)] %v", Cach_prefix_Tag+fid+"."+fragmentHash, fmt.Sprintf("%d", block), err))
												} else {
													n.Stag("info", fmt.Sprintf("[Cache.Put(%s, %s)]", Cach_prefix_Tag+fid+"."+fragmentHash, fmt.Sprintf("%d", block)))
												}
												break
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
						n.Stag("err", err.Error())
						if (j + 1) >= 10 {
							os.Remove(fmt.Sprintf("%s.tag", f))
							break
						}
						time.Sleep(time.Minute)
						continue
					}
					onChainFlag = true
					n.Stag("info", fmt.Sprintf("ReportTagCalculated[%s.%s]: [%s]", fid, fragmentHash, txhash))
					blocknumber, err = n.QueryBlockHeight(txhash)
					if err != nil {
						n.Stag("err", fmt.Sprintf("[QueryBlockHeight(%s)] %v", txhash, err))
						break
					}
					err = n.Put([]byte(Cach_prefix_Tag+fid+"."+fragmentHash), []byte(fmt.Sprintf("%d", blocknumber)))
					if err != nil {
						n.Stag("err", fmt.Sprintf("[Cache.Put(%s, %s)] %v", Cach_prefix_Tag+fid+"."+fragmentHash, fmt.Sprintf("%d", blocknumber), err))
						break
					}
					n.Stag("info", fmt.Sprintf("Cach.Put[%s.%s]: [%d]", fid, fragmentHash, blocknumber))
					break
				}
				if onChainFlag {
					break
				}
			}
		}
	}
}

func getFragmentAndTag(path string) ([]string, []string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, nil, err
	}
	if !st.IsDir() {
		return nil, nil, errors.New("not dir")
	}
	files, err := utils.DirFiles(path, 0)
	if err != nil {
		return nil, nil, err
	}
	var fragments []string
	var tags []string
	for i := 0; i < len(files); i++ {
		if strings.Contains(files[i], ".tag") {
			tags = append(tags, files[i])
			continue
		}
		if len(filepath.Base(files[i])) == pattern.FileHashLen {
			fragments = append(fragments, files[i])
		}
	}
	return fragments, tags, nil
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

func calcRequestDigest(fragment string, tags []string) ([]byte, []*pb.DigestInfo, uint16, error) {
	if len(tags) == 0 {
		return nil, nil, 0, nil
	}
	var maxIndex uint16
	var latestSig []byte
	var digest []*pb.DigestInfo
	for _, v := range tags {
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
			return nil, nil, 0, err
		}
		if tag.Index > maxIndex {
			maxIndex = tag.Index
			latestSig = tag.Signature
		}
		var dig = &pb.DigestInfo{
			FragmentName: tag.FragmentName,
			TeeAccountId: tag.TeeAccountId,
		}
		digest = append(digest, dig)
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
