package SmokeTest

import (
	"context"
	ldapis "github.com/hwameistor/local-disk-manager/pkg/apis"
	ldv1 "github.com/hwameistor/local-disk-manager/pkg/apis/hwameistor/v1alpha1"
	"github.com/hwameistor/local-disk-manager/test/e2e/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var _ = ginkgo.Describe("test Local Disk Manager  ", ginkgo.Label("smokeTest"), func() {
	f := framework.NewDefaultFramework(ldapis.AddToScheme)
	client := f.GetClient()
	ctx := context.TODO()
	localDiskNumber := 0
	ginkgo.Context("test Local Disk ", func() {
		ginkgo.It("Configure the base environment", func() {
			configureEnvironment(ctx)
		})
		ginkgo.It("Check existed Local Disk", func() {
			localDiskList := &ldv1.LocalDiskList{}
			err := client.List(ctx, localDiskList)
			if err != nil {
				f.ExpectNoError(err)
			}

			for i, localDisk := range localDiskList.Items {
				logrus.Printf("%+v ", localDisk.Name)
				localDiskNumber = i + 1
			}
			logrus.Printf("There are %d local volumes ", localDiskNumber)
			gomega.Expect(localDiskNumber).ToNot(gomega.Equal(0))
		})
		ginkgo.It("Manage new disks", func() {
			newlocalDiskNumber := 0
			output := runInLinux("sh adddisk.sh")
			logrus.Printf("add  disk : %+v", output)
			logrus.Printf("wait 2 minutes ")
			time.Sleep(2 * time.Minute)
			localDiskList := &ldv1.LocalDiskList{}
			err := client.List(ctx, localDiskList)
			if err != nil {
				f.ExpectNoError(err)
				logrus.Printf("%+v ", err)
			}
			for i, localDisk := range localDiskList.Items {
				logrus.Printf("%+v ", localDisk.Name)
				newlocalDiskNumber = i + 1
			}
			logrus.Printf("There are %d local volumes ", newlocalDiskNumber)

			output = runInLinux("sh deletedisk.sh")
			logrus.Printf("delete disk : %+v", output)
			gomega.Expect(newlocalDiskNumber).ToNot(gomega.Equal(localDiskNumber))

		})

	})
	ginkgo.Context("test LocalDiskClaim", func() {
		ginkgo.It("Create new LocalDiskClaim", func() {
			nodelist := nodeList()
			createLdc()
			for _, nodes := range nodelist.Items {
				localDiskClaim := &ldv1.LocalDiskClaim{}
				localDiskClaimKey := k8sclient.ObjectKey{
					Name:      "localdiskclaim-" + nodes.Name,
					Namespace: "kube-system",
				}
				err := client.Get(ctx, localDiskClaimKey, localDiskClaim)
				if err != nil {
					logrus.Printf("%+v ", err)
					f.ExpectNoError(err)
				}

				gomega.Expect(localDiskClaim.Status.Status).To(gomega.Equal(ldv1.LocalDiskClaimStatusBound))
			}
		})
	})
	ginkgo.Context("Clean up the environment", func() {
		ginkgo.It("Clean helm & crd", func() {
			uninstallHelm()
		})
	})

})
