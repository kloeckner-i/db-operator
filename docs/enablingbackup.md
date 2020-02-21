# Enabling regular backup

The DB Operator supports automatic database backups with Cronjob resource in Kubernetes.
It creates database dumps and uploads them to a Google Cloud Storage(GCS) bucket.
Currently it supports only GCS as backup storage.

## Prerequisites

* Google Cloud Storage(GCS) bucket
* Service account with privilege for writing into the bucket

## Limitation

* Only GCS as backend storage
* Only Postgres engine 

## How to enable

Create secret resource with name `google-cloud-storage-bucket-cred` which contains service account json in the same namespace of Database resource.

```YAML
apiVersion: v1
kind: Secret
metadata:
  name: google-cloud-storage-bucket-cred
type: Opaque
data:
  credentials.json: << google service account json base64 encoded >>
```

To set which bucket to use, set bucket name in DbInstance spec.

```YAML
apiVersion: kci.rocks/v1alpha1
kind: DbInstance
metadata:
  name: example-instance
spec:
...
  backup:
    bucket: "<< name of google cloud bucket >>"
```

Change `backup.enable` to **true** in Database custom resource spec and set schedule with cronjob syntax.

```YAML
apiVersion: "kci.rocks/v1alpha1"
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
