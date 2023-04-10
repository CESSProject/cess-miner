package utils

import (
	"errors"
	"fmt"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/blake2b"
)

var (
	SSPrefix            = []byte{0x53, 0x53, 0x35, 0x38, 0x50, 0x52, 0x45}
	SubstratePrefix     = []byte{0x2a}
	CESSChainTestPrefix = []byte{0x50, 0xac}
)

func DecodeToPub(address string, prefix []byte) ([]byte, error) {
	err := VerityAddress(address, prefix)
	if err != nil {
		return nil, errors.New("Invalid addrss")
	}
	data := base58.Decode(address)
	if len(data) != (34 + len(prefix)) {
		return nil, errors.New("base58 decode error")
	}
	return data[len(prefix) : len(data)-2], nil
}

func DecodeToCessPub(address string) ([]byte, error) {
	err := VerityAddress(address, CESSChainTestPrefix)
	if err != nil {
		return nil, errors.New("Invalid addrss")
	}
	data := base58.Decode(address)
	if len(data) != (34 + len(CESSChainTestPrefix)) {
		return nil, errors.New("base58 decode error")
	}
	return data[len(CESSChainTestPrefix) : len(data)-2], nil
}

func PubBytesToString(b []byte) string {
	s := ""
	for i := 0; i < len(b); i++ {
		tmp := fmt.Sprintf("%#02x", b[i])
		s += tmp[2:]
	}
	return s
}

func EncodeToSS58(publicKeyHash []byte) (string, error) {
	if len(publicKeyHash) != 32 {
		return "", errors.New("public hash length is not equal 32")
	}
	payload := appendBytes(SubstratePrefix, publicKeyHash)
	input := appendBytes(SSPrefix, payload)
	ck := blake2b.Sum512(input)
	checkum := ck[:2]
	address := base58.Encode(appendBytes(payload, checkum))
	if address == "" {
		return address, errors.New("base58 encode error")
	}
	return address, nil
}

func EncodeToCESSAddr(publicKeyHash []byte) (string, error) {
	if len(publicKeyHash) != 32 {
		return "", errors.New("public hash length is not equal 32")
	}
	payload := appendBytes(CESSChainTestPrefix, publicKeyHash)
	input := appendBytes(SSPrefix, payload)
	ck := blake2b.Sum512(input)
	checkum := ck[:2]
	address := base58.Encode(appendBytes(payload, checkum))
	if address == "" {
		return address, errors.New("base58 encode error")
	}
	return address, nil
}

func appendBytes(data1, data2 []byte) []byte {
	if data2 == nil {
		return data1
	}
	return append(data1, data2...)
}

func VerityAddress(address string, prefix []byte) error {
	decodeBytes := base58.Decode(address)
	if len(decodeBytes) != (34 + len(prefix)) {
		return errors.New("base58 decode error")
	}
	if decodeBytes[0] != prefix[0] {
		return errors.New("prefix valid error")
	}
	pub := decodeBytes[len(prefix) : len(decodeBytes)-2]

	data := append(prefix, pub...)
	input := append(SSPrefix, data...)
	ck := blake2b.Sum512(input)
	checkSum := ck[:2]
	for i := 0; i < 2; i++ {
		if checkSum[i] != decodeBytes[32+len(prefix)+i] {
			return errors.New("checksum valid error")
		}
	}
	if len(pub) != 32 {
		return errors.New("decode public key length is not equal 32")
	}
	return nil
}
