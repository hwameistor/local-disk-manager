package localdiskvolume

import (
	"testing"
)

func TestLocalDiskVolumeHandler_MoveMountPoint(t *testing.T) {
	//v := newEmptyVolumeHandler()
	//v.ldv = &v1alpha1.LocalDiskVolume{}
	//
	//mountPointCases := []struct {
	//	MountPath string
	//	VolumeCap *csi.VolumeCapability
	//}{
	//	{
	//		MountPath: "a/b/c",
	//		VolumeCap: &csi.VolumeCapability{
	//			AccessType: &csi.VolumeCapability_Block{},
	//		},
	//	},
	//	{
	//		MountPath: "a/b/c/d",
	//		VolumeCap: &csi.VolumeCapability{
	//			AccessType: &csi.VolumeCapability_Mount{},
	//		},
	//	},
	//}
	//
	//for _, c := range mountPointCases {
	//	v.AppendMountPoint(c.MountPath, c.VolumeCap)
	//}
	//if len(mountPointCases) != len(v.ldv.Status.MountPoints) {
	//	t.Fatalf("MountPoints Append fail, want %d actual %d",
	//		len(mountPointCases), len(v.ldv.Status.MountPoints))
	//}
	//
	//for _, c := range mountPointCases {
	//	v.MoveMountPoint(c.MountPath)
	//}
	//if len(v.ldv.Status.MountPoints) != 0 {
	//	t.Fatalf("MountPoints Move fail, want %d actual %d",
	//		0, len(v.ldv.Status.MountPoints))
	//}
}

func newEmptyVolumeHandler() *DiskVolumeHandler {
	return &DiskVolumeHandler{}
}
