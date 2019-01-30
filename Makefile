BIN_DIR=_output/bin
IMAGE = admission-controller-server
TAG = 1.0

all: controllers scheduler cli admission-controller

init:
	mkdir -p ${BIN_DIR}

controllers:
	go build -o ${BIN_DIR}/vk-controllers ./cmd/controllers

scheduler:
	go build -o ${BIN_DIR}/vk-scheduler ./cmd/scheduler

cli:
	go build -o ${BIN_DIR}/vkctl ./cmd/cli

admission-controller:
	go build -o ${BIN_DIR}/ad-controller ./cmd/admission-controller

rel-admission-controller:
	CGO_ENABLED=0 go build -a -installsuffix cgo -o  ${BIN_DIR}/ad-controller ./cmd/admission-controller

admission-images: rel-admission-controller
	cp ${BIN_DIR}/ad-controller ./cmd/admission-controller/
	docker build --no-cache -t $(IMAGE):$(TAG) ./cmd/admission-controller

generate-code:
	go build -o ${BIN_DIR}/deepcopy-gen ./cmd/deepcopy-gen/
	${BIN_DIR}/deepcopy-gen -i ./pkg/apis/batch/v1alpha1/ -O zz_generated.deepcopy
	${BIN_DIR}/deepcopy-gen -i ./pkg/apis/bus/v1alpha1/ -O zz_generated.deepcopy

e2e-test:
	./hack/run-e2e.sh

clean:
	rm -rf _output/
