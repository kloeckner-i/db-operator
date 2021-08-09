# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

.PHONY: all deploy build helm
.ONESHELL: test

help:   ## show this help
	@echo 'usage: make [target] ...'
	@echo ''
	@echo 'targets:'
	@grep -E '^(.+)\:\ .*##\ (.+)' ${MAKEFILE_LIST} | sort | sed 's/:.*##/#/' | column -t -c 2 -s '#'

build: ## build db-operator docker image
	@docker build -t my-db-operator:local .
	@docker save my-db-operator > my-image.tar

helm: ## install helm if not exist and install local chart using helm upgrade --install command
	@helm upgrade --install --create-namespace --namespace operator my-dboperator helm/db-operator -f helm/db-operator/values.yaml -f helm/db-operator/values-local.yaml

helm-lint: ## lint helm manifests
	@helm lint -f helm/db-operator/values.yaml -f helm/db-operator/ci/ci-1.yaml --strict ./helm/db-operator
	@helm lint -f helm/db-instances/values.yaml --strict ./helm/db-instances

addexamples: ## add examples via kubectl create -f examples/
	cd ./examples/; ls | while read line; do kubectl apply -f $$line; done

setup: build helm ## build db-operator image, install helm

deploy:
	@kubectl delete pod -l app=db-operator -n operator &
	watch -n0.2 -c 'kubectl logs -l app=db-operator --all-containers=true -n operator'

update: build deploy ## build db-operator image again and delete running pod

test: ## spin up mysql, postgres containers and run go unit test
	tearDown() {
		docker-compose down
	}
	trap tearDown EXIT
	docker-compose up -d
	docker-compose restart sqladmin
	sleep 10
	go test -count=1 -tags tests ./... -v -cover

lint:
	@golint ./...

vet:
	@go vet ./...

minisetup: miniup miniimage helm

miniup: ## start minikube
	@minikube start --kubernetes-version=v1.19.12 --cpus 2 --memory 4096

minidown: ## stop minikube
	@minikube stop

minidelete: ## delete minikube
	@minikube delete

minidashboard: ## open minikube dashboard
	@minikube dashboard

miniimage: build
	@minikube image load my-image.tar

k3d_setup: k3d_install k3d_image helm ## install microk8s locally and deploy db-operator (only for linux and mac)

k3d_install:
	@wget -q -O - https://raw.githubusercontent.com/rancher/k3d/main/install.sh | bash
	@k3d cluster create myk3s -i rancher/k3s:v1.19.12-k3s1
	@kubectl get pod

k3d_image: build
	@k3d image import my-image.tar -c myk3s

## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
manifests: controller-gen ## generate custom resource definitions
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
generate: controller-gen ## generate supporting code for custom resource types
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

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