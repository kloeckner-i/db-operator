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

build:
	@docker build -t my-db-operator:local .
	@docker save my-db-operator > my-image.tar
	@docker build -t mock-googleapi:local mock/
	@docker save mock-googleapi > mock-google-api.tar

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
	@docker-compose restart sqladmin
	@sleep 2
	@go test -count=1 -tags tests ./... -v -cover
	@docker-compose down

lint:
	@golint ./...

vet:
	@go vet ./...

minisetup: miniup miniimage helm

miniup:
	@minikube start --kubernetes-version=v1.17.17 --cpus 2 --memory 4096

minidown:
	@minikube stop

minidelete:
	@minikube delete

minidashboard:
	@minikube dashboard

miniimage: build
	@minikube image load my-image.tar
	@minikube image load mock-google-api.tar

k3d_setup: k3d_install k3d_image helm

k3d_install:
	@wget -q -O - https://raw.githubusercontent.com/rancher/k3d/main/install.sh | bash
	@k3d cluster create myk3s -i rancher/k3s:v1.17.17-k3s1
	@kubectl get pod

k3d_image: build
	@k3d image import my-image.tar -c myk3s
	@k3d image import mock-google-api.tar -c myk3s
