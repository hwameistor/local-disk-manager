package manager

import (
	"fmt"
	"github.com/hwameistor/local-disk-manager/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/wxnacy/wgo/arrays"
	"regexp"
	"strconv"
	"strings"
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

var RaidModels = []string{"MR9361-8i"}

const (
	RaidType0 RaidType = "RAID0"

	RaidType1 RaidType = "RAID1"

	RaidType5 RaidType = "RAID5"
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
	log.Infof("ParseRaidInfo = %v, attr.Model = %v", attr, attr.Model)
	var ri RaidInfo
	if arrays.ContainsString(RaidModels, attr.Model) != -1 {
		// only deals with raid5
		dgoutput, err := utils.Bash(fmt.Sprintf("storcli  /c0 /vall show | grep %s | awk '{print $1,$2,$3,$11}'", RaidType5))

		if err != nil {
			log.Errorf("ParseRaidInfo /vall err = %v", err)
		}

		var dg, dgvd string
		for _, item := range utils.ConvertShellOutputs(dgoutput) {
			log.Infof("ParseRaidInfo ParseRAIDKeyValuePairString item = %v", item)
			props := utils.ParseRAIDKeyValuePairString(item)
			log.Infof("ParseRaidInfo ParseRAIDKeyValuePairString props = %v", props)
			if esval, ok := props["TYPE"]; ok {
				ri.RaidType = RaidType(esval)
				ri.HasRaid = true
			}
			if esval, ok := props["State"]; ok {
				ri.RaidState = RAIDState(esval)
			}
			if esval, ok := props["Name"]; ok {
				ri.RaidName = esval
			}
			if esval, ok := props["DG/VD"]; ok {
				dgvd = esval
			}
		}
		dgvds := strings.Split(dgvd, "/")
		if len(dgvds) == 2 {
			dg = dgvds[0]
		}
		escount, err := utils.Bash(fmt.Sprintf("storcli  /c0 /eall /sall show | grep %s -c", "252:"))
		if err != nil {
			log.Errorf("ParseRaidInfo  /eall /sall  err = %v", err)
		}
		re := regexp.MustCompile("[0-9]+")
		escountint, err := strconv.Atoi(re.FindString(escount))

		var rdList []RaidDisk
		for i := 0; i < escountint; i++ {
			esoutput, err := utils.Bash(fmt.Sprintf("storcli  /c0 /eall /sall show | grep %s:%s | awk '{print $1,$2,$3,$4,$5,$7,$8,$12}' ", "252", strconv.Itoa(i)))
			if err != nil {
				log.Errorf("ParseRaidInfo  /eall /sall  err = %v", err)
			}
			var rd RaidDisk
			for _, item := range utils.ConvertShellOutputs(esoutput) {
				log.Infof("ParseRaidInfo ParseRAIDDisksKeyValuePairString item = %v", item)
				props := utils.ParseRAIDDisksKeyValuePairString(item)
				log.Infof("ParseRaidInfo ParseRAIDDisksKeyValuePairString props = %v", props)
				if val, ok := props["EID:Slt"]; ok {
					kvp := strings.Split(val, ":")
					if len(kvp) == 2 {
						rd.EnclosureDeviceID = kvp[0]
						rd.SlotNo = kvp[1]
					}
				}
				if val, ok := props["DID"]; ok {
					rd.DeviceID = val
				}
				if val, ok := props["State"]; ok {
					rd.RAIDDiskState = RAIDDiskState(val)
				}
				if val, ok := props["DG"]; ok {
					if val == dg {
						rd.DriveGroup = val
					} else {
						break
					}
				}
				if val, ok := props["Med"]; ok {
					rd.MediaType = val
				}
				log.Infof("ParseRaidInfo  rd = %v", rd)
				rdList = append(rdList, rd)
			}
		}
		ri.RaidDiskList = rdList
	}
	log.Infof("ParseRaidInfo  ri = %v", ri)

	return ri
}
