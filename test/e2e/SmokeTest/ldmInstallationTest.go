package SmokeTest

import (
	"context"
	ldapis "github.com/hwameistor/local-disk-manager/pkg/apis"
	"github.com/hwameistor/local-disk-manager/test/e2e/framework"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = ginkgo.Describe("test Local Disk Manager installation", func() {
	f := framework.NewDefaultFramework(ldapis.AddToScheme)
	client := f.GetClient()
	ctx := context.TODO()

	ginkgo.It("Configure the base environment", func() {
		installHwameiStorByHelm()
	})
	ginkgo.Context("test local-disk-manager", func() {
		ginkgo.It("check status", func() {
			daemonset := &appsv1.DaemonSet{}
			daemonsetKey := k8sclient.ObjectKey{
				Name:      "hwameistor-local-disk-manager",
				Namespace: "hwameistor",
			}

			err := client.Get(ctx, daemonsetKey, daemonset)
			if err != nil {
				f.ExpectNoError(err)
				logrus.Printf("%+v ", err)
			}
			gomega.Expect(daemonset.Status.DesiredNumberScheduled).To(gomega.Equal(daemonset.Status.NumberAvailable))
		})
	})
	ginkgo.It("Clean up the environment", func() {
		uninstallHelm()

	})

})
