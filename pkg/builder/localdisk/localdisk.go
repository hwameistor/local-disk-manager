package localdisk

import (
	"fmt"
	log "github.com/sirupsen/logrus"

	"github.com/hwameistor/local-disk-manager/pkg/apis/hwameistor/v1alpha1"
	"github.com/hwameistor/local-disk-manager/pkg/disk/manager"
)

// Builder for LocalDisk resource
type Builder struct {
	disk *v1alpha1.LocalDisk
	errs []error
}

func NewBuilder() *Builder {
	return &Builder{
		disk: &v1alpha1.LocalDisk{},
	}
}

func (builder *Builder) WithName(name string) *Builder {
	if builder.errs != nil {
		return builder
	}
	builder.disk.Name = name
	return builder
}

func (builder *Builder) SetupAttribute(attribute manager.Attribute) *Builder {
	if builder.errs != nil {
		return builder
	}
	builder.disk.Spec.Capacity = attribute.Capacity
	builder.disk.Spec.DevicePath = attribute.DevName
	builder.disk.Spec.DiskAttributes.Type = attribute.DriverType
	builder.disk.Spec.DiskAttributes.Vendor = attribute.Vendor
	builder.disk.Spec.DiskAttributes.ModelName = attribute.Model
	builder.disk.Spec.DiskAttributes.Protocol = attribute.Bus
	builder.disk.Spec.DiskAttributes.SerialNumber = attribute.Serial
	builder.disk.Spec.DiskAttributes.DevType = attribute.DevType

	return builder
}

func (builder *Builder) SetupRaidInfo(raid manager.RaidInfo) *Builder {
	log.Infof("debug builder.errs = %v", builder.errs)

	if builder.errs != nil {
		return builder
	}

	log.Infof("debug SetupRaidInfo = %v", raid)

	builder.disk.Spec.HasRAID = raid.HasRaid
	builder.disk.Spec.RAIDInfo.RaidType = v1alpha1.RaidType(raid.RaidType)
	builder.disk.Spec.RAIDInfo.RaidState = v1alpha1.RAIDState(raid.RaidState)
	builder.disk.Spec.RAIDInfo.RaidName = raid.RaidName

	log.Infof("debug builder.disk.Spec.RAIDInfo 1= %v", builder.disk.Spec.RAIDInfo)

	var rdList []v1alpha1.RaidDisk
	for _, raidDisk := range raid.RaidDiskList {
		var rd v1alpha1.RaidDisk
		rd.RAIDDiskState = v1alpha1.RAIDDiskState(raidDisk.RAIDDiskState)
		rd.DriveGroup = raidDisk.DriveGroup
		rd.SlotNo = raidDisk.SlotNo
		rd.DeviceID = raidDisk.DeviceID
		rd.MediaType = raidDisk.MediaType
		rd.EnclosureDeviceID = raidDisk.EnclosureDeviceID
		rdList = append(rdList, rd)
	}

	builder.disk.Spec.RAIDInfo.RaidDiskList = rdList

	log.Infof("debug builder.disk.Spec.RAIDInfo 2= %v", builder.disk.Spec.RAIDInfo)

	// complete RAID INFO here
	return builder
}

func (builder *Builder) SetupUUID(uuid string) *Builder {
	if builder.errs != nil {
		return builder
	}

	builder.disk.Spec.UUID = uuid
	return builder
}

func (builder *Builder) SetupNodeName(node string) *Builder {
	if builder.errs != nil {
		return builder
	}

	builder.disk.Spec.NodeName = node
	return builder
}

func (builder *Builder) SetupPartitionInfo(originParts []manager.PartitionInfo) *Builder {
	if builder.errs != nil {
		return builder
	}
	for _, part := range originParts {
		builder.disk.Spec.HasPartition = true
		p := v1alpha1.PartitionInfo{}
		p.HasFileSystem = true
		p.FileSystem.Type = part.Filesystem
		builder.disk.Spec.PartitionInfo = append(builder.disk.Spec.PartitionInfo, p)
	}
	return builder
}

func (builder *Builder) GenerateStatus() *Builder {
	if builder.errs != nil {
		return builder
	}
	if builder.disk.Spec.HasPartition {
		builder.disk.Status.State = v1alpha1.LocalDiskInUse
	} else {
		builder.disk.Status.State = v1alpha1.LocalDiskUnclaimed
	}
	return builder
}

func (builder *Builder) Build() (v1alpha1.LocalDisk, error) {
	if builder.errs != nil {
		return v1alpha1.LocalDisk{}, fmt.Errorf("%v", builder.errs)
	}

	return *builder.disk, nil
}
