package tools

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/blake2b"
)

var (
	SSPrefix            = []byte{0x53, 0x53, 0x35, 0x38, 0x50, 0x52, 0x45}
	PolkadotPrefix      = []byte{0x00}
	KsmPrefix           = []byte{0x02}
	KatalPrefix         = []byte{0x04}
	PlasmPrefix         = []byte{0x05}
	BifrostPrefix       = []byte{0x06}
	EdgewarePrefix      = []byte{0x07}
	KaruraPrefix        = []byte{0x08}
	ReynoldsPrefix      = []byte{0x09}
	AcalaPrefix         = []byte{0x0a}
	LaminarPrefix       = []byte{0x0b}
	PolymathPrefix      = []byte{0x0c}
	SubstraTEEPrefix    = []byte{0x0d}
	KulupuPrefix        = []byte{0x10}
	DarkPrefix          = []byte{0x11}
	DarwiniaPrefix      = []byte{0x12}
	StafiPrefix         = []byte{0x14}
	DockTestNetPrefix   = []byte{0x15}
	DockMainNetPrefix   = []byte{0x16}
	ShiftNrgPrefix      = []byte{0x17}
	SubsocialPrefix     = []byte{0x1c}
	PhalaPrefix         = []byte{0x1e}
	RobonomicsPrefix    = []byte{0x20}
	DataHighwayPrefix   = []byte{0x21}
	CentrifugePrefix    = []byte{0x24}
	MathMainPrefix      = []byte{0x27}
	MathTestPrefix      = []byte{0x28}
	SubstratePrefix     = []byte{0x2a}
	ChainXPrefix        = []byte{0x2c}
	ChainCessTestPrefix = []byte{0x50, 0xac}
)

//prefix: chain.SubstratePrefix
func EncodeByPubHex(publicHex string, prefix []byte) (string, error) {
	publicKeyHash, err := hex.DecodeString(publicHex)
	if err != nil {
		return "", err
	}
	return Encode(publicKeyHash, prefix)
}

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
	err := VerityAddress(address, ChainCessTestPrefix)
	if err != nil {
		return nil, errors.New("Invalid addrss")
	}
	data := base58.Decode(address)
	if len(data) != (34 + len(ChainCessTestPrefix)) {
		return nil, errors.New("base58 decode error")
	}
	return data[len(ChainCessTestPrefix) : len(data)-2], nil
}

func PubBytesToString(b []byte) string {
	s := ""
	for i := 0; i < len(b); i++ {
		tmp := fmt.Sprintf("%#02x", b[i])
		s += tmp[2:]
	}
	return s
}

func Encode(publicKeyHash []byte, prefix []byte) (string, error) {
	if len(publicKeyHash) != 32 {
		return "", errors.New("public hash length is not equal 32")
	}
	payload := appendBytes(prefix, publicKeyHash)
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
