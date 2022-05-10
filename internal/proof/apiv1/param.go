package proof

import (
	"github.com/Nik-U/pbc"
)

const (
	Success            = 200
	Error              = 201
	ErrorParam         = 202
	ErrorParamNotFound = 203
)

type PBCKeyPair struct {
	Spk          []byte
	Ssk          []byte
	SharedParams string
	SharedG      []byte
	Alpha        *pbc.Element
	V            *pbc.Element
	G            *pbc.Element
}

type PoDR2Commit struct {
	FilePath  string `json:"file_path"`
	BlockSize int64  `json:"block_size"`
}

type PoDR2CommitResponse struct {
	T         FileTagT       `json:"file_tag_t"`
	Sigmas    [][]byte       `json:"sigmas"`
	StatueMsg PoDR2StatueMsg `json:"statue_msg"`
}
type PoDR2StatueMsg struct {
	StatusCode int    `json:"status"`
	Msg        string `json:"msg"`
}

type PoDR2Prove struct {
	QSlice []QElement `json:"q_slice"`
	T      FileTagT   `json:"file_tag_t"`
	Sigmas [][]byte   `json:"sigmas"`
	Matrix [][]byte   `json:"matrix"`
	S      int64      `json:"s"`
}

type PoDR2ProveResponse struct {
	Sigma     []byte         `json:"sigma"`
	MU        [][]byte       `json:"mu"`
	StatueMsg PoDR2StatueMsg `json:"statue_msg"`
}

type PoDR2Verify struct {
	T      FileTagT   `json:"file_tag_t"`
	QSlice []QElement `json:"q_slice"`
	MU     [][]byte   `json:"mu"`
	Sigma  []byte     `json:"sigma"`
}
type FileTagT struct {
	T0        `json:"t0"`
	Signature []byte `json:"signature"`
}

type T0 struct {
	Name []byte   `json:"name"`
	N    int64    `json:"n"`
	U    [][]byte `json:"u"`
}

type QElement struct {
	I int64
	V []byte
}

type HashNameAndI struct {
	Name string
	I    int64
}
