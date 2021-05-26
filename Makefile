.PHONY: all deploy build helm
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
	@minikube start --kubernetes-version=v1.17.17 --cpus 2 --memory 4096

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
	@kubectl delete pod -l app=db-operator -n operator &
	watch -n0.2 -c 'kubectl logs -l app=db-operator --all-containers=true -n operator'

update: build deploy

test:
	@docker-compose up -d
	@sleep 2
	@go test -tags tests ./... -v -cover
	@docker-compose down

lint:
	@golint ./...

vet:
	@go vet ./...

microsetup: microup microbuild microhelm

microup:
	@sudo snap install microk8s --classic --channel=1.18/stable
	@sudo microk8s.status --wait-ready
	@sudo microk8s.enable dns registry helm3
	@sudo microk8s.status --wait-ready

microbuild:
	@docker build -t my-db-operator:local .
	@docker save my-db-operator > my-image.tar
	@sudo microk8s ctr image import my-image.tar

microhelm:
	@sudo microk8s kubectl create ns operator --dry-run=client -o yaml | sudo microk8s kubectl apply -f -
	@sudo microk8s helm3 upgrade --install --namespace operator db-operator helm/db-operator -f helm/db-operator/values.yaml -f helm/db-operator/values-local.yaml

k3d_setup: k3d_install k3d_build helm

k3d_install:
	@wget -q -O - https://raw.githubusercontent.com/rancher/k3d/main/install.sh | bash
	@k3d cluster create myk3s -i rancher/k3s:v1.17.17-k3s1
	@kubectl get pod

k3d_build:
	@docker build -t my-db-operator:local .
	@docker save my-db-operator > my-image.tar
	@k3d image import my-image.tar -c myk3s