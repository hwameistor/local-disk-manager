package localdiskclaim

import (
	"context"
	"github.com/hwameistor/local-disk-manager/pkg/controller/localdisk"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/reference"

	ldmv1alpha1 "github.com/hwameistor/local-disk-manager/pkg/apis/hwameistor/v1alpha1"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new LocalDiskClaim Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileLocalDiskClaim{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("localdiskclaim-controller"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("localdiskclaim-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource LocalDiskClaim
	err = c.Watch(&source.Kind{Type: &ldmv1alpha1.LocalDiskClaim{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileLocalDiskClaim implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileLocalDiskClaim{}

// ReconcileLocalDiskClaim reconciles a LocalDiskClaim object
type ReconcileLocalDiskClaim struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a LocalDiskClaim object and makes changes based on the state read
// and what is in the LocalDiskClaim.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileLocalDiskClaim) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	log.Infof("Reconcile LocalDiskClaim %s", req.Name)
	ldcHandler := NewLocalDiskClaimHandler(r.Client, r.Recorder)

	ldc, err := ldcHandler.GetLocalDiskClaim(req.NamespacedName)
	if err != nil {
		log.WithError(err).Errorf("Get localdiskclaim fail, due to error: %v", err)
		return reconcile.Result{}, err
	}

	if ldc != nil {
		ldcHandler = ldcHandler.For(*ldc.DeepCopy())
	} else {
		// Not found
		return reconcile.Result{}, nil
	}

	switch ldcHandler.Phase() {
	case ldmv1alpha1.DiskClaimStatusEmpty:
		fallthrough
	case ldmv1alpha1.LocalDiskClaimStatusPending:
		if err = ldcHandler.AssignFreeDisk(); err != nil {
			r.Recorder.Eventf(ldc, v1.EventTypeWarning, "LocalDiskClaimFail", "Assign free disk fail, due to error: %v", err)
			log.WithError(err).Errorf("Assign free disk for locadiskclaim %v/%v fail", ldc.GetNamespace(), ldc.GetName())
			return reconcile.Result{}, err
		}

	case ldmv1alpha1.LocalDiskClaimStatusBound:
		// TODO: handle delete events
	default:
		log.Warningf("LocalDiskClaim %s status %v is UNKNOWN", ldc.Name, ldcHandler.Phase())
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, nil
}

// LocalDiskClaimHandler
type LocalDiskClaimHandler struct {
	client.Client
	record.EventRecorder
	ldc ldmv1alpha1.LocalDiskClaim
}

// NewLocalDiskClaimHandler
func NewLocalDiskClaimHandler(client client.Client, recorder record.EventRecorder) *LocalDiskClaimHandler {
	return &LocalDiskClaimHandler{
		Client:        client,
		EventRecorder: recorder,
	}
}

// ListLocalDiskClaim
func (ldcHandler *LocalDiskClaimHandler) ListLocalDiskClaim() (*ldmv1alpha1.LocalDiskClaimList, error) {
	list := &ldmv1alpha1.LocalDiskClaimList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "LocalDiskClaim",
			APIVersion: "v1alpha1",
		},
	}

	err := ldcHandler.List(context.TODO(), list)
	return list, err
}

// GetLocalDiskClaim
func (ldcHandler *LocalDiskClaimHandler) GetLocalDiskClaim(key client.ObjectKey) (*ldmv1alpha1.LocalDiskClaim, error) {
	ldc := &ldmv1alpha1.LocalDiskClaim{}
	if err := ldcHandler.Get(context.Background(), key, ldc); err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return ldc, nil
}

// ListLocalDiskClaim
func (ldcHandler *LocalDiskClaimHandler) ListUnboundLocalDiskClaim() (*ldmv1alpha1.LocalDiskClaimList, error) {
	list := &ldmv1alpha1.LocalDiskClaimList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "LocalDiskClaim",
			APIVersion: "v1alpha1",
		},
	}

	// NOTE: runtime selector is only support equal
	unboundSelector := fields.OneTermEqualSelector("status.status", "")
	// emptyDiskSelector :=  fields.OneTermNotEqualSelector("spec.diskRefs", "nil")
	// nameSelector := fields.OneTermEqualSelector("metadata.name", "test")

	err := ldcHandler.List(context.TODO(), list, &client.ListOptions{FieldSelector: unboundSelector})
	return list, err
}

// For
func (ldcHandler *LocalDiskClaimHandler) For(ldc ldmv1alpha1.LocalDiskClaim) *LocalDiskClaimHandler {
	ldcHandler.ldc = ldc
	return ldcHandler
}

// AssignFreeDisk
func (ldcHandler *LocalDiskClaimHandler) AssignFreeDisk() error {
	ldHandler := localdisk.NewLocalDiskHandler(ldcHandler.Client, ldcHandler.EventRecorder)
	ldc := *ldcHandler.ldc.DeepCopy()
	ldList, err := ldHandler.ListLocalDisk()
	if err != nil {
		return err
	}

	var assignedDisks []string
	for _, ld := range ldList.Items {
		ldHandler.For(ld)
		if !ldHandler.FilterDisk(ldc) {
			continue
		}
		if err = ldHandler.BoundTo(ldc); err != nil {
			return err
		}
		if err = ldcHandler.BoundWith(ld); err != nil {
			return err
		}

		assignedDisks = append(assignedDisks, ld.GetName())
	}

	if len(assignedDisks) == 0 {
		log.Infof("There is no available disk assigned to %v", ldc.GetName())
		return nil
	}

	log.Infof("Disk %v has been assigned to %v", assignedDisks, ldc.GetName())
	return ldcHandler.UpdateClaimStatus()
}

// Bounded
func (ldcHandler *LocalDiskClaimHandler) UpdateSpec() error {
	return ldcHandler.Update(context.Background(), &ldcHandler.ldc)
}

// Bounded
func (ldcHandler *LocalDiskClaimHandler) Bounded() bool {
	return ldcHandler.ldc.Status.Status == ldmv1alpha1.LocalDiskClaimStatusBound
}

// DiskRefs
func (ldcHandler *LocalDiskClaimHandler) DiskRefs() []*v1.ObjectReference {
	return ldcHandler.ldc.Spec.DiskRefs
}

// DiskRefs
func (ldcHandler *LocalDiskClaimHandler) Phase() ldmv1alpha1.DiskClaimStatus {
	return ldcHandler.ldc.Status.Status
}

// BoundWith
func (ldcHandler *LocalDiskClaimHandler) BoundWith(ld ldmv1alpha1.LocalDisk) error {
	ldRef, err := reference.GetReference(nil, &ld)
	if err != nil {
		return err
	}

	ldcHandler.ldc.Spec.DiskRefs = append(ldcHandler.ldc.Spec.DiskRefs, ldRef)
	ldcHandler.ldc.Status.Status = ldmv1alpha1.LocalDiskClaimStatusBound

	ldcHandler.EventRecorder.Eventf(&ldcHandler.ldc, v1.EventTypeNormal, "BoundLocalDisk", "Bound disk %v", ld.Name)
	return nil
}

// SetupClaimStatus
func (ldcHandler *LocalDiskClaimHandler) SetupClaimStatus(status ldmv1alpha1.DiskClaimStatus) {
	ldcHandler.ldc.Status.Status = status
}

// UpdateStatus
func (ldcHandler *LocalDiskClaimHandler) UpdateClaimStatus() error {
	return ldcHandler.Update(context.Background(), &ldcHandler.ldc)
}

// Refresh
func (ldcHandler *LocalDiskClaimHandler) Refresh() error {
	ldc, err := ldcHandler.GetLocalDiskClaim(client.ObjectKey{Name: ldcHandler.ldc.GetName(), Namespace: ldcHandler.ldc.GetNamespace()})
	if err != nil {
		return err
	}
	ldcHandler.For(*ldc.DeepCopy())
	return nil
}
