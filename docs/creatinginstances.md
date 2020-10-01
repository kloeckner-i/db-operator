# Creating DbInstances

## Before start

At least a **DbInstance** is necessary to create databases. DbInstance defines the target server for database creation.

For more details about how it works, check [here](howitworks.md)

## Next
You can use an existing database server or create/use Google Cloud SQL instance to create a **DbInstance**.

* [Using existing database server](#GenericDbInstance)
* [Creating or updating Google Cloud SQL Instance](#GoogleCloudSQLDbInstance)
* [Checking DbInstance status](#CheckingStatus)
* [Using SSL connection](#UsingSSLconnection)

### GenericDbInstance
Using existing database server

#### Prerequisite
* running database server accessible by ip or hostname

Create a new secret containing admin username and password of an instance.
```
kubectl create secret generic example-generic-admin-secret --from-literal=user=<admin user name> --from-literal=password='<admin user password>'
```

Or use existing secret created by stable mysql/postgres helm chart.

Create **DbInstance** custom resource.
```YAML
apiVersion: kci.rocks/v1alpha1
kind: DbInstance
metadata:
  name: example-generic
spec:
  adminSecretRef:
    Name: example-generic-admin-secret
    Namespace: <namespace of secret existing>
  engine: <postgres or mysql>
  generic:
    host: <host address to connect database server>
    port: <port to connect database server>
```

### GoogleCloudSQLDbInstance
Creating or using Google Cloud SQL Instance

#### Prerequisite
* Google Cloud Platform(GCP) project;
* service account json key with **Cloud SQL Admin** role;
* service account json key with **Cloud SQL Client** role;

> **Cloud SQL Admin** credential is used by **operator** for creating/using Google Cloud SQL instances.
> **Cloud SQL Client** credential is used by **cloud proxy** for accessing database. **Cloud proxy** works as an endpoint between pods and Google Cloud SQL instances. **Cloud SQL Client** role has only privileges to connect to Google Cloud SQL instances. The role has only the following permissions.
> * cloudsql.instances.connect
> * cloudsql.instances.get

> It's recommended for security reasons to create separated service accounts, each one for each role.

Create service account on a GCP project (check [Creating and managing service account keys](https://cloud.google.com/iam/docs/creating-managing-service-account-keys))

Upgrade db-operator helm release with service account

```
$ helm upgrade my-release helm/db-operator --set secrets.gsql.admin="<< Service Account Cloud SQL Admin >>" --set secrets.gsql.readonly="<< Service Account Cloud SQL Client >>"
```

Create a configmap containing a Google Cloud SQL configuration, according to its [API specification](https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1beta4/instances#DatabaseInstance)

```YAML
apiVersion: v1
kind: ConfigMap
metadata:
  name: example-gsql-config
data:
  config: |
    {
      "databaseVersion": "POSTGRES_9_6",
      "settings": {
        "tier": "db-f1-micro",
        "availabilityType": "ZONAL",
        "pricingPlan": "PER_USE",
        "replicationType": "SYNCHRONOUS",
        "activationPolicy": "ALWAYS",
        "dataDiskType": "PD_SSD",
        "backupConfiguration": {
          "enabled": false
        },
        "storageAutoResizeLimit": "0",
        "storageAutoResize": true
      },
      "backendType": "SECOND_GEN",
      "region": "europe-west1"
    }
```

Create a secret containing admin username and password of an instance.
```
kubectl create secret generic example-gsql-admin-secret --from-literal=user=<admin user name> --from-literal=password='<admin user password>'
```

Create **DbInstance** custom resource.
```YAML
apiVersion: kci.rocks/v1alpha1
kind: DbInstance
metadata:
  name: example-gsql
spec:
  adminSecretRef:
    Name: example-generic-admin-secret
    Namespace: <namespace of secret existing>
  configmap: example-gsql-config
  engine: <postgres or mysql>
  google:
    instance: dboperator-example-gsql # Cloud SQL Instance resource name in google project
    accessSecret: cloudsql-client-serviceaccount # DB Operator will create secret with this name when database resource is created
```


### CheckingStatus

Check **DbInstance** status
```
kubectl get dbin example-generic
```

The output should be like
```
NAME              PHASE      STATUS
example-generic   Creating   false
```

Possible phases and meanings

| Phase                 | Description                           |
|-------------------    |-----------------------                |
| `Valitating`          | Validate all the necessary fields provided in the resource spec |
| `Creating`            | Create (only google type) or check if the database server is reachable |
| `Broadcasting`        | Trigger `Database` phase cycle if there was an update on `DbInstance` |
| `ProxyCreating`       | Creating Google Cloud Proxy `Deployment` and `Service` to be used as endpoint for connecting to the database (only google type) |
| `Running`             | Backend database server connection checked and ready for database creation |


### UsingSSLconnection

By default, db-operator use non ssl connection to database instances.
In case you are using public connection, you can enable ssl connection.
To use ssl connection, set `sslConnection.enabled` to `true` in `DbInstance` spec.

#### No SSL

* postgres: disable
* mysql: disabled

```YAML
apiVersion: kci.rocks/v1alpha1
kind: DbInstance
metadata:
  name: example-generic
spec:
  sslConnection:
    enabled: false
    skip-verify: false
...
```

#### Always SSL (skip verification)

* postgres: require
* mysql: required

```YAML
apiVersion: kci.rocks/v1alpha1
kind: DbInstance
metadata:
  name: example-generic
spec:
  sslConnection:
    enabled: true
    skip-verify: true
...
```

#### Always SSL (verify that the certificate presented by the server was signed by a trusted CA)

* postgres: verify-ca
* mysql: verify_ca

```YAML
apiVersion: kci.rocks/v1alpha1
kind: DbInstance
metadata:
  name: example-generic
spec:
  sslConnection:
    enabled: true
    skip-verify: false
...
```

> * Do not enable SSL connection with google type instance. It connect via google cloud proxy instead of using public ip.
> * Self-signed certificates with verify option is currently not supported.