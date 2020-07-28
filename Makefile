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
	operator-sdk build my-db-operator:local

helm: helm-init
	@helm upgrade --install --namespace operator my-dboperator helm/db-operator -f helm/db-operator/values.yaml -f helm/db-operator/values-local.yaml

helm-init:
	@helm init --upgrade --wait

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
	@sudo microk8s.helm upgrade --install --namespace operator db-operator helm/db-operator -f helm/db-operator/values.yaml -f helm/db-operator/values-local.yaml
