# Database Backups

Current backup implementation is not considered stable and will probably be changed in the future, but it can be already used to take database snapshots and upload them to a storage, that is compatible with rclone.

## Prepare the operator

First images for the backup jobs must be specified in the operator config. Currently it's only possible to set a single image per DbInstance type.

When installing with helm, default values would be:

```
postgres:
  image: "ghcr.io/db-operator/pgdump-rclone:<Version>"
mysql:
  image: "ghcr.io/db-operator/mydump-rclone:<Version>"

```

You can find more info about how to use those containers in corresponding git repositories:
  - https://github.com/db-operator/mydump-rclone
  - https://github.com/db-operator/pgdump-rclone


## How to configure an Instance

The bucket name (if not s3 it can also simply mean a folder) is set on the db instance level

```yaml
kind: DbInstance
spec:
  backup:
    bucket: <Bucket name>
```

## How to add backups to a Database

Once you set `.spec.backup.enable` to `true`, DB Operator will create a cronjob in the same namespace as the database.

It's possible to provide a secret that would be used as environment for the backup pod.

Default containers are using `rclone` that is hardcoded to use a backend with the name `storage`, so to configure them using environment variables, you'll need to create a secret like that:

```
apiVersion: v1
kind: Secret
metadata:
  name: backup-creds
stringData:
  RCLONE_CONFIG_STORAGE_TYPE: s3
  RCLONE_CONFIG_STORAGE_ACCESS_KEY_ID: <Access token>
  RCLONE_CONFIG_STORAGE_SECRET_ACCESS_KEY: <Secret Token>
  RCLONE_CONFIG_STORAGE_ENDPOINT: <Storage URL>
  RCLONE_CONFIG_STORAGE_FORCE_PATH_STYLE: "true"
  RCLONE_CONFIG_STORAGE_PROVIDER: Other

```

And then you can use it in the `Database` resource:

```yaml
apiVersion: kinda.rocks/v1beta1
kind: Database
metadata:
  name: db-to-backup
spec:
  backup:
    cron: 0 0 * * *
    enable: true
    envFromSecret: backup-creds
```


The `Secret` must be in the same namespace as the corresponding `Database`.
