.PHONY: all build
.ONESHELL: test

SRC = $(shell find . -type f -name '*.go')

ifeq ($(K8S_VERSION),)
K8S_VERSION := v1.22.3
endif

help:   ## show this help
	@echo 'usage: make [target] ...'
	@echo ''
	@echo 'targets:'
	@grep -E '^(.+)\:\ .*##\ (.+)' ${MAKEFILE_LIST} | sort | sed 's/:.*##/#/' | column -t -c 2 -s '#'

build: $(SRC) ## build db-operator docker image
	@docker build -t my-db-operator:v1.0.0-dev .
	@docker save my-db-operator > my-image.tar

addexamples: ## add examples via kubectl create -f examples/
	cd ./examples/; ls | while read line; do kubectl apply -f $$line; done

test: $(SRC) ## run go unit test
	docker-compose down
	docker-compose up -d
	docker-compose restart sqladmin
	sleep 10
	go test -count=1 -tags tests ./... -v -cover
	docker-compose down

lint: $(SRC) ## lint go code
	@go mod tidy
	@gofumpt -l -w $^
	@golangci-lint run ./...

vet: $(SRC)  ## go vet to find issues
	@go vet ./...

k3s_mac_lima_create: ## create local k8s using lima
	limactl start --tty=false ./resources/lima/k3s.yaml

k3s_mac_lima_start: ## start local lima k8s
	limactl start k3s

k3s_mac_deploy: build k3s_mac_image ## build image and import image to local lima k8s

k3s_mac_image: ## import built image to local lima k8s
	limactl copy my-image.tar k3s:/tmp/db.tar
	limactl shell k3s sudo k3s ctr images import /tmp/db.tar
	limactl shell k3s rm -f /tmp/db.tar

k3d_setup: k3d_install k3d_image ## install k3d and import image to your k3d cluster

k3d_install: ## install k3d cluster locally
	@curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
	@k3d cluster create myk3s -i rancher/k3s:$(K8S_VERSION)-k3s1
	@kubectl get pod

k3d_image: build ## rebuild the docker images and upload into your k3d cluster
	@k3d image import my-image.tar -c myk3s

## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
manifests: controller-gen ## generate custom resource definitions
	$(CONTROLLER_GEN) crd rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(CONTROLLER_GEN) crd webhook paths="./..." output:crd:artifacts:config=charts/db-operator/crds

## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
generate: controller-gen ## generate supporting code for custom resource types
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
