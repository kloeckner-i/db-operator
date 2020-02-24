# db-operator
DB Operator is Kubernetes operator 

## Prerequisites
* Kubernetes 1.13+
* Helm 2.11+

## Configuring helm client
```
$ helm repo add myhelmrepo https://kloeckner-i.github.io/db-operator/helm/
```
Test the helm chart repository
```
$ helm search db-operator
```

## Installing Chart
To install the chart with the release name my-release:
```
$ helm install --name my-release myhelmrepo/db-operator
```
The command deploys DB Operator on Kubernetes with default configuration. For the configuration options see details [Parameters](#Parameters)

## Adding new charts to an existing repository
```
helm repo index --url https://kloeckner-i.github.io/db-operator/helm/ --merge index.yaml .
```

## Uninstalling Chart
To uninstall the `my-release` deployment:
```
$ helm delete my-release
```

## Parameters

The following table lists the configurable parameters of the db-operator chart and their default values.

| Parameter             | Description                           | Default                   |
|-------------------    |-----------------------                |---------------            |
| `appVersion`          | Application Version (DB Operator)     | TODO                      |
| `image.repository`    | Container image name                  | `kloeckneri/db-operator`  |
| `image.tag`           | Container image tag                   | `latest`                  |
| `image.pullPolicy`    | Container pull policy                 | `Always`                  |
| `imagePullSecrets`    | Reference to secret to be used when pulling images | "" |
| `rbac.create`         | Create RBAC resources                 | `true`                    |
| `serviceAccount.create` | If `true`, create a new service account | `true`                |
| `resources`           | CPU/memory resource requests/limits   | `{}`                      |
| `nodeSelector`        | Node labels for pod assignment        | `{}`                      |
| `affinity`            | Node affinity for pod assignment      | `{}`                      |
| `annotations`         | Annotations to add to the db-operator pod | `{}`                  |
| `config.instance.google.proxy.image` | Container image of google cloud proxy | `gcr.io/cloudsql-docker/gce-proxy:1.11` |
| `config.instance.google.proxy.nodeSelector` | Node labels for google cloud proxy pod assignment | `{}` |
| `config.backup.nodeSelector` | Node labels for backup pod assignment | `{}` |
| `config.backup.postgres.image` | Container image of backup cronjob (only for postgres databases) | `kloeckneri/pgdump-gcs:latest` |
| `config.monitoring.nodeSelector` | Node labels for monitoring pod assignment | `{}` |
| `config.monitoring.postgres.image` | Container image of prometheus exporter (only for postgres databases) | `wrouesnel/postgres_exporter:latest` |
| `config.monitoring.postgres.queries` | Queries executed by prometheus exporter (only for postgres databases) | see `values.yaml` for defaults |
| `secrets.gsql.admin`  |  Service account json used by operator to create Cloud SQL instance in GCE(**Cloud SQL Admin**) | `{}` |
| `secrets.gsql.readonly`   |  Service account json will be used by application to access database Cloud SQL in GCE(**Cloud SQL Client** role) | `{}` |