package pt

import (
	api "cess-bucket/internal/proof/apiv1"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/disk"
)

type TagInfo struct {
	T      api.FileTagT
	Sigmas [][]byte `json:"sigmas"`
}

type MountPathInfo struct {
	Path  string
	Total uint64
	Free  uint64
}

func GetMountPathInfo(mountpath string) (MountPathInfo, error) {
	var mp MountPathInfo
	pss, err := disk.Partitions(false)
	if err != nil {
		return mp, err
	}

	for _, ps := range pss {
		us, err := disk.Usage(ps.Mountpoint)
		if err != nil {
			continue
		}
		if us.Path == mountpath {
			mp.Path = us.Path
			mp.Free = us.Free
			mp.Total = us.Total
			return mp, nil
		}
	}
	return mp, errors.New("Mount path not found")
}
