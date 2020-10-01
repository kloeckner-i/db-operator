# DB Operator

The DB Operator eases the pain of managing PostgreSQL and MySQL instances for applications running in Kubernetes. The Operator creates databases and make them available in the cluster via Custom Resource. It is designed to support the on demand creation of test environments in CI/CD pipelines.

## Features

DB Operator provides following features:

* Create/Delete databases on the database server running outside/inside Kubernetes by creating `Database` custom resource;
* Create Google Cloud SQL instances by creating `DbInstance` custom resource;
* Automatically create backup `CronJob` with defined schedule (limited feature);

## Documentations
* [How it works](docs/howitworks.md) - a general overview and definitions
* [Creating Instances](docs/creatinginstances.md) - make database instances available for the operator
* [Creating Databases](docs/creatingdatabases.md) - creating databases in those instances
* [Enabling regular Backup](docs/enablingbackup.md) - and schedule cronjob

## Quickstart
TODO: update when public helm repo set up done to not use local helm chart

### To install DB Operator with helm:

```
$ helm repo add myhelmrepo https://kloeckner-i.github.io/db-operator/helm/
$ helm install --name my-release myhelmrepo/db-operator
```

To see more options of helm values [here](helm/README.md)

## Development

#### Prerequisites
* go 1.12+
* docker
* [operator-sdk v0.13.0](https://github.com/operator-framework/operator-sdk/releases/tag/v0.13.0)
* make
* kubectl v1.14+
* helm 2.11+
* [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) or [microk8s](https://microk8s.io/)

To have kubernetes environment locally, you can set [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) or [microk8s](https://microk8s.io/) up.


#### makefile help

```
make miniup: start minikube
make minidown: stop minikube
make minidelete: delete minikube
make minidashboard: open minikube dashboard
make build: build db-operator docker image using operator-sdk
make helm: install helm if not exist and install local chart using helm upgrade --install command
make setup: build db-operator image, install helm
make update: build db-operator image again and delete running pod
make addexamples: kubectl create -f examples/
make test: spin up mysql, postgres containers and run go unit test
make microsetup: install microk8s locally and deploy db-operator (only for linux)
```

### Developing with Minikube

#### How to run db-operator

```
$ make miniup
$ make setup
```

#### After code changes

rebuild local docker image
```
$ make build
```

delete running db-operator and apply new image
```
$ make deploy
```

or both at once
```
$ make update
```

#### After helm template changes

```
$ make helm
```
helm upgrade --install -f {LOCAL CHART DIR}/values-local.yaml {LOCAL CHART DIR}

### Developing with microk8s

* Microk8s supports only linux environment. Non linux user can use microk8s using vm for example multipass. Please find details [here](https://microk8s.io/)

#### How to run db-operator

```
$ make microsetup
```

microsetup is used for our integration test in pipeline.

### Run unit test locally

```
$ make test
```