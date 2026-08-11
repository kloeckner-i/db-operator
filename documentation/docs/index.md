---
icon: lucide/package-open
---
# Getting Started

**DB Operator** is a Kubernetes operator for managing **MySQL** and **PostgreSQL** databases through **CRDs**.

This operator does not launch database servers. Instead, it connects to existing ones. Because of this, the database does not need to run inside Kubernetes, the operator can work with any database server that is reachable from within the cluster.

Once connected to a server, you can manage databases and users through Custom Resource Definitions (CRDs). When a user or database is created, the db-operator automatically creates a `Secret` and a `ConfigMap` in the same namespace where the CR is deployed. These resources can then be used by pods to establish connections to the database.

## Quick install

### Helm

The officially supported way to install DB Operator is using a helm chart.

You can find the source code of the helm charts here: <https://github.com/db-operator/charts>

The charts are distributed both as a standard Helm repository and as an OCI artifact.

To install the repo, run the following

```sh
helm repo add db-operator https://db-operator.github.io/charts
helm search repo db-operator
```

OCI artifacts are available under `ghcr.io/db-operator/charts/`

Then install the chart using the helm repo:

```sh
helm install db-operator/db-operator
```

Or using the OCI artifact:

```sh
helm install ghcr.io/db-operator/charts/db-operator
```

### Kustomize

It's also possible to install the operator using the kustomizations created by the kubebuilder. To do so you can run the following:

```sh
git clone https://github.com/db-operator/db-operator.git
cd db-operator
# To install CRDs
task install
# To deploy the operator
task deploy
```

Even though it's possible, this way is not supported and if you have issues with it, you will most probably fight them on your own. You might encounter problems with webhook, and the configuration in general will not be that flexible.

## Image verification

Starting from version 2.26.0, our images are signed using Cosign. To verify the image, run the following command:

```shell
export TAG=<desired operator version>
cosign verify --certificate-identity=https://github.com/db-operator/db-operator/.github/workflows/image-publish.yaml@refs/tags/${TAG} --certificate-oidc-issuer=https://token.actions.githubusercontent.com  ghcr.io/db-operator/db-operator:${TAG}
```

## Usage example

Let's imagine, you have an application that requires a connection to a **Postgres DB**, it receives credentials from the environment variable `POSTGRES_DATASOURCE`, and it needs to be in a following format: `postgresql://${USER}:${PASSWORD}@${HOSTNAME}:${PORT}/${DATABASE}?search_path=myapp`

Your server is available at the following address: `postgres.postgres.svc.cluster.local:5432`
And with the following credentials: `admin:qwertyu9`

### Instance creation

We assume that you already have installed the operator, and now we need to create a `DbInstance` resource to connect the operator to the database. `DbInstance` is a cluster resource, so it will be available from any namespace in the cluster.

You'll need to prepare a secret with credentials like this:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: admin-creds
  namespace: postgres
stringData:
  user: admin
  password: qwertyu9
```

And now you can create a `DbInstance` resource:

```yaml
apiVersion: kinda.rocks/v1beta1
kind: DbInstance
metadata:
  name: postgres
spec
  adminSecretRef:
    Name: admin-creds
    Namespace: postgres
  # Backup is a legacy setting, please set it like this
  # it's going to be removed in the next api version
  backup:
    bucket: ""
  engine: postgres
  generic:
    host: postgre.postgres.svc.cluster.local
    port: "5432"
  monitoring:
    enabled: false
```

To make sure that the operator is now connected to the server, execute `kubectl get dbinstance postgres`, and you should see the following:

```shell
NAME         PHASE     STATUS
postgres     Running   true

```

### Database creation

```yaml
apiVersion: kinda.rocks/v1beta1
kind: Database
metadata:
  name: my-app
  namespace: my-namespace
spec:
  backup:
    enable: false
  credentials:
    templates:
    # - This template will be used by the operator to add POSTGRES_DATASOURCE to the secret.
    - name: POSTGRES_DATASOURCE
      secret: true
      template: '{{ .Protocol }}://{{ .Username }}:{{ .Password }}@{{ .Hostname }}:{{ .Port }}/{{ .Database }}?search_path=my-app'
  deletionProtected: true
  instance: some-postgres
  postgres:
    dropPublicSchema: true
    schemas:
    - my-app
  secretName: my-app-db-creds
```

[Read more about templates here](./templates.md)

> (@allanger): This logic might be changed when the `Database` resource will be upgrade to the version `v1`
After the reconciliation you should be able to find a `ConfigMap` and a `Secret` in the namespace `my-namespace`. They both will be called `my-app-db-creds`, and you can use them to connect your application to the database. No manual interactions are needed, everything is managed within a Kubernetes cluster.

## F.A.Q

### How to add ownerReferences to Secrets and ConfigMaps created by the operator?

DB Operator is designed in a way, that an application should be able to connect to a database, even if the operator was accidentally removed with all the CRDs. That's why by default Secrets and ConfigMaps are just created in the cluster without owner references. But if you would like these resources to be cleaned up after a databases is removed, or you need to see connection between them in ArgoCD, you can set the `.spec.cleanup` to `true`

If ArgoCD is used to manage Databases and the `cleanup` is set to `true`, please make sure that the `PrunePropagationPolicy` is not set to `foreground`, because db-operator is using secrets to understand which Database must be removed, and with the `foreground` policy the secret is removed before the Database, that makes it impossible for the operator to finish the reconciliation.

### How to connect the operator to an existing Database?

DB Operator is reading the secret using the `spec.secretName` entry. If this secret doesn't exist, operator will create it and read data out of it. But if a secret is found, it will try to get items that are required to connect to a database from there.

There are the keys, they a secret must contain:

```
# for PosrgreSQL
POSTGRES_DB: $DATABASE_NAME
POSTGRES_PASSWORD: $PASSWORD
POSTGRES_USER: $USERNAME
# and for MySQL
DB: $DATABASE_NAME
PASSWORD: $PASSWORD
USER: $USER
```

### How to rotate passwords?

To rotate a db password with a help of DB Operator, it's enough to remove/update the secret that is used by a database. You might need to restart pods that are using these secrets to reload the environment.

### Is ARM supported?

At this moment, db-operator can run on arm nodes, but currently we're not providing arm images for backup jobs. So if you have an ARM db-operator installation and you want to have the backup functionality enabled, you will need to create your own docker image for that and update configuration via values:

```YAML
  backup:
    activeDeadlineSeconds: 600  # 10m
    nodeSelector: {}
    postgres:
      image: "${{ image_name }}:${{image_tag}}"
    mysql:
      image: "${{ image_name }}:${{image_tag}}"
```

## Experimental features

Some experimental features are added via annotations, to see the full list of supported annotations, have a look at this file: <https://github.com/db-operator/db-operator/blob/main/pkg/consts/consts.go#L44>
