package SmokeTest

import (
	"context"
	ldapis "github.com/hwameistor/local-disk-manager/pkg/apis"
	ldv1 "github.com/hwameistor/local-disk-manager/pkg/apis/hwameistor/v1alpha1"
	lsv1 "github.com/hwameistor/local-storage/pkg/apis/hwameistor/v1alpha1"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/hwameistor/local-disk-manager/test/e2e/framework"
	apiv1 "k8s.io/api/core/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os/exec"
	"regexp"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func runInLinux(cmd string) string {
	result, err := exec.Command("/bin/sh", "-c", cmd).Output()
	if err != nil {
		logrus.Printf("ERROR:%+v ", err)
	}
	return string(result)
}

func nodeList() *apiv1.NodeList {
	logrus.Printf("get node list")
	f := framework.NewDefaultFramework(ldapis.AddToScheme)
	client := f.GetClient()
	nodelist := &apiv1.NodeList{}
	err := client.List(context.TODO(), nodelist)
	if err != nil {
		logrus.Printf("%+v ", err)
		f.ExpectNoError(err)
	}
	return nodelist
}

func addLabels() {
	logrus.Printf("add node labels")
	f := framework.NewDefaultFramework(ldapis.AddToScheme)
	client := f.GetClient()
	nodelist := &apiv1.NodeList{}
	err := client.List(context.TODO(), nodelist)
	if err != nil {
		f.ExpectNoError(err)
		logrus.Printf("%+v ", err)
	}
	for _, nodes := range nodelist.Items {
		node := &apiv1.Node{}
		nodeKey := k8sclient.ObjectKey{
			Name: nodes.Name,
		}
		err := client.Get(context.TODO(), nodeKey, node)
		if err != nil {
			logrus.Printf("%+v ", err)
			f.ExpectNoError(err)
		}

		if _, exists := node.Labels["lvm.hwameistor.io/enable"]; !exists {
			node.Labels["lvm.hwameistor.io/enable"] = "true"
			logrus.Printf("adding labels ")
			err := client.Update(context.TODO(), node)
			if err != nil {
				logrus.Printf("%+v ", err)
				f.ExpectNoError(err)
			}
		}

	}
}

func installHwameiStorByHelm() {
	logrus.Infof("helm install hwameistor")
	_ = runInLinux("cd ../helm-charts/charts && helm install hwameistor -n hwameistor --create-namespace --generate-name")
}

func configureEnvironment(ctx context.Context) bool {
	installHwameiStorByHelm()
	addLabels()
	f := framework.NewDefaultFramework(lsv1.AddToScheme)
	client := f.GetClient()

	localStorage := &appsv1.DaemonSet{}
	localStorageKey := k8sclient.ObjectKey{
		Name:      "hwameistor-local-storage",
		Namespace: "hwameistor",
	}
	err := client.Get(ctx, localStorageKey, localStorage)
	if err != nil {
		logrus.Error("%+v ", err)
		f.ExpectNoError(err)
	}

	controller := &appsv1.Deployment{}
	controllerKey := k8sclient.ObjectKey{
		Name:      "hwameistor-local-storage-csi-controller",
		Namespace: "hwameistor",
	}
	err = client.Get(context.TODO(), controllerKey, controller)
	if err != nil {
		logrus.Error(err)
		f.ExpectNoError(err)
	}

	scheduler := &appsv1.Deployment{}
	schedulerKey := k8sclient.ObjectKey{
		Name:      "hwameistor-scheduler",
		Namespace: "hwameistor",
	}

	err = client.Get(context.TODO(), schedulerKey, scheduler)
	if err != nil {
		logrus.Error(err)
		f.ExpectNoError(err)
	}
	localDiskManager := &appsv1.DaemonSet{}
	localDiskManagerKey := k8sclient.ObjectKey{
		Name:      "hwameistor-local-disk-manager",
		Namespace: "hwameistor",
	}

	err = client.Get(ctx, localDiskManagerKey, localDiskManager)
	if err != nil {
		logrus.Error(err)
		f.ExpectNoError(err)

	}

	logrus.Infof("waiting for ready")
	ch := make(chan struct{}, 1)
	go func() {
		for localStorage.Status.DesiredNumberScheduled != localStorage.Status.NumberAvailable || controller.Status.AvailableReplicas != int32(1) || scheduler.Status.AvailableReplicas != int32(1) || localDiskManager.Status.DesiredNumberScheduled != localDiskManager.Status.NumberAvailable {
			time.Sleep(10 * time.Second)
			err := client.Get(ctx, localStorageKey, localStorage)
			if err != nil {
				logrus.Error("%+v ", err)
				f.ExpectNoError(err)
			}
			err = client.Get(ctx, controllerKey, controller)
			if err != nil {
				logrus.Error("%+v ", err)
				f.ExpectNoError(err)
			}
			err = client.Get(ctx, schedulerKey, scheduler)
			if err != nil {
				logrus.Error("%+v ", err)
				f.ExpectNoError(err)
			}
			err = client.Get(ctx, localDiskManagerKey, localDiskManager)
			if err != nil {
				logrus.Error(err)
				f.ExpectNoError(err)
			}

		}
		ch <- struct{}{}
	}()

	select {
	case <-ch:
		logrus.Infof("Components are ready ")
		return true
	case <-time.After(5 * time.Minute):
		logrus.Error("timeout")
		return false

	}

}

func uninstallHelm() {
	logrus.Printf("helm uninstall hwameistor")
	_ = runInLinux("helm list -A | grep 'hwameistor' | awk '{print $1}' | xargs helm uninstall -n hwameistor")
	logrus.Printf("clean all hwameistor crd")
	f := framework.NewDefaultFramework(extv1.AddToScheme)
	client := f.GetClient()
	crdList := extv1.CustomResourceDefinitionList{}
	err := client.List(context.TODO(), &crdList)
	if err != nil {
		logrus.Printf("%+v ", err)
		f.ExpectNoError(err)
	}
	for _, crd := range crdList.Items {
		myBool, _ := regexp.MatchString(".*hwameistor.*", crd.Name)
		if myBool {
			err := client.Delete(context.TODO(), &crd)
			if err != nil {
				logrus.Printf("%+v ", err)
				f.ExpectNoError(err)
			}
		}

	}
	logrus.Printf("waiting for uninstall hwameistor")

}

func createLdc(ctx context.Context) error {
	logrus.Printf("create ldc for each node")
	nodelist := nodeList()
	f := framework.NewDefaultFramework(ldapis.AddToScheme)
	client := f.GetClient()
	for _, nodes := range nodelist.Items {
		exmlocalDiskClaim := &ldv1.LocalDiskClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "localdiskclaim-" + nodes.Name,
				Namespace: "kube-system",
			},
			Spec: ldv1.LocalDiskClaimSpec{
				NodeName: nodes.Name,
				Description: ldv1.DiskClaimDescription{
					DiskType: "HDD",
				},
			},
		}
		err := client.Create(ctx, exmlocalDiskClaim)
		if err != nil {
			logrus.Printf("Create LDC failed ：%+v ", err)
			f.ExpectNoError(err)
		}
	}

	err := wait.PollImmediate(3*time.Second, 3*time.Minute, func() (done bool, err error) {
		for _, nodes := range nodelist.Items {
			time.Sleep(3 * time.Second)
			localDiskClaim := &ldv1.LocalDiskClaim{}
			localDiskClaimKey := k8sclient.ObjectKey{
				Name:      "localdiskclaim-" + nodes.Name,
				Namespace: "kube-system",
			}
			err := client.Get(ctx, localDiskClaimKey, localDiskClaim)
			if err != nil {
				logrus.Error(err)
				f.ExpectNoError(err)
			}
			if localDiskClaim.Status.Status != ldv1.LocalDiskClaimStatusBound {
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		logrus.Error(err)
		return err
	} else {
		return nil
	}

}
