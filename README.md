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

### To install DB Operator with helm:

```
$ helm repo add db-operator https://kloeckner-i.github.io/db-operator/helm/
$ helm install --name my-release db-operator/db-operator
```

To see more options of helm values [here](helm/README.md)

To see which version is working together check out our [version matrix](https://github.com/kloeckner-i/db-operator/wiki/Version-Matrix).

## Development

#### Prerequisites
* go 1.15+
* docker
* make
* kubectl v1.14+ (< v1.21)
* helm v3.0.2+
* [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) or [k3d](https://github.com/rancher/k3d)

To have kubernetes environment locally, you need to install [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) or [microk8s](https://microk8s.io/).


#### makefile help

```
addexamples      add examples via kubectl create -f examples/
build            build db-operator docker image
controller-gen   Download controller-gen locally if necessary.
generate         generate supporting code for custom resource types
helm             install helm if not exist and install local chart using helm upgrade --install command
helm-lint        lint helm manifests
help             show this help
k3d_setup        install microk8s locally and deploy db-operator (only for linux and mac)
manifests        generate custom resource definitions
minidashboard    open minikube dashboard
minidelete       delete minikube
minidown         stop minikube
miniup           start minikube
setup            build db-operator image, install helm
test             spin up mysql, postgres containers and run go unit test
update           build db-operator image again and delete running pod
```

### Developing with Minikube

#### How to run db-operator

```
$ make miniup
$ make setup
```

#### After code changes

rebuild CRD manifests
```
$ make manifests
```

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

### Developing with k3d
#### How to run db-operator
```
$ make k3d_setup
```
#### After code changes

rebuild local docker image
```
$ make k3d_build
```

delete running db-operator and apply new image
```
$ make deploy
```
#### After helm template changes

```
$ make helm
```

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