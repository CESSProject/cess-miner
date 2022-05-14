package pt

import (
	api "cess-bucket/internal/proof/apiv1"
)

type TagInfo struct {
	T      api.FileTagT `json:"file_tag_t"`
	Sigmas [][]byte     `json:"sigmas"`
}
