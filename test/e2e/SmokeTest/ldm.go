package SmokeTest

import (
	"context"
	ldapis "github.com/hwameistor/local-disk-manager/pkg/apis"
	ldv1 "github.com/hwameistor/local-disk-manager/pkg/apis/hwameistor/v1alpha1"
	"github.com/hwameistor/local-disk-manager/test/e2e/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

var _ = ginkgo.AfterSuite(func() {
	output := runInLinux("sh deletedisk.sh")
	logrus.Info("delete disk", output)
})

var _ = ginkgo.Describe("test Local Disk Manager", ginkgo.Label("pr"), func() {
	f := framework.NewDefaultFramework(ldapis.AddToScheme)
	client := f.GetClient()
	ctx := context.TODO()
	ginkgo.Context("test Local Disk", func() {
		ginkgo.It("Configure the base environment", func() {
			configureEnvironment(ctx)
		})
		ginkgo.It("Check existed Local Disk", func() {
			localDiskList := &ldv1.LocalDiskList{}
			err := client.List(ctx, localDiskList)
			if err != nil {
				f.ExpectNoError(err)
			}
			logrus.Printf("There are %d local volumes ", len(localDiskList.Items))
			gomega.Expect(len(localDiskList.Items)).To(gomega.Equal(6))
		})
		ginkgo.It("Manage new disks", func() {
			output := runInLinux("sh adddisk.sh")
			logrus.Printf("add  disk : %+v", output)
			err := wait.PollImmediate(3*time.Second, 5*time.Minute, func() (done bool, err error) {
				localDiskList := &ldv1.LocalDiskList{}
				err = client.List(ctx, localDiskList)
				if err != nil {
					f.ExpectNoError(err)
					logrus.Error(err)
				}
				if len(localDiskList.Items) != 7 {
					return false, nil
				} else {
					logrus.Infof("There are %d local volumes ", len(localDiskList.Items))
					return true, nil
				}
			})
			if err != nil {
				logrus.Error("Manage new disks error", err)
				f.ExpectNoError(err)
			}
			gomega.Expect(err).To(gomega.BeNil())

		})

	})
	ginkgo.Context("test LocalDiskClaim", func() {
		ginkgo.It("Create new LocalDiskClaim", func() {
			err := createLdc(ctx)
			gomega.Expect(err).To(gomega.BeNil())

		})
	})
	ginkgo.Context("Clean up the environment", func() {
		ginkgo.It("Clean helm & crd", func() {
			uninstallHelm()
		})
	})

})
