DOCKER_REGISTRY ?= hwameistor.io/hwameistor
RELEASE_DOCKER_REGISTRY ?= hwameistor.io/hwameistor

GO_VERSION = $(shell go version)
BUILD_TIME = ${shell date +%Y-%m-%dT%H:%M:%SZ}
BUILD_VERSION = ${shell git rev-parse --short "HEAD^{commit}" 2>/dev/null}
BUILD_ENVS = CGO_ENABLED=0 GOOS=linux
BUILD_FLAGS = -X 'main.BUILDVERSION=${BUILD_VERSION}' -X 'main.BUILDTIME=${BUILD_TIME}' -X 'main.GOVERSION=${GO_VERSION}'
BUILD_OPTIONS = -a -mod vendor -installsuffix cgo -ldflags "${BUILD_FLAGS}"

PROJECT_SOURCE_CODE_DIR=$(CURDIR)
BINS_DIR = ${PROJECT_SOURCE_CODE_DIR}/_build
CMDS_DIR = ${PROJECT_SOURCE_CODE_DIR}/cmd
IMAGES_DIR = ${PROJECT_SOURCE_CODE_DIR}/images

BUILD_CMD = go build
OPERATOR_CMD = operator-sdk
RUN_CMD = go run
K8S_CMD = kubectl

BUILDER_NAME = hwameistor/builder
BUILDER_TAG = latest
BUILDER_MOUNT_SRC_DIR = ${PROJECT_SOURCE_CODE_DIR}/../
BUILDER_MOUNT_DST_DIR = /go/src/github.com/HwameiStor
BUILDER_WORKDIR = /go/src/github.com/HwameiStor/local-disk-manager

DOCKER_SOCK_PATH=/var/run/docker.sock
DOCKER_MAKE_CMD = docker run --rm -v ${BUILDER_MOUNT_SRC_DIR}:${BUILDER_MOUNT_DST_DIR} -v ${DOCKER_SOCK_PATH}:${DOCKER_SOCK_PATH} -w ${BUILDER_WORKDIR} -i ${BUILDER_NAME}:${BUILDER_TAG}
DOCKER_DEBUG_CMD = docker run --rm -v ${BUILDER_MOUNT_SRC_DIR}:${BUILDER_MOUNT_DST_DIR} -v ${DOCKER_SOCK_PATH}:${DOCKER_SOCK_PATH} -w ${BUILDER_WORKDIR} -it ${BUILDER_NAME}:${BUILDER_TAG}
DOCKER_BUILDX_CMD_AMD64 = DOCKER_CLI_EXPERIMENTAL=enabled docker buildx build --platform=linux/amd64 -o type=docker
DOCKER_BUILDX_CMD_ARM64 = DOCKER_CLI_EXPERIMENTAL=enabled docker buildx build --platform=linux/arm64 -o type=docker
MUILT_ARCH_PUSH_CMD = ${PROJECT_SOURCE_CODE_DIR}/docker-push-with-multi-arch.sh

CLUSTER_CRD_DIR = ${PROJECT_SOURCE_CODE_DIR}/deploy/crds

# image_tag/release_tag will be applied to all the images
IMAGE_TAG ?= 99.9-dev
RELEASE_TAG ?= $(shell tagged="$$(git describe --tags --match='v*' --abbrev=0 2> /dev/null)"; latestCommit="$$(git rev-list -n1 HEAD)"; if [ "$$tagged" ] && [ "${latestCommit}" = "$$(git rev-list -n1 $$tagged)" ]; then echo $$tagged; else echo dev-"$${latestCommit:0-7}"; fi)
include ./makefiles/disk-manager.mk

.PHONY: builder
builder:
	docker build -t ${BUILDER_NAME}:${BUILDER_TAG} -f images/builder/Dockerfile .
	docker push ${BUILDER_NAME}:${BUILDER_TAG}

.PHONY: debug
debug:
	${DOCKER_DEBUG_CMD} ash

.PHONY: install-crd
install-crd: gen-code
	${K8S_CMD} apply -f ${CLUSTER_CRD_DIR}

.PHONY: uninstall-crd
uninstall-crd:
	${K8S_CMD} delete -f ${CLUSTER_CRD_DIR}

.PHONY: _gen-code
_gen-code:
	${OPERATOR_CMD} generate k8s
	${OPERATOR_CMD} generate crds

.PHONY: gen-code
gen-code:
	${DOCKER_MAKE_CMD} make _gen-code

.PHONY: vendor
vendor:
	${DOCKER_MAKE_CMD} go mod tidy
	${DOCKER_MAKE_CMD} go mod vendor

.PHONY: clean
clean:
	go clean -r -x
	rm -rf ${BINS_DIR}
	docker container prune -f
	docker rmi -f $(shell docker images -f dangling=true -qa)

.PHONY: all
all: gen-code disk_manager_image

.PHONY: release
release: vendor disk_manager_release 
