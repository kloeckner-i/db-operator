.PHONY: all deploy build helm
detected_OS := $(shell uname -s)

ifeq ($(detected_OS),Darwin)
PROJECT_DIR=/db-operator
else ifeq ($(detected_OS),Linux)
PROJECT_DIR=.
else
PROJECT_DIR=.
endif

all: help

help:
	@echo "make miniup: start minikube"
	@echo "make minidown: stop minikube"
	@echo "make minidelete: delete minikube"
	@echo "make minidashboard: open minikube dashboard"
	@echo "make build: build db-operator docker image using operator-sdk"
	@echo "make helm: install helm if not exist and install local chart using helm upgrade --install command"
	@echo "make setup: build db-operator image, install helm"
	@echo "make update: build db-operator image again and delete running pod"
	@echo "make addexamples: kubectl create -f examples/"
	@echo "make test: spin up mysql, postgres containers and run go unit test"
	@echo "make microsetup: install microk8s locally and deploy db-operator (only for linux and mac)"

miniup:
	@minikube start --cpus 2 --memory 4096

minidown:
	@minikube stop

minidelete:
	@minikube delete

minidashboard:
	@minikube dashboard

build:
	@eval $$(minikube docker-env) ;\
	operator-sdk build my-db-operator:local

helm:
	@helm upgrade --install --create-namespace --namespace operator my-dboperator helm/db-operator -f helm/db-operator/values.yaml -f helm/db-operator/values-local.yaml

helm-lint:
	@helm lint -f helm/db-operator/values.yaml -f helm/db-operator/ci/ci-1.yaml --strict ./helm/db-operator
	@helm lint -f helm/db-instances/values.yaml --strict ./helm/db-instances

addexamples:
	cd ./examples/; ls | while read line; do kubectl apply -f $$line; done

setup: build helm

deploy:
	@kubectl delete pod -l app=db-operator &
	watch -n0.2 -c 'kubectl logs -l app=db-operator --all-containers=true'

update: build deploy

test:
	@docker run --rm --name postgres -e POSTGRES_PASSWORD=test1234 -p 5432:5432 -d postgres:11-alpine
	@docker run --rm --name mysql -e MYSQL_ROOT_PASSWORD=test1234 -p 3306:3306 -d mysql:5.7
	@sleep 2
	@go test -tags tests ./... -v -cover
	@docker stop postgres
	@docker stop mysql

lint:
	@golint ./...

vet:
	@go vet ./...

microsetup: microinstall microup microbuild microhelm

microinstall:
	@$(MAKE) microinstall_$(detected_OS)

microinstall_Darwin: # Mac OS X
	@brew install ubuntu/microk8s/microk8s
	@brew install --cask multipass
	@microk8s install
	@multipass mount $(PWD) microk8s-vm:$(PROJECT_DIR)

microinstall_Linux:
	@sudo snap install microk8s --classic --channel=1.21/stable

microup:
	@sudo microk8s status --wait-ready
	@sudo microk8s enable dns helm3
	@sudo microk8s status --wait-ready

microbuild:
	@docker build -t my-db-operator:local .
	@docker save my-db-operator > my-image.tar
	@sudo microk8s ctr image import $(PROJECT_DIR)/my-image.tar

microhelm:
	@sudo microk8s kubectl create ns operator --dry-run=client -o yaml | sudo microk8s kubectl apply -f -
	@sudo microk8s helm3 upgrade --install --namespace operator db-operator $(PROJECT_DIR)/helm/db-operator -f $(PROJECT_DIR)/helm/db-operator/values.yaml -f $(PROJECT_DIR)/helm/db-operator/values-local.yaml

