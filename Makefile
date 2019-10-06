.PHONY: all
all: build

.PHONY: build-amd64-default
build-amd64-default:
	docker build .  --platform=linux/amd64 -f docker/Dockerfile.amd64 -t robertme/cec2mqtt-amd64:${VERSION}

.PHONY: build-arm32v7-default
build-arm32v7-default: crosscompile
	docker build .  --platform=linux/arm32/v7 -f docker/Dockerfile.arm32v7 -t robertme/cec2mqtt-arm32v7:${VERSION}

.PHONY: build-arm64v8-default
build-arm64v8-default: crosscompile
	docker build .  --platform=linux/arm64/v8 -f docker/Dockerfile.arm64v8 -t robertme/cec2mqtt-arm64v8:${VERSION}

.PHONE: build-default
build-default: build-amd64-default build-arm32v7-default build-arm64v8-default

.PHONY: push-amd64-default
push-amd64-default: build-amd64-default
	docker push robertme/cec2mqtt-amd64:${VERSION}

.PHONY: push-arm32v7-default
push-arm32v7-default: build-arm32v7-default
	docker push robertme/cec2mqtt-arm32v7:${VERSION}

.PHONY: push-arm64v8-default
push-arm64v8-default: build-arm64v8-default
	docker push robertme/cec2mqtt-arm64v8:${VERSION}

.PHONY: push-default
push-default: export DOCKER_CLI_EXPERIMENTAL = enabled
push-default: push-amd64-default push-arm32v7-default push-arm64v8-default
	docker manifest create robertme/cec2mqtt:${VERSION} robertme/cec2mqtt-amd64:${VERSION} robertme/cec2mqtt-arm32v7:${VERSION} robertme/cec2mqtt-arm64v8:${VERSION}
	docker manifest annotate robertme/cec2mqtt:${VERSION} robertme/cec2mqtt-arm32v7:${VERSION} --os linux --arch arm --variant 7
	docker manifest annotate robertme/cec2mqtt:${VERSION} robertme/cec2mqtt-arm64v8:${VERSION} --os linux --arch arm64 --variant 8
	docker manifest inspect robertme/cec2mqtt:${VERSION}
	docker manifest push --purge robertme/cec2mqtt:${VERSION}

.PHONY: release-default
release-default: push-default
	# Tag
	docker tag robertme/cec2mqtt-amd64:${VERSION} robertme/cec2mqtt-amd64:latest
	docker tag robertme/cec2mqtt-arm32v7:${VERSION} robertme/cec2mqtt-arm32v7:latest
	docker tag robertme/cec2mqtt-arm32v8:${VERSION} robertme/cec2mqtt-arm32v8:latest

	# Push
	docker push robertme/cec2mqtt-amd64:latest
	docker push robertme/cec2mqtt-arm32v7:latest
	docker push robertme/cec2mqtt-arm32v8:latest

	# Create manifest
	docker manifest create robertme/cec2mqtt:latest robertme/cec2mqtt-amd64:latest robertme/cec2mqtt-arm32v7:latest robertme/cec2mqtt-arm64v8:latest
	docker manifest annotate robertme/cec2mqtt:latest robertme/cec2mqtt-arm32v7:latest --os linux --arch arm --variant 7
	docker manifest annotate robertme/cec2mqtt:latest robertme/cec2mqtt-arm64v8:latest --os linux --arch arm64 --variant 8
	docker manifest inspect robertme/cec2mqtt:latest
	docker manifest push --purge robertme/cec2mqtt:latest

.PHONY: build-arm32v7-rpi
build-arm32v7-rpi: crosscompile
	docker build .  --platform=linux/arm32/v7 --build-arg PLATFORM=rpi -f docker/Dockerfile.arm32v7 -t robertme/cec2mqtt-arm32v7-rpi:${VERSION}

.PHONY: build-arm64v8-rpi
build-arm64v8-rpi: crosscompile
	docker build .  --platform=linux/arm64/v8 --build-arg PLATFORM=rpi -f docker/Dockerfile.arm64v8 -t robertme/cec2mqtt-arm64v8-rpi:${VERSION}

.PHONY: build-rpi
build-rpi: build-arm32v7-rpi build-arm64v8-rpi

.PHONY: push-rpi
push-rpi: export DOCKER_CLI_EXPERIMENTAL=enabled
push-rpi: build-rpi
	# Images need to be pushed before manifest can be created
	docker push robertme/cec2mqtt-arm32v7-rpi:${VERSION}
	docker push robertme/cec2mqtt-arm64v8-rpi:${VERSION}

	docker manifest create robertme/cec2mqtt:rpi-${VERSION} robertme/cec2mqtt-arm32v7-rpi:${VERSION} robertme/cec2mqtt-arm64v8-rpi:${VERSION}
	docker manifest annotate robertme/cec2mqtt:rpi-${VERSION} robertme/cec2mqtt-arm32v7-rpi:${VERSION} --os linux --arch arm --variant 7
	docker manifest annotate robertme/cec2mqtt:rpi-${VERSION} robertme/cec2mqtt-arm64v8-rpi:${VERSION} --os linux --arch arm64 --variant 8
	docker manifest inspect robertme/cec2mqtt:rpi-${VERSION}
	docker manifest push --purge robertme/cec2mqtt:rpi-${VERSION}

.PHONY: release-rpi
release-rpi: push-rpi
	# Tag
	docker tag robertme/cec2mqtt-arm32v7-rpi:${VERSION} robertme/cec2mqtt-arm32v7-rpi:latest
	docker tag robertme/cec2mqtt-arm32v8-rpi:${VERSION} robertme/cec2mqtt-arm32v8-rpi:latest

	# Push
	docker push robertme/cec2mqtt-arm32v7-rpi:latest
	docker push robertme/cec2mqtt-arm32v8-rpi:latest

	# Create manifest
	docker manifest create robertme/cec2mqtt:rpi robertme/cec2mqtt-arm32v7-rpi:latest robertme/cec2mqtt-:latest
	docker manifest annotate robertme/cec2mqtt:rpi robertme/cec2mqtt-arm32v7-rpi:latest --os linux --arch arm --variant 7
	docker manifest annotate robertme/cec2mqtt:rpi robertme/cec2mqtt-arm64v8-rpi:latest --os linux --arch arm64 --variant 8
	docker manifest inspect robertme/cec2mqtt:rpi
	docker manifest push --purge robertme/cec2mqtt:rpi

.PHONY: build
build: build-default build-rpi

.PHONY: push
push: push-default push-rpi

.PHONY: release
release: push

.PHONY: crosscompile
crosscompile:
	docker run --rm --privileged multiarch/qemu-user-static:register --reset
