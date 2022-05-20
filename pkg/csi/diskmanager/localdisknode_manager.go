package diskmanager

import (
	"fmt"
	"reflect"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/hwameistor/local-disk-manager/pkg/utils"
	log "github.com/sirupsen/logrus"

	"github.com/hwameistor/local-disk-manager/pkg/apis/hwameistor/v1alpha1"
	"github.com/hwameistor/local-disk-manager/pkg/builder/localdisknode"
	"github.com/hwameistor/local-disk-manager/pkg/controller/localdisk"
	"github.com/hwameistor/local-disk-manager/pkg/utils/kubernetes"
)

var (
	once sync.Once
	ldn  *LocalDiskNodesManager
)

// LocalDiskNodesManager manage all disks in the cluster by interacting with LocalDisk resources
type LocalDiskNodesManager struct {
	// GetClient for query LocalDiskNode resources from k8s
	GetClient func() (*localdisknode.Kubeclient, error)

	// distributed lock or mutex lock(controller already has distributed lock )
	mutex sync.Mutex

	// DiskHandler manage LD resources in cluster
	DiskHandler *localdisk.LocalDiskHandler
}

func (ldn *LocalDiskNodesManager) ReleaseDisk(diskName string) error {
	ld, err := ldn.DiskHandler.GetLocalDisk(client.ObjectKey{Name: diskName})
	if err != nil {
		return err
	}
	ldn.DiskHandler.For(*ld)
	ldn.DiskHandler.SetupStatus(v1alpha1.LocalDiskReleased)
	return ldn.DiskHandler.UpdateStatus()
}

func NewLocalDiskManager() *LocalDiskNodesManager {
	once.Do(func() {
		ldn = &LocalDiskNodesManager{}
		ldn.GetClient = localdisknode.NewKubeclient
		cli, _ := kubernetes.NewClient()
		recoder, _ := kubernetes.NewRecorderFor("localdisknodemanager")
		ldn.DiskHandler = localdisk.NewLocalDiskHandler(cli, recoder)
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

func (ldn *LocalDiskNodesManager) filterNotReservedDisk(reqDisk, existDisk Disk) bool {
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

func (ldn *LocalDiskNodesManager) filterReservedDisk(reqDisk, existDisk Disk) bool {
	if existDisk.Status != DiskStatusReserved {
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

// ClaimDisk claim a LocalDisk by update LocalDisk status to InUse
func (ldn *LocalDiskNodesManager) ClaimDisk(disk *Disk) error {
	if disk == nil {
		return fmt.Errorf("disk is nil")
	}

	ld, err := ldn.DiskHandler.GetLocalDisk(client.ObjectKey{Name: disk.Name})
	if err != nil {
		log.Errorf("failed to get LocalDisk %s", err.Error())
		return err
	}
	ldn.DiskHandler.For(*ld)
	ldn.DiskHandler.SetupStatus(v1alpha1.LocalDiskInUse)

	return ldn.DiskHandler.UpdateStatus()
}

// ReserveDisk reserve a LocalDisk by update LocalDisk status to Reserved
func (ldn *LocalDiskNodesManager) ReserveDisk(disk *Disk) error {
	if disk == nil {
		return fmt.Errorf("disk is nil")
	}

	ld, err := ldn.DiskHandler.GetLocalDisk(client.ObjectKey{Name: disk.Name})
	if err != nil {
		log.Errorf("failed to get LocalDisk %s", err.Error())
		return err
	}
	ldn.DiskHandler.For(*ld)
	ldn.DiskHandler.SetupStatus(v1alpha1.LocalDiskReserved)

	return ldn.DiskHandler.UpdateStatus()
}

func (ldn *LocalDiskNodesManager) PreSelectFreeDisks(reqDisks []Disk) (bool, error) {
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

	var reservedDisks []*Disk

	// try to reserve disks
	for _, reqDisk := range reqDisks {
		// find out all matchable disks
		var matchDisks []*Disk
		for _, existDisk := range existDisks {
			if ldn.filterNotReservedDisk(reqDisk, *existDisk) {
				matchDisks = append(matchDisks, existDisk)
			}
		}

		if len(matchDisks) == 0 {
			return false, fmt.Errorf("no available disk for request: %+v", reqDisk)
		}

		// reserve one disk
		reserveDisk := ldn.diskScoreMax(reqDisk, matchDisks)
		reservedDisks = append(reservedDisks, reserveDisk)
		for i, d := range existDisks {
			if isSameDisk(*d, *reserveDisk) {
				existDisks[i].Status = DiskStatusReserved
				break
			}
		}
	}

	// update disk status to Reserved
	for _, reserveDisk := range reservedDisks {
		if err = ldn.ReserveDisk(reserveDisk); err != nil {
			log.WithError(err).Errorf("failed to reserve disk %s", reserveDisk.Name)
			return false, err
		}
	}

	return true, nil
}

func (ldn *LocalDiskNodesManager) SelectFreeDisk(reqDisk Disk) (selDisk *Disk, err error) {
	ldn.mutex.Lock()
	defer func() {
		if selDisk != nil {
			if err = ldn.ClaimDisk(selDisk); err != nil {
				err = fmt.Errorf("no available existDisk(selected existDisk %s/%s but update status fail due to err: %v)",
					selDisk.AttachNode, selDisk.DevPath, err)
				selDisk = nil
			}
		}

		ldn.mutex.Unlock()
	}()

	disks, err := ldn.GetNodeDisks(reqDisk.AttachNode)
	if err != nil {
		return nil, err
	}
	if len(disks) == 0 {
		return nil, fmt.Errorf("no available disk on node %s", reqDisk.AttachNode)
	}

	// only use the disk which is reserved already
	var matchDisks []*Disk
	for _, existDisk := range disks {
		if ldn.filterReservedDisk(reqDisk, *existDisk) {
			matchDisks = append(matchDisks, existDisk)
		}
	}

	if len(matchDisks) == 0 {
		return nil, fmt.Errorf("no available disk on node %s", reqDisk.AttachNode)
	}

	selDisk = ldn.diskScoreMax(reqDisk, matchDisks)
	return
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
