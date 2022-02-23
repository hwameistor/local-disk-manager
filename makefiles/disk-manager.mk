DISK_MANAGER_NAME = local-disk-manager
DISK_MANAGER_IMAGE_DIR = ${PROJECT_SOURCE_CODE_DIR}/images/disk-manager
DISK_MANAGER_BUILD_BIN = ${BINS_DIR}/${DISK_MANAGER_NAME}-run
DISK_MANAGER_BUILD_MAIN = ${CMDS_DIR}/manager/main.go

.PHONY: disk_manager
disk_manager:
	GOARCH=amd64 ${BUILD_ENVS} ${BUILD_CMD} ${BUILD_OPTIONS} -o ${DISK_MANAGER_BUILD_BIN} ${DISK_MANAGER_BUILD_MAIN}

.PHONY: disk_manager_arm64
disk_manager_arm64:
	GOARCH=arm64 ${BUILD_ENVS} ${BUILD_CMD} ${BUILD_OPTIONS} -o ${DISK_MANAGER_BUILD_BIN} ${DISK_MANAGER_BUILD_MAIN}

.PHONY: disk_manager_image
disk_manager_image:
	${DOCKER_MAKE_CMD} make disk_manager
	docker build -t ${DOCKER_REGISTRY}:${IMAGE_TAG} -f ${DISK_MANAGER_IMAGE_DIR}/Dockerfile ${PROJECT_SOURCE_CODE_DIR}
	docker push ${DOCKER_REGISTRY}:${IMAGE_TAG}

.PHONY: disk_manager_release
disk_manager_release:
	# build for amd64 version
	${DOCKER_MAKE_CMD} make disk_manager
	${DOCKER_BUILDX_CMD_AMD64} -t ${RELEASE_DOCKER_REGISTRY}:${RELEASE_TAG}-amd64 -f ${DISK_MANAGER_IMAGE_DIR}/Dockerfile ${PROJECT_SOURCE_CODE_DIR}
	# build for arm64 version
	${DOCKER_MAKE_CMD} make disk_manager_arm64
	${DOCKER_BUILDX_CMD_ARM64} -t ${RELEASE_DOCKER_REGISTRY}:${RELEASE_TAG}-arm64 -f ${DISK_MANAGER_IMAGE_DIR}/Dockerfile ${PROJECT_SOURCE_CODE_DIR}
	# push to a public registry
	${MUILT_ARCH_PUSH_CMD} ${RELEASE_DOCKER_REGISTRY}:${RELEASE_TAG}
