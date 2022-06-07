package localdisknode

import (
	"context"
	"reflect"

	v1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"

	ldm "github.com/hwameistor/local-disk-manager/pkg/apis/hwameistor/v1alpha1"
	"github.com/hwameistor/local-disk-manager/pkg/controller/localdisk"
	"github.com/hwameistor/local-disk-manager/pkg/utils"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new LocalDiskNode Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileLocalDiskNode{
		client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("localdisknode-controller"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("localdisknode-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource LocalDiskNode
	err = c.Watch(&source.Kind{Type: &ldm.LocalDiskNode{}}, &handler.EnqueueRequestForObject{}, withCurrentNode())
	if err != nil {
		return err
	}

	localDiskToLocalDiskNodeRequestFunc := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			ld, ok := a.Object.(*ldm.LocalDisk)
			if !ok || ld.Spec.NodeName != utils.GetNodeName() {
				return []reconcile.Request{}
			}

			return []reconcile.Request{
				reconcile.Request{
					NamespacedName: types.NamespacedName{Name: ld.Spec.NodeName},
				},
			}
		})

	// Watch for changes for resource LocalDisk on this node
	err = c.Watch(&source.Kind{Type: &ldm.LocalDisk{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: localDiskToLocalDiskNodeRequestFunc})
	if err != nil {
		return err
	}

	return nil
}

// withCurrentNode filter volume request for this node
func withCurrentNode() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			node, _ := event.Object.DeepCopyObject().(*ldm.LocalDiskNode)
			return node.Spec.AttachNode == utils.GetNodeName()
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			node, _ := deleteEvent.Object.DeepCopyObject().(*ldm.LocalDiskNode)
			return node.Spec.AttachNode == utils.GetNodeName()
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			node, _ := updateEvent.ObjectNew.DeepCopyObject().(*ldm.LocalDiskNode)
			return node.Spec.AttachNode == utils.GetNodeName() &&
				updateEvent.MetaNew.GetGeneration() != updateEvent.MetaOld.GetGeneration()
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			node, _ := genericEvent.Object.DeepCopyObject().(*ldm.LocalDiskNode)
			return node.Spec.AttachNode == utils.GetNodeName()
		},
	}
}

// blank assignment to verify that ReconcileLocalDiskNode implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileLocalDiskNode{}

// ReconcileLocalDiskNode reconciles a LocalDiskNode object
type ReconcileLocalDiskNode struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client   client.Client
	scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a LocalDiskNode object and makes changes based on the state read
func (r *ReconcileLocalDiskNode) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.WithField("LocalDiskNode", request.NamespacedName).Info("Reconciling LocalDiskNode")
	ldnHandler := NewDiskNodeHelper(r.client, r.Recorder)
	err := ldnHandler.For(request.NamespacedName)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		log.WithError(err).Errorf("failed to get LocalDiskNode %s", request.NamespacedName)
		return reconcile.Result{}, err
	}

	// The main task here is to update the resource status of
	// the LocalDisk of this node to the latest status.
	newDisks, err := ldnHandler.ListNodeDisks()
	if err != nil {
		log.WithError(err).Errorf("failed to list disks on node %s", request.Name)
		return reconcile.Result{}, err
	}

	// find out new disks
	needUpdateDisks := map[string]ldm.Disk{}
	for name, newDisk := range newDisks {
		if !ldnHandler.IsSameDisk(name, newDisk) {
			needUpdateDisks[name] = *newDisk.DeepCopy()
		}
	}

	// find out disk which is removed already
	needRemoveDisks := map[string]ldm.Disk{}
	for name, disk := range ldnHandler.Disks() {
		if _, exist := newDisks[name]; !exist {
			needRemoveDisks[name] = *disk.DeepCopy()
		}
	}

	// update or remove disk
	ldnHandler.UpdateDiskLists(needUpdateDisks, needRemoveDisks)
	ldnHandler.UpdateDiskStats()

	return reconcile.Result{}, ldnHandler.UpdateStatus()
}

type DiskNodeHandler struct {
	client.Client
	record.EventRecorder
	diskNode    *ldm.LocalDiskNode
	diskHandler *localdisk.LocalDiskHandler
}

func NewDiskNodeHelper(cli client.Client, recorder record.EventRecorder) *DiskNodeHandler {
	return &DiskNodeHandler{
		Client:        cli,
		EventRecorder: recorder,
		diskHandler:   localdisk.NewLocalDiskHandler(cli, recorder),
	}
}

func (n *DiskNodeHandler) For(name types.NamespacedName) error {
	ldn := &ldm.LocalDiskNode{}
	err := n.Get(context.Background(), name, ldn)
	if err != nil {
		return err
	}

	n.diskNode = ldn
	return nil
}

func (n *DiskNodeHandler) UpdateStatus() error {
	err := n.Update(context.Background(), n.diskNode)
	if err != nil {
		log.WithError(err).Errorf("failed to update disks")
	} else {
		log.Infof("Update disks successfully")
	}

	return err
}

func (n *DiskNodeHandler) UpdateDiskLists(updateDisks, removedDisks map[string]ldm.Disk) {
	// remove disk
	for name, removeDisk := range removedDisks {
		delete(n.diskNode.Status.Disks, name)
		if removeDisk.Status != string(ldm.LocalDiskUnclaimed) && removeDisk.Status != string(ldm.LocalDiskReleased) {
			n.EventRecorder.Eventf(n.diskNode, v1.EventTypeWarning, "RemoveDisk", ""+
				"Disk %s is removed but state is %s, disk last info: %+v", removeDisk.DevPath, removeDisk.Status, removeDisk)
		} else {
			n.EventRecorder.Eventf(n.diskNode, v1.EventTypeNormal, "RemoveDisk", "Remove disk %s", removeDisk.DevPath)
		}
	}

	// update disk
	if n.diskNode.Status.Disks == nil {
		n.diskNode.Status.Disks = make(map[string]ldm.Disk, len(updateDisks))
	}
	for name, updateDisk := range updateDisks {
		oldDisk, exist := n.diskNode.Status.Disks[name]
		if !exist {
			n.EventRecorder.Eventf(n.diskNode, v1.EventTypeNormal, "AddDisk", "Add new disk %s", updateDisk.DevPath)
		} else {
			n.EventRecorder.Eventf(n.diskNode, v1.EventTypeNormal, "UpdateDisk", ""+
				"Disk %s old info: %+v", name, oldDisk)
		}

		n.diskNode.Status.Disks[name] = updateDisk
	}
}

func (n *DiskNodeHandler) UpdateDiskStats() {
	n.diskNode.Status.TotalDisk = 0
	n.diskNode.Status.AllocatableDisk = 0
	for _, disk := range n.Disks() {
		n.diskNode.Status.TotalDisk++
		if disk.Status == string(ldm.LocalDiskUnclaimed) ||
			disk.Status == string(ldm.LocalDiskReleased) {
			n.diskNode.Status.AllocatableDisk++
		}
	}
}

func (n *DiskNodeHandler) Disks() map[string]ldm.Disk {
	return n.diskNode.Status.Disks
}

func (n *DiskNodeHandler) ListNodeDisks() (map[string]ldm.Disk, error) {
	lds, err := n.diskHandler.ListNodeLocalDisk(n.diskNode.Spec.AttachNode)
	if err != nil {
		return nil, err
	}

	disks := map[string]ldm.Disk{}
	for _, ld := range lds.Items {
		disks[ld.GetName()] = convertToDisk(ld)
	}
	return disks, nil
}

// IsSameDisk judge the disk in LocalDiskNode is same as disk in LocalDisk
func (n *DiskNodeHandler) IsSameDisk(name string, newDisk ldm.Disk) bool {
	oldDisk := n.Disks()[name]

	return reflect.DeepEqual(&oldDisk, &newDisk)
}

func convertToDisk(ld ldm.LocalDisk) ldm.Disk {
	return ldm.Disk{
		DevPath:  ld.Spec.DevicePath,
		Capacity: ld.Spec.Capacity,
		DiskType: ld.Spec.DiskAttributes.Type,
		Status:   string(ld.Status.State),
	}
}
