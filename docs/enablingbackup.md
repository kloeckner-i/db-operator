# Enabling regular backup

The DB Operator supports automatic database backups with Cronjob resource in Kubernetes.
It creates database dumps and uploads them to a Google Cloud Storage(GCS) bucket.
Currently it supports only GCS as backup storage.

## Prerequisites

* Google Cloud Storage(GCS) bucket
* Service account with privilege for writing into the bucket

## Limitation

* Only GCS as backend storage

## How to enable

Create [Google Service Account](https://cloud.google.com/iam/docs/service-accounts) with `Storage Legacy Bucket Writer` role to the GCS bucket.
In case the DbInstance type is GSQL, `Cloud SQL Client` role need to be assigned to the Service Account additionally.

Create secret resource with name `google-cloud-storage-bucket-cred` which contains service account json in the same namespace of Database resource.

Creating the secret from file can be done by following command. Service Account json key has to be saved with name `credentials.json` first.

```
kubectl create secret generic google-cloud-storage-bucket-cred --from-file ./credentials.json
```

The Secret should look like below.

```YAML
apiVersion: v1
kind: Secret
metadata:
  name: google-cloud-storage-bucket-cred
type: Opaque
data:
  credentials.json: << google service account json base64 encoded >>
```

Configure bucket name in DbInstance spec.

```YAML
apiVersion: kinda.rocks/v1alpha1
kind: DbInstance
metadata:
  name: example-instance
spec:
...
  backup:
    bucket: "<< name of the GCS bucket >>"
```

When the DbInstance type is generic, the host address which will be used by backup job can be set differently by adding `backupHost` in spec. For example, slave can be used for backup.
When it's not specified, backup job will use `host` address by default.

```YAML
apiVersion: kinda.rocks/v1alpha1
kind: DbInstance
metadata:
  name: example-instance
spec:
...
  generic:
    host: master.host
    port: 1234
    backupHost: slave.host
...
```


Change `backup.enable` to **true** in Database custom resource spec and set schedule with cronjob syntax.

```YAML
apiVersion: "kinda.rocks/v1alpha1"
kind: "Database"
metadata:
  name: "example-db"
spec:
...
  backup:
    enable: true
    cron: "0 0 * * *"
```

The DB Operator will create a kubernetes Cronjob in the same namespace of Database to run backup regularly.

The Cronjob needs permission to push the dump file to the GCS bucket. It will use Secret `google-cloud-storage-bucket-cred`.

## Monitoring

For monitoring a backup job, you can define in the db-operator config a general prometheus pushgateway endpoint (`monitoring.promPushGateway`). If monitoring is enabled, this variable is added to the related backup cronjob environment variables as `PROMETHEUS_PUSH_GATEWAY`.