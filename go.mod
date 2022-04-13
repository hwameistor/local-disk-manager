module github.com/hwameistor/local-disk-manager

go 1.17

require (
	github.com/hwameistor/local-storage v0.1.6
	github.com/onsi/ginkgo/v2 v2.1.3
	github.com/onsi/gomega v1.17.0
	github.com/operator-framework/operator-sdk v0.18.2
	github.com/pilebones/go-udev v0.9.0
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.18.6
	k8s.io/apiextensions-apiserver v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.6.3
)

require (
	github.com/container-storage-interface/spec v1.3.0
	github.com/kubernetes-csi/csi-lib-utils v0.7.1
	google.golang.org/grpc v1.27.0
	k8s.io/code-generator v0.18.6
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.18.2 // Required by prometheus-operator
)
