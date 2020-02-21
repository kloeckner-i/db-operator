minikube_env := $(shell minikube docker-env)

.PHONY: all deploy build

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
	@echo "make microsetup: install microk8s locally and deploy db-operator (only for linux)"

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
	operator-sdk build db-operator:local

helm: helm-init
	@helm upgrade --install --namespace default kci-db-operator helm/kci-db-operator -f helm/kci-db-operator/values.yaml -f helm/kci-db-operator/values-local.yaml -f helm/kci-db-operator/secrets-local.yaml
	@helm upgrade --install --namespace default kci-db-instances helm/kci-db-instances --set operatorNamespace="default" -f helm/kci-db-instances/values.yaml -f helm/kci-db-instances/values-local.yaml

helm-init:
	@helm init --upgrade --wait

helm-lint:
	@helm lint -f helm/kci-db-operator/values.yaml -f helm/kci-db-operator/ci/ci-1.yaml --strict ./helm/kci-db-operator
	@helm lint -f helm/kci-db-instances/values.yaml --strict ./helm/kci-db-instances

addexamples:
	cd ./examples/; ls | while read line; do kubectl apply -f $$line; done

setup: build helm

deploy:
	@kubectl delete pod -l app=kci-db-operator &
	watch -n0.2 -c 'kubectl logs -l app=kci-db-operator --all-containers=true'

update: build deploy

test:
	@docker run --rm --name postgres -e POSTGRES_PASSWORD=test1234 -p 5432:5432 -d postgres:9.6-alpine
	@docker run --rm --name mysql -e MYSQL_ROOT_PASSWORD=test1234 -p 3306:3306 -d mysql:5.7
	@sleep 2
	@go test -tags tests ./... -v -cover
	@docker stop postgres
	@docker stop mysql

vet:
	@go vet ./...

microsetup: microup microhelm microbuild microinstall

microup:
	@sudo snap install microk8s --classic
	@sudo microk8s.status --wait-ready
	@sudo microk8s.enable dns registry helm rbac
	@sudo microk8s.status --wait-ready

microhelm:
	@sudo microk8s.kubectl create -f integration/helm-rbac.yaml
	@sudo microk8s.helm init --service-account tiller

microbuild:
	@docker build -t my-db-operator:local .
	@docker save my-db-operator > my-image.tar
	@sudo microk8s.ctr image import my-image.tar

microinstall:
	@sudo microk8s.helm upgrade --install --namespace operator db-operator helm/kci-db-operator -f helm/kci-db-operator/values.yaml -f helm/kci-db-operator/values-local.yaml