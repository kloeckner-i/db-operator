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
* [Upgrade guide](docs/upgradeguide.md) - breaking changes and guide for the upgrade

## Helm Chart is migrated!
The repository contains helm charts for db-operator is moved to https://github.com/kloeckner-i/charts
New chart after db-operator > 1.2.7, db-instances > 1.3.0 will be only available in new repository.

### Downloading old charts

Installing older version of charts is still possible.
Check available versions by following command.

```
$ helm repo add kloeckneri-old https://kloeckner-i.github.io/db-operator/
$ helm search repo kloeckneri-old/ --versions
```

## Quickstart

### To install DB Operator with helm:

```
$ helm repo add kloeckneri https://kloeckner-i.github.io/charts/
$ helm install --name my-release kloeckneri/db-operator
```

To see more options of helm values, [see chart repo]([https://github.com/kloeckner-i/charts/tree/main/charts/db-operator])

To see which version is working together check out our [version matrix](https://github.com/kloeckner-i/db-operator/wiki/Version-Matrix).

## Development

#### Prerequisites
* go 1.15+
* docker
* make
* kubectl v1.14+ (< v1.21)
* helm v3.0.2+
* [k3d](https://github.com/rancher/k3d)

To have kubernetes environment locally, you need to install [k3d](https://github.com/rancher/k3d).


#### makefile help

```
addexamples           add examples via kubectl create -f examples/
build                 build db-operator docker image
controller-gen        Download controller-gen locally if necessary.
generate              generate supporting code for custom resource types
help                  show this help
k3d_image             rebuild the docker images and upload into your k3d cluster
k3d_install           install k3d cluster locally
k3d_setup             install k3d and import image to your k3d cluster
k3s_mac_deploy        build image and import image to local lima k8s
k3s_mac_image         import built image to local lima k8s
k3s_mac_lima_create   create local k8s using lima
k3s_mac_lima_start    start local lima k8s
lint                  lint go code
manifests             generate custom resource definitions
test                  run go unit test
vet                   go vet to find issues
```

### Developing locally

#### After code changes

rebuild CRD manifests
```
$ make manifests
```

rebuild local docker image
```
$ make build
```

#### Run local kubernetes
```
$ make k3d_setup
```

or 

```
$ make k3s_mac_lima_create
```

#### Import local docker image

```
$ make k3d_build
```

or
```
$ make k3s_mac_image
```

### Deploy

```
helm upgrade my-release kloeckneri/db-operator --set image.repository=my-db-operator --set image.tag=1.0.0-dev --set image.pullPolicy=IfNotPresent
```

### Run unit test locally

```
$ make test
```
