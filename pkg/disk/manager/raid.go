package manager

import (
	log "github.com/sirupsen/logrus"
)

// IRaid
type IRaid interface {
	HasRaid() bool

	ParseRaidInfo() RaidInfo
}

// Raid
type RaidParser struct {
	// DiskIdentify Uniquely identify a disk
	*DiskIdentify

	IRaid
}

type RaidType string

const (
	RaidType0 RaidType = "Raid0"

	RaidType1 RaidType = "Raid1"

	RaidType5 RaidType = "Raid5"
)

type RAIDState string

const (
	RAIDStateDgrd RAIDState = "Dgrd"

	RAIDStateOptl RAIDState = "Optl"
)

type RAIDDiskState string

const (
	RAIDDiskStateUGood RAIDDiskState = "UGood"

	RAIDDiskStateUBad RAIDDiskState = "UBad"

	RAIDDiskStateOnln RAIDDiskState = "Onln"

	RAIDDiskStateOffln RAIDDiskState = "Offln"

	RAIDDiskStateMissing RAIDDiskState = "Missing"

	RAIDDiskStateRbld RAIDDiskState = "Rbld"
)

type RaidDisk struct {
	DriveGroup        string        `json:"driveGroup,omitempty"`
	EnclosureDeviceID string        `json:"enclosureDeviceID,omitempty"`
	SlotNo            string        `json:"slotNo,omitempty"`
	DeviceID          string        `json:"deviceID,omitempty"`
	MediaType         string        `json:"mediaType,omitempty"`
	RAIDDiskState     RAIDDiskState `json:"raidDiskState,omitempty"`
}

// RaidInfo
type RaidInfo struct {
	// HasRaid
	HasRaid   bool
	RaidName  string
	RaidType  RaidType
	RaidState RAIDState
	// PD LIST
	RaidDiskList []RaidDisk
}

// NewAttributeParser
func NewRaidParser(disk *DiskIdentify) RaidParser {
	return RaidParser{
		DiskIdentify: disk,
	}
}

func (rp RaidParser) ParseRaidInfo(attr Attribute) RaidInfo {
	var ri RaidInfo
	log.Infof("ParseRaidInfo = %v", attr.Vendor)
	if attr.Vendor == "AVAGO" {

		ri.HasRaid = true
		ri.RaidName = "raid5"
		ri.RaidState = RAIDStateOptl
		ri.RaidType = RaidType5
		var rdList []RaidDisk
		var rd1 RaidDisk
		rd1.DeviceID = "10"
		rd1.MediaType = "HDD"
		rd1.EnclosureDeviceID = "252"
		rd1.DriveGroup = "0"
		rd1.SlotNo = "1"
		rd1.RAIDDiskState = RAIDDiskStateOnln

		var rd2 RaidDisk
		rd2.DeviceID = "9"
		rd2.MediaType = "HDD"
		rd2.EnclosureDeviceID = "252"
		rd2.DriveGroup = "0"
		rd2.SlotNo = "2"
		rd2.RAIDDiskState = RAIDDiskStateOnln

		var rd3 RaidDisk
		rd3.DeviceID = "11"
		rd3.MediaType = "HDD"
		rd3.EnclosureDeviceID = "252"
		rd3.DriveGroup = "0"
		rd3.SlotNo = "3"
		rd3.RAIDDiskState = RAIDDiskStateOnln

		rdList = append(rdList, rd1)
		rdList = append(rdList, rd2)
		rdList = append(rdList, rd3)

		ri.RaidDiskList = rdList
	}
	log.Infof("ParseRaidInfo  ri = %v", ri)

	return ri
}
