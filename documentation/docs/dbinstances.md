---
icon: lucide/server
---
# Database Instances

Currently, DbInstances are more or less a meta resource that connects the operator to a database server.

> There are actually two types of databases: generic and gsql, and the gsql one is supposed to bootstrap an sql instance in GCP, but they are going to be deprecated soon, and hence I don't feel like writing docs for them. The only important thing about them is that you **shouldn't use them**

Here we only talk about the generic instances

## How to configure a DbInstance

You need to have a **PostgreSQL** or a **MySQL** server running, and it has to be accessible by the operator. You also need a user with sufficient permissions, if it's fine in your environment, I would suggest to use an admin user.

Now let's get started:
```yaml
apiVersion: kinda.rocks/v1beta1
kind: DbInstance
metadata:
  name: cloudnative-pg
spec:
  adminSecretRef:
    Name: cnpg17-admin-creds
    Namespace: databases
  backup:
    bucket: <A name of the s3 bucket to be used for backups>
  engine: postgres
  generic:
    hostFrom:
      key: host
      kind: ConfigMap
      name: cloudnative-pg-config
      namespace: databases
    portFrom:
      key: port
      kind: ConfigMap
      name: cloudnative-pg-config
      namespace: databases
  monitoring:
    enabled: false
```

Let's quickly go through the yaml

With `.adminSecretRef` you are pointing the operator to a secret, where the admin credentials are stored.
They must be stored in a following format

```yaml
kind: Secret
data:
  # -- user might be omitted, then the following values will be used
  # -- for PostgreSQL - postgres
  # -- for MySQL      - root
  user: <base64 encoded admin username>
  password: <base64 encoded admin password>
```

With `.engine` you let the operator know, if it should treat a server as a PostgreSQL or a MySQL one. Possible values are `postgres` and `mysql`

Then you need to configure a URL and a port that the operator should try to connect to, there are two options to do that, you can set them directly in the manifest:

```yaml
spec:
  generic:
    host: ${HOST}
    port: ${PORT}
```

Or you can read them from a `ConfigMap` or a `Secret`:

```yaml
spec:
  generic
    hostFrom:
      key: host
      kind: ConfigMap
      name: cloudnative-pg-config
      namespace: databases
    portFrom:
      key: port
      kind: ConfigMap
      name: cloudnative-pg-config
      namespace: databases
```

## Automatic Reconciliation on Resource Changes

The `DbInstance` controller automatically reconciles when referenced `Secrets` or `ConfigMaps` change. The controller adds the label `kinda.rocks/dbinstance-name: <dbinstance-name>` to all Secrets or ConfigMaps that are referenced by a particular `DbInstance`.

**Note:** Since the operator uses a label to track the relationship, there is a **one-to-one relationship** between the `DbInstance` and its referenced resources. Sharing a Secret or ConfigMap between multiple `DbInstance` objects is discouraged, as the labeling won't be consistent.

## Additional configurations

### Extra Grants

To use the `.extraGrants` feature of `Databases`, you need to enabled it on the instance level. To do so, set the `.spec.allowExtraGrants` to `true`

### Allowed Privileges

To use the `.extraPrivileges` feature of `DbUsers`, you also need to enabled the privileges on the instance level. Extra privileges is a list of roles that can be granteed to `DbUsers`. For example:

```yaml
spec:
  generic:
    allowedPrivileges:
      - readOnlyAdmin
      - rds-iam
```

Then you will be able to assigned these roles to DbUsers. The roles are not managed by the operator, they must be already on a server when a user is created.

### Instance Vars

It may happen that you need to share the same variable in the Database/DbUser [templates](templates.md). Let's say you have a **RW** and **RO** urls, and your application need two env variables to connect: one - for actibely writing, and another - only for reading.

The **RW** URL will be available anyway, but how to set a **RO** one.

Without the instance vars you could just create a template with a hardcoded string in it:

```yaml
templates:
  - name: PG_READONLYHOST
    secret: false
    template: "my-read-only-postgres-url.test"
```

Or you can use the instance variables.

```yaml
kind: DbInstance
spec:
  instanceVars:
    PG_READONLYHOST: my-read-only-postgres-url.test
```

And then later use it in a template like that:
```yaml
templates:
  - name: PG_READONLYHOST
    secret: false
    template: '{{ .instanceVar "PG_READONLYHOST" }}'
```

If a value of a variable is changed on the instance, it will be also synced for each Database and User.

### UsingSSLconnection

By default, db-operator use non ssl connection to database instances.
In case you are using public connection, you can enable ssl connection.
To use ssl connection, set `sslConnection.enabled` to `true` in `DbInstance` spec.

#### No SSL

* postgres: disable
* mysql: disabled

```YAML
apiVersion: kinda.rocks/v1beta1
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
apiVersion: kinda.rocks/v1beta1
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
apiVersion: kinda.rocks/v1beta1
kind: DbInstance
metadata:
  name: example-generic
spec:
  sslConnection:
    enabled: true
    skip-verify: false
...
```
