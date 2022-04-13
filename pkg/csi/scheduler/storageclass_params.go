package scheduler

import "github.com/hwameistor/local-disk-manager/pkg/csi/volumemanager"

type StorageClassParams struct {
	DiskType string `json:"diskType"`
}

func parseParams(params map[string]string) *StorageClassParams {
	return &StorageClassParams{
		DiskType: params[volumemanager.VolumeParameterDiskTypeKey],
	}
}
