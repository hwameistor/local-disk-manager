package disk

import (
	"fmt"
	ldm "github.com/hwameistor/local-disk-manager/pkg/apis/hwameistor/v1alpha1"
	"github.com/hwameistor/local-disk-manager/pkg/disk/manager"
	localdisk2 "github.com/hwameistor/local-disk-manager/pkg/handler/localdisk"
	"github.com/hwameistor/local-disk-manager/pkg/localdisk"
	"github.com/hwameistor/local-disk-manager/pkg/lsblk"
	_ "github.com/hwameistor/local-disk-manager/pkg/udev"
	"github.com/hwameistor/local-disk-manager/pkg/utils"
	apisv1alpha1 "github.com/hwameistor/reliable-helper-system/pkg/apis/hwameistor/v1alpha1"
	rdmgr "github.com/hwameistor/reliable-helper-system/pkg/replacedisk/manager"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	crmanager "sigs.k8s.io/controller-runtime/pkg/manager"
	"strings"
	"time"
)

// Controller
type Controller struct {
	// diskManager Represents how to discover and manage disks
	diskManager manager.Manager

	// diskQueue disk events queue
	diskQueue chan manager.Event

	// localDiskController
	localDiskController localdisk.Controller

	rdhandler *rdmgr.ReplaceDiskHandler

	ldhandler *localdisk2.LocalDiskHandler
}

// NewController
func NewController(mgr crmanager.Manager) *Controller {
	var recorder record.EventRecorder
	return &Controller{
		diskManager:         manager.NewManager(),
		localDiskController: localdisk.NewController(mgr),
		diskQueue:           make(chan manager.Event),
		rdhandler:           rdmgr.NewReplaceDiskHandler(mgr.GetClient(), recorder),
		ldhandler:           localdisk2.NewLocalDiskHandler(mgr.GetClient(), recorder),
	}
}

// StartMonitor
func (ctr Controller) StartMonitor() {
	// Wait cache synced
	ctr.localDiskController.Mgr.GetCache().WaitForCacheSync(make(chan struct{}))

	// Start event handler
	go ctr.HandleEvent()

	go ctr.HandleRaidEvent()

	// Start list disk exist
	for _, disk := range ctr.diskManager.ListExist() {
		ctr.Push(disk)
	}

	// Start monitor disk event
	diskEventChan := make(chan manager.Event)
	go ctr.diskManager.Monitor(diskEventChan)
	for disk := range diskEventChan {
		ctr.Push(disk)
	}
}

// HandleEvent
func (ctr Controller) HandleRaidEvent() {
	log.Debug("HandleRaidEvent ... ")
	for {
		replaceDiskList, err := ctr.rdhandler.ListReplaceDisk()
		if err != nil {
			log.WithError(err).Error("HandleRaidEvent Failed to get ReplaceDiskList")
			return
		}
		log.Debug("HandleRaidEvent replaceDiskList %v", replaceDiskList)
		for _, replacedisk := range replaceDiskList.Items {
			if replacedisk.Spec.NodeName == utils.GetNodeName() {
				if replacedisk.Spec.SltId != "" && replacedisk.Spec.EID != "" {
					var findStr = replacedisk.Spec.EID + ":" + replacedisk.Spec.SltId

					output, err := utils.Bash(fmt.Sprintf("storcli  /c0 /eall /sall  show | grep %s |awk '{print $3}'", findStr))
					log.Debug("HandleRaidEvent output = %v", output)

					if err != nil {
						log.Errorf("ParseRaidInfo  err = %v", err)
					}

					uuid := replacedisk.Spec.OldUUID
					diskName, _ := ctr.getDiskNameByDiskUUID(uuid, replacedisk.Spec.NodeName)

					oldLocalDisk, err := ctr.getLocalDiskByDiskName(diskName, replacedisk.Spec.NodeName)
					if err != nil {
						log.WithError(err).Error("HandleRaidEvent: Failed to getLocalDiskByDiskName")
						return
					}

					var newRaidDiskList []ldm.RaidDisk
					for _, raidDisk := range oldLocalDisk.Spec.RAIDInfo.RaidDiskList {
						if raidDisk.SlotNo == replacedisk.Spec.SltId && raidDisk.EnclosureDeviceID == replacedisk.Spec.EID {
							if strings.Contains(output, string(ldm.RAIDDiskStateRbld)) || strings.Contains(output, string(ldm.RAIDDiskStateMissing)) {
								oldLocalDisk.Spec.RAIDInfo.RaidState = ldm.RAIDStateDgrd
							}
							if strings.Contains(output, string(ldm.RAIDDiskStateOnln)) {
								oldLocalDisk.Spec.RAIDInfo.RaidState = ldm.RAIDStateOptl
							}
							raidDisk.RAIDDiskState = ldm.RAIDDiskState(output)
							newRaidDiskList = append(newRaidDiskList, raidDisk)
							continue
						}
						newRaidDiskList = append(newRaidDiskList, raidDisk)
					}
					oldLocalDisk.Spec.RAIDInfo.RaidDiskList = newRaidDiskList

					if err := ctr.localDiskController.UpdateLocalDisk(oldLocalDisk); err != nil {
						log.WithError(err).Errorf("Update LocalDisk fail for disk %v", oldLocalDisk)
					}
					if replacedisk.Status.NewDiskReplaceStatus == apisv1alpha1.ReplaceDisk_Succeed {
						break
					}
					time.Sleep(2 * time.Second)
				}
				break
			}
			break
		}
	}
}

func (ctr Controller) getLocalDiskByDiskName(diskName, nodeName string) (ldm.LocalDisk, error) {
	log.Debug("getLocalDiskByDiskName start ... ")
	// replacedDiskName e.g.(/dev/sdb -> sdb)
	var replacedDiskName string
	if strings.HasPrefix(diskName, "/dev") {
		replacedDiskName = strings.Replace(diskName, "/dev/", "", 1)
	}

	// ConvertNodeName e.g.(10.23.10.12 => 10-23-10-12)
	localDiskName := utils.ConvertNodeName(nodeName) + "-" + replacedDiskName
	key := client.ObjectKey{Name: localDiskName, Namespace: ""}
	localDisk, err := ctr.localDiskController.GetLocalDisk(key)
	if err != nil {
		log.WithError(err).Error("getLocalDiskByDiskName: Failed to GetLocalDisk")
		return localDisk, err
	}
	log.Debug("getLocalDiskByDiskName end ... ")
	return localDisk, nil
}

// start with /dev/sdx
func (ctr Controller) getDiskNameByDiskUUID(diskUUID, nodeName string) (string, error) {

	ldList, err := ctr.ldhandler.ListLocalDisk()
	if err != nil {
		return "", err
	}

	var diskName string
	for _, ld := range ldList.Items {
		if ld.Spec.NodeName == nodeName {
			if ld.Spec.UUID == diskUUID {
				diskName = ld.Spec.DevicePath
				break
			}
		}
	}

	return diskName, nil
}

// HandleEvent
func (ctr Controller) HandleEvent() {
	var DiskParser = defaultDiskParser()
	for {
		event := ctr.Pop()
		log.Infof("Receive disk event %+v", event)
		DiskParser.For(*manager.NewDiskIdentifyWithName(event.DevPath, event.DevName))

		switch event.Type {
		case manager.ADD:
			fallthrough
		case manager.EXIST:
			// Get disk basic info
			log.Infof("Debug test")
			newDisk := DiskParser.ParseDisk()
			log.Infof("Disk %v basicinfo: %v", event.DevPath, newDisk)

			// Convert disk resource to LocalDisk
			ld := ctr.localDiskController.ConvertDiskToLocalDisk(newDisk)

			log.Infof("Debug ConvertDiskToLocalDisk after ld = %v", ld)

			// Judge whether the disk is completely new
			if ctr.localDiskController.IsAlreadyExist(ld) {
				log.Debugf("Disk %+v has been already exist", newDisk)
				// If the disk already exists, try to update
				if err := ctr.localDiskController.UpdateLocalDisk(ld); err != nil {
					log.WithError(err).Errorf("Update LocalDisk fail for disk %v", newDisk)
				}
				continue
			}

			// Create disk resource
			if err := ctr.localDiskController.CreateLocalDisk(ld); err != nil {
				log.WithError(err).Errorf("Create LocalDisk fail for disk %v", newDisk)
				continue
			}

		default:
			log.Infof("UNKNOWN event %v, skip it", event)
		}
	}
}

// defaultDiskParser
func defaultDiskParser() *manager.DiskParser {
	diskBase := &manager.DiskIdentify{}
	return manager.NewDiskParser(
		diskBase,
		lsblk.NewPartitionParser(diskBase),
		manager.NewRaidParser(diskBase),
		lsblk.NewAttributeParser(diskBase))
}
