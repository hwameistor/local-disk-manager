package localdiskvolume

import (
	"context"
	"fmt"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/hwameistor/local-disk-manager/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	ldm "github.com/hwameistor/local-disk-manager/pkg/apis/hwameistor/v1alpha1"
	lscsi "github.com/hwameistor/local-storage/pkg/member/csi"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// LocalDiskFinalizer for the LocalDiskVolume CR
	LocalDiskFinalizer string = "localdisk.hwameistor.io/finalizer"
)

// Add creates a new LocalDiskVolume Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileLocalDiskVolume{
		client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("localdiskvolume-controller"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("localdiskvolume-controller", mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: 1,
	})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource LocalDiskVolume
	err = c.Watch(&source.Kind{Type: &ldm.LocalDiskVolume{}}, &handler.EnqueueRequestForObject{}, withCurrentNode())
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner LocalDiskVolume
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &ldm.LocalDiskVolume{},
	})
	if err != nil {
		return err
	}

	return nil
}

// withCurrentNode filter volume request for this node
func withCurrentNode() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			volume, _ := event.Object.DeepCopyObject().(*ldm.LocalDiskVolume)
			return volume.Spec.Accessibility.Node == utils.GetNodeName()
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			volume, _ := deleteEvent.Object.DeepCopyObject().(*ldm.LocalDiskVolume)
			return volume.Spec.Accessibility.Node == utils.GetNodeName()
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			volume, _ := updateEvent.ObjectNew.DeepCopyObject().(*ldm.LocalDiskVolume)
			return volume.Spec.Accessibility.Node == utils.GetNodeName()
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			volume, _ := genericEvent.Object.DeepCopyObject().(*ldm.LocalDiskVolume)
			return volume.Spec.Accessibility.Node == utils.GetNodeName()
		},
	}
}

// blank assignment to verify that ReconcileLocalDiskVolume implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileLocalDiskVolume{}

// ReconcileLocalDiskVolume reconciles a LocalDiskVolume object
type ReconcileLocalDiskVolume struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client   client.Client
	scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// Reconcile
func (r *ReconcileLocalDiskVolume) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.WithField("LocalDiskVolume", request.Name).Info("Reconciling LocalDiskVolume")
	var result reconcile.Result
	v, err := r.reconcileForVolume(request.NamespacedName)
	if err != nil {
		if errors.IsNotFound(err) {
			return result, nil
		}
		log.WithError(err).Errorf("Failed to new handler for LocalDiskVolume %s", request.Name)

		return result, err
	}

	// check finalizers
	if err := v.CheckFinalizers(); err != nil {
		log.WithError(err).Errorf("Failed to check finalizers for LocalDiskVolume %s", request.Name)
	}

	switch v.VolumeState() {
	// Mount Volumes
	case ldm.VolumeStateNotReady, ldm.VolumeStateEmpty:
		return v.reconcileMount()

	// Unmount Volumes
	case ldm.VolumeStateToBeUnmount:
		return v.reconcileUnmount()

	// ToDelete Volume
	case ldm.VolumeStateToBeDeleted:
		return v.reconcileToBeDeleted()

	// Delete Volume
	case ldm.VolumeStateDeleted:
		return v.reconcileDeleted()

	// Volume Ready/Creating/... do nothing
	default:
		log.Infof("Volume state %s , no handling now", v.VolumeState())
	}

	return result, nil
}

func (r *ReconcileLocalDiskVolume) reconcileForVolume(name client.ObjectKey) (*DiskVolumeHandler, error) {
	volumeHandler := NewLocalDiskVolumeHandler(r.client, r.Recorder)
	volume, err := volumeHandler.GetLocalDiskVolume(name)
	if err != nil {
		return nil, err
	}

	volumeHandler.For(volume)
	return volumeHandler, nil
}

// DiskVolumeHandler
type DiskVolumeHandler struct {
	client.Client
	record.EventRecorder
	ldv     *ldm.LocalDiskVolume
	mounter lscsi.Mounter
}

// NewLocalDiskHandler
func NewLocalDiskVolumeHandler(cli client.Client, recorder record.EventRecorder) *DiskVolumeHandler {
	logger := log.WithField("Module", "CSIPlugin")
	return &DiskVolumeHandler{
		Client:        cli,
		EventRecorder: recorder,
		mounter:       lscsi.NewLinuxMounter(logger),
	}
}

func (v *DiskVolumeHandler) reconcileMount() (reconcile.Result, error) {
	var err error
	var result reconcile.Result
	var devPath = v.GetDevPath()
	var mountPoints = v.GetMountPoints()

	if devPath == "" || len(mountPoints) == 0 {
		log.Infof("DevPath or MountPoints is empty, no operation here")
		return result, nil
	}

	// Mount RawBlock or FileSystem Volumes
	for _, mountPoint := range mountPoints {
		if mountPoint.Phase != ldm.MountPointToBeMounted {
			continue
		}

		switch mountPoint.VolumeCap.AccessType {
		case ldm.VolumeCapability_AccessType_Mount:
			err = v.MountFileSystem(devPath, mountPoint.TargetPath, mountPoint.FsTye, mountPoint.MountOptions...)
		case ldm.VolumeCapability_AccessType_Block:
			err = v.MountRawBlock(devPath, mountPoint.TargetPath)
		default:
			// record and skip this mountpoint
			v.RecordEvent(corev1.EventTypeWarning, "ValidAccessType", "AccessType of MountPoint %s "+
				"is invalid, ignore it", mountPoint)
		}

		if err != nil {
			log.WithError(err).Errorf("Failed to mount %s to %s", devPath, mountPoint.TargetPath)
			result.Requeue = true
			continue
		}
		v.UpdateMountPointPhase(mountPoint.TargetPath, ldm.MountPointMounted)
	}

	if !result.Requeue {
		v.SetupVolumeStatus(ldm.VolumeStateReady)
	}

	return result, v.UpdateLocalDiskVolume()
}

func (v *DiskVolumeHandler) reconcileUnmount() (reconcile.Result, error) {
	var err error
	var result reconcile.Result
	var mountPoints = v.GetMountPoints()

	for _, mountPoint := range mountPoints {
		if mountPoint.Phase == ldm.MountPointToBeUnMount {
			if err = v.UnMount(mountPoint.TargetPath); err != nil {
				log.WithError(err).Errorf("Failed to unmount %s", mountPoint)
				result.Requeue = true
				continue
			}

			v.MoveMountPoint(mountPoint.TargetPath)
		}
	}
	if !result.Requeue {
		// Considering that a disk will be mounted to multiple pods,
		// if a mount point is processed, the entire LocalDiskVolume will be setup to the Ready state
		v.SetupVolumeStatus(ldm.VolumeStateReady)
	}

	return result, v.UpdateLocalDiskVolume()
}

func (v *DiskVolumeHandler) reconcileToBeDeleted() (reconcile.Result, error) {
	var result reconcile.Result
	var mountPoints = v.GetMountPoints()

	if len(mountPoints) > 0 {
		log.Infof("Volume %s remains %d mountpoint, no operation here",
			v.ldv.Name, len(mountPoints))
		result.Requeue = true
		return result, nil
	}

	return result, v.DeleteLocalDiskVolume()
}

func (v *DiskVolumeHandler) reconcileDeleted() (reconcile.Result, error) {
	return reconcile.Result{}, v.Delete(context.Background(), v.ldv)
}

func (v *DiskVolumeHandler) GetLocalDiskVolume(key client.ObjectKey) (volume *ldm.LocalDiskVolume, err error) {
	volume = &ldm.LocalDiskVolume{}
	err = v.Get(context.Background(), key, volume)
	return
}

func (v *DiskVolumeHandler) RecordEvent(eventtype, reason, messageFmt string, args ...interface{}) {
	v.EventRecorder.Eventf(v.ldv, eventtype, reason, messageFmt, args...)
}

func (v *DiskVolumeHandler) UpdateLocalDiskVolume() error {
	return v.Update(context.Background(), v.ldv)
}

func (v *DiskVolumeHandler) DeleteLocalDiskVolume() error {
	v.SetupVolumeStatus(ldm.VolumeStateDeleted)
	return v.UpdateLocalDiskVolume()
}

func (v *DiskVolumeHandler) RemoveFinalizers() error {
	// range all finalizers and notify all owners
	v.ldv.Finalizers = nil
	return nil
}

func (v *DiskVolumeHandler) AddFinalizers(finalizer []string) {
	v.ldv.Finalizers = finalizer
	return
}

func (v *DiskVolumeHandler) GetMountPoints() []ldm.MountPoint {
	return v.ldv.Status.MountPoints
}

func (v *DiskVolumeHandler) GetDevPath() string {
	return v.ldv.Status.DevPath
}

func (v *DiskVolumeHandler) MountRawBlock(devPath, mountPoint string) error {
	// fixme: should check mount points again when do mount
	// mount twice will cause error
	if err := v.mounter.MountRawBlock(devPath, mountPoint); err != nil {
		v.RecordEvent(corev1.EventTypeWarning, "MountFailed", "Failed to mount block %s to %s due to err: %v",
			devPath, mountPoint, err)
		return err
	}

	v.RecordEvent(corev1.EventTypeNormal, "MountSuccess", "MountRawBlock %s to %s successfully",
		devPath, mountPoint)
	return nil
}

func (v *DiskVolumeHandler) MountFileSystem(devPath, mountPoint, fsType string, options ...string) error {
	// fixme: should check mount points again when do mount
	// mount twice will cause error
	if err := v.mounter.FormatAndMount(devPath, mountPoint, fsType, options); err != nil {
		v.RecordEvent(corev1.EventTypeWarning, "MountFailed", "Failed to mount filesystem %s to %s due to err: %v",
			devPath, mountPoint, err)
		return err
	}

	v.RecordEvent(corev1.EventTypeNormal, "MountSuccess", "MountFileSystem %s to %s successfully",
		devPath, mountPoint)
	return nil
}

func (v *DiskVolumeHandler) UnMount(mountPoint string) error {
	if err := v.mounter.Unmount(mountPoint); err != nil {
		v.RecordEvent(corev1.EventTypeWarning, "UnMountFailed", "Failed to umount %s due to err: %v",
			mountPoint, err)
		return err
	}

	v.RecordEvent(corev1.EventTypeNormal, "UnMountSuccess", "Unmount %s successfully",
		mountPoint)
	return nil
}

func (v *DiskVolumeHandler) VolumeState() ldm.State {
	return v.ldv.Status.State
}

func (v *DiskVolumeHandler) ExistMountPoint(targetPath string) bool {
	for _, mountPoint := range v.GetMountPoints() {
		if mountPoint.TargetPath == targetPath {
			return true
		}
	}

	return false
}

func (v *DiskVolumeHandler) AppendMountPoint(targetPath string, volCap *csi.VolumeCapability) {
	mountPoint := ldm.MountPoint{TargetPath: targetPath, Phase: ldm.MountPointToBeMounted}
	switch volCap.AccessType.(type) {
	case *csi.VolumeCapability_Block:
		mountPoint.VolumeCap = ldm.VolumeCapability{AccessType: ldm.VolumeCapability_AccessType_Block}
	case *csi.VolumeCapability_Mount:
		mountPoint.FsTye = volCap.GetMount().FsType
		mountPoint.MountOptions = volCap.GetMount().MountFlags
		mountPoint.VolumeCap = ldm.VolumeCapability{AccessType: ldm.VolumeCapability_AccessType_Mount}
	default:
	}
	v.ldv.Status.MountPoints = append(v.ldv.Status.MountPoints, mountPoint)
}

func (v *DiskVolumeHandler) MoveMountPoint(targetPath string) {
	for i, mountPoint := range v.GetMountPoints() {
		if mountPoint.TargetPath == targetPath {
			v.ldv.Status.MountPoints = append(v.ldv.Status.MountPoints[:i], v.ldv.Status.MountPoints[i+1:]...)
			return
		}
	}
}

func (v *DiskVolumeHandler) UpdateMountPointPhase(targetPath string, phase ldm.State) {
	for i, mountPoint := range v.GetMountPoints() {
		if mountPoint.TargetPath == targetPath {
			v.ldv.Status.MountPoints[i].Phase = phase
			return
		}
	}
}

// WaitVolumeReady wait LocalDiskVolume Ready
func (v *DiskVolumeHandler) WaitVolumeReady(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		return fmt.Errorf("no deadline is set")
	}

	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context error occured when wait volume ready: %v", ctx.Err())
		case <-timer.C:
		}

		if err := v.RefreshVolume(); err != nil {
			return err
		}
		if v.VolumeState() == ldm.VolumeStateReady {
			return nil
		}
		timer.Reset(1 * time.Second)
	}
}

func (v *DiskVolumeHandler) WaitVolume(ctx context.Context, state ldm.State) error {
	if _, ok := ctx.Deadline(); !ok {
		return fmt.Errorf("no deadline is set")
	}

	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context error occured when wait volume ready: %v", ctx.Err())
		case <-timer.C:
		}

		if err := v.RefreshVolume(); err != nil {
			return err
		}
		if v.VolumeState() == state {
			return nil
		}
		timer.Reset(1 * time.Second)
	}
}

// WaitVolumeUnmounted wait a special mountpoint is unmounted
func (v *DiskVolumeHandler) WaitVolumeUnmounted(ctx context.Context, mountPoint string) error {
	if _, ok := ctx.Deadline(); !ok {
		return fmt.Errorf("no deadline is set")
	}

	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context error occured when wait volume ready: %v", ctx.Err())
		case <-timer.C:
		}

		if err := v.RefreshVolume(); err != nil {
			return err
		}
		if v.VolumeState() == ldm.VolumeStateReady && !v.ExistMountPoint(mountPoint) {
			return nil
		}
		timer.Reset(1 * time.Second)
	}
}

func (v *DiskVolumeHandler) RefreshVolume() error {
	newVolume, err := v.GetLocalDiskVolume(client.ObjectKey{
		Namespace: v.ldv.GetNamespace(),
		Name:      v.ldv.GetName()})
	if err != nil {
		return err
	}

	v.For(newVolume)
	return nil
}

func (v *DiskVolumeHandler) SetupVolumeStatus(status ldm.State) {
	v.ldv.Status.State = status
}

func (v *DiskVolumeHandler) CheckFinalizers() error {
	finalizers := v.ldv.GetFinalizers()
	_, ok := utils.StrFind(finalizers, LocalDiskFinalizer)
	if ok {
		return nil
	}

	// volume is ready for delete, don't append finalizer
	if v.VolumeState() == ldm.VolumeStateDeleted {
		return nil
	}

	// add localdisk finalizers to prevent resource deleted by mistake
	finalizers = append(finalizers, LocalDiskFinalizer)
	v.AddFinalizers(finalizers)
	if err := v.UpdateLocalDiskVolume(); err != nil {
		return err
	}

	return v.RefreshVolume()
}

func (v *DiskVolumeHandler) GetBoundDisk() string {
	return v.ldv.Status.LocalDiskName
}

func (v *DiskVolumeHandler) For(volume *ldm.LocalDiskVolume) {
	v.ldv = volume
}
