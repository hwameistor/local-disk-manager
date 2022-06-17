package diskmanager

import (
	"fmt"
	"reflect"
	"sync"

	localdisk2 "github.com/hwameistor/local-disk-manager/pkg/handler/localdisk"

	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/hwameistor/local-disk-manager/pkg/utils"
	log "github.com/sirupsen/logrus"

	"github.com/hwameistor/local-disk-manager/pkg/apis/hwameistor/v1alpha1"
	"github.com/hwameistor/local-disk-manager/pkg/builder/localdisknode"
	"github.com/hwameistor/local-disk-manager/pkg/utils/kubernetes"
)

var (
	once sync.Once
	ldn  *LocalDiskNodesManager
)

const (
	ReservedPVCKey = "disk.hwameistor.io/pvc"
)

// LocalDiskNodesManager manage all disks in the cluster by interacting with LocalDisk resources
type LocalDiskNodesManager struct {
	// GetClient for query LocalDiskNode resources from k8s
	GetClient func() (*localdisknode.Kubeclient, error)

	// distributed lock or mutex lock(controller already has distributed lock )
	mutex sync.Mutex

	// DiskHandler manage LD resources in cluster
	DiskHandler *localdisk2.LocalDiskHandler
}

func (ldn *LocalDiskNodesManager) ReleaseDisk(disk string) error {
	if disk == "" {
		log.Debug("ReleaseDisk skipped due to disk needs to release is empty")
		return nil
	}
	ld, err := ldn.DiskHandler.GetLocalDisk(client.ObjectKey{Name: disk})
	if err != nil {
		return err
	}
	ldn.DiskHandler.For(*ld)
	ldn.DiskHandler.RemoveLabel(labels.Set{ReservedPVCKey: ""})
	ldn.DiskHandler.SetupStatus(v1alpha1.LocalDiskReleased)
	return ldn.DiskHandler.UpdateStatus()
}

func (ldn *LocalDiskNodesManager) UnReserveDiskForPVC(pvc string) error {
	label := labels.Set{ReservedPVCKey: pvc}
	list, err := ldn.DiskHandler.GetLocalDiskWithLabels(label)
	if err != nil {
		return err
	}

	for _, disk := range list.Items {
		if disk.Status.State != v1alpha1.LocalDiskReserved {
			continue
		}
		ldn.DiskHandler.For(disk)
		ldn.DiskHandler.RemoveLabel(label)
		ldn.DiskHandler.SetupStatus(v1alpha1.LocalDiskReleased)
		if err = ldn.DiskHandler.UpdateStatus(); err != nil {
			return err
		}
	}

	return err
}

func NewLocalDiskManager() *LocalDiskNodesManager {
	once.Do(func() {
		ldn = &LocalDiskNodesManager{}
		ldn.GetClient = localdisknode.NewKubeclient
		cli, _ := kubernetes.NewClient()
		recoder, _ := kubernetes.NewRecorderFor("localdisknodemanager")
		ldn.DiskHandler = localdisk2.NewLocalDiskHandler(cli, recoder)
	})

	return ldn
}

// GetClusterDisks
// Here is just a simple implementation
func (ldn *LocalDiskNodesManager) GetClusterDisks() (map[string][]*Disk, error) {
	cli, err := ldn.GetClient()
	if err != nil {
		return nil, err
	}

	// fixme: should do more check
	var clusterDisks = make(map[string][]*Disk)

	var nodes *v1alpha1.LocalDiskNodeList

	nodes, err = cli.List()
	if err != nil {
		return nil, err
	}

	for _, node := range nodes.Items {
		var nodeDisks []*Disk
		for name, disk := range node.Status.Disks {
			nodeDisks = append(nodeDisks, convertToDisk(node.Spec.AttachNode, name, disk))
		}

		clusterDisks[node.Spec.AttachNode] = nodeDisks
	}

	return clusterDisks, nil
}

// GetNodeDisks get disks which attached on the node
func (ldn *LocalDiskNodesManager) GetNodeDisks(node string) ([]*Disk, error) {
	cli, err := ldn.GetClient()
	if err != nil {
		return nil, err
	}

	var diskNode *v1alpha1.LocalDiskNode
	diskNode, err = cli.Get(node)
	if err != nil {
		return nil, err
	}

	var nodeDisks []*Disk
	for name, disk := range diskNode.Status.Disks {
		nodeDisks = append(nodeDisks, convertToDisk(node, name, disk))
	}

	return nodeDisks, nil
}

func (ldn *LocalDiskNodesManager) filterDisk(reqDisk, existDisk Disk) bool {
	if !(existDisk.Status == DiskStatusUnclaimed ||
		existDisk.Status == DiskStatusReleased) {
		return false
	}
	if existDisk.DiskType == reqDisk.DiskType &&
		existDisk.Capacity >= reqDisk.Capacity {
		return true
	}
	return false
}

func (ldn *LocalDiskNodesManager) diskScoreMax(reqDisk Disk, existDisks []*Disk) *Disk {
	if len(existDisks) == 0 {
		return nil
	}

	selDisk := existDisks[0]
	for _, existDisk := range existDisks {
		if existDisk.Capacity < selDisk.Capacity {
			selDisk = existDisk
		}
	}

	return selDisk
}

func (ldn *LocalDiskNodesManager) GetReservedDiskByPVC(pvc string) (*Disk, error) {
	list, err := ldn.DiskHandler.GetLocalDiskWithLabels(labels.Set{ReservedPVCKey: pvc})
	if err != nil {
		return nil, err
	}

	if len(list.Items) == 0 {
		return nil, fmt.Errorf("there is no disk reserved by pvc %s", pvc)
	}

	reservedDisk := list.Items[0]
	return &Disk{
		AttachNode: reservedDisk.Spec.NodeName,
		Name:       reservedDisk.Name,
		DevPath:    reservedDisk.Spec.DevicePath,
		Capacity:   reservedDisk.Spec.Capacity,
		DiskType:   reservedDisk.Spec.DiskAttributes.Type,
	}, nil
}

// ClaimDisk claim a LocalDisk by update LocalDisk status to InUse
func (ldn *LocalDiskNodesManager) ClaimDisk(name string) error {
	if name == "" {
		return fmt.Errorf("disk is empty")
	}

	ld, err := ldn.DiskHandler.GetLocalDisk(client.ObjectKey{Name: name})
	if err != nil {
		log.Errorf("failed to get LocalDisk %s", err.Error())
		return err
	}
	ldn.DiskHandler.For(*ld)
	ldn.DiskHandler.SetupStatus(v1alpha1.LocalDiskInUse)

	return ldn.DiskHandler.UpdateStatus()
}

func (ldn *LocalDiskNodesManager) reserve(disk *Disk, volume string) error {
	if disk == nil {
		return fmt.Errorf("disk is nil")
	}

	ld, err := ldn.DiskHandler.GetLocalDisk(client.ObjectKey{Name: disk.Name})
	if err != nil {
		log.Errorf("failed to get LocalDisk %s", err.Error())
		return err
	}
	ldn.DiskHandler.For(*ld)
	ldn.DiskHandler.SetupLabel(labels.Set{ReservedPVCKey: volume})
	ldn.DiskHandler.SetupStatus(v1alpha1.LocalDiskReserved)

	return ldn.DiskHandler.UpdateStatus()
}

// ReserveDiskForVolume reserve a LocalDisk by update LocalDisk status to Reserved and label this disk for the volume
func (ldn *LocalDiskNodesManager) ReserveDiskForVolume(reqDisk Disk, volume string) error {
	ldn.mutex.Lock()
	defer ldn.mutex.Unlock()

	// get all disks attached on this node
	existDisks, err := ldn.GetNodeDisks(reqDisk.AttachNode)
	if err != nil {
		log.WithError(err).Errorf("failed to get node %s disks", reqDisk.AttachNode)
		return err
	}

	// find out all matchable disks
	var matchDisks []*Disk
	for _, existDisk := range existDisks {
		if ldn.filterDisk(reqDisk, *existDisk) {
			matchDisks = append(matchDisks, existDisk)
		}
	}
	if len(matchDisks) == 0 {
		return fmt.Errorf("no available disk for request: %+v", reqDisk)
	}

	// reserve one most matchable disk
	finalSelectDisk := ldn.diskScoreMax(reqDisk, matchDisks)

	// update disk status to Reserved
	if err = ldn.reserve(finalSelectDisk, volume); err != nil {
		log.WithError(err).Errorf("failed to reserve disk %s", finalSelectDisk.Name)
		return err
	}

	return nil
}

func (ldn *LocalDiskNodesManager) FilterFreeDisks(reqDisks []Disk) (bool, error) {
	ldn.mutex.Lock()
	defer ldn.mutex.Unlock()
	if len(reqDisks) == 0 {
		return true, nil
	}

	// get all disks attached on this node
	existDisks, err := ldn.GetNodeDisks(reqDisks[0].AttachNode)
	if err != nil {
		log.WithError(err).Errorf("failed to get node %s disks", reqDisks[0].AttachNode)
		return false, err
	}

	for _, reqDisk := range reqDisks {
		// find out all matchable disks
		var matchDisks []*Disk
		for _, existDisk := range existDisks {
			if ldn.filterDisk(reqDisk, *existDisk) {
				matchDisks = append(matchDisks, existDisk)
			}
		}

		if len(matchDisks) == 0 {
			return false, fmt.Errorf("no available disk for request: %+v", reqDisk)
		}
	}

	return true, nil
}

func convertToDisk(diskNode, diskName string, disk v1alpha1.Disk) *Disk {
	return &Disk{
		AttachNode: diskNode,
		Name:       diskName,
		DevPath:    disk.DevPath,
		Capacity:   disk.Capacity,
		DiskType:   disk.DiskType,
		Status:     disk.Status,
	}
}

func isSameDisk(d1, d2 Disk) bool {
	return reflect.DeepEqual(d1, d2)
}

func init() {
	// create LocalDiskNode Resource first when this module is imported
	cli, err := NewLocalDiskManager().GetClient()
	if err != nil {
		log.Errorf("failed to get cli %s.", err.Error())
		return
	}

	// LocalDiskNode will be created if not exist
	ldn, err := cli.Get(utils.GetNodeName())
	if ldn.GetName() != "" {
		log.Infof("LocalDiskNode %s is already exist.", ldn.GetName())
		return
	}

	ldn, _ = localdisknode.NewBuilder().WithName(utils.GetNodeName()).
		SetupAttachNode(utils.GetNodeName()).Build()
	if _, err = cli.Create(ldn); err != nil {
		log.Errorf("failed to create LocalDiskNode instance %s.", err.Error())
		return
	}

	log.Infof("LocalDiskNode %s create successfully.", ldn.GetName())
}
