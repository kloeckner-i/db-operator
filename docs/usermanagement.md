# User management

## Database Custom resource

When you create a `Database`, db-operator will create a main user, that full access to the created database. You should use this user for you workloads. 

## DbUser Custom Resource
### Intro

> DbUser was just delivered, so you can expect bugs. Feel free to open issues if something is not working as intended

If you happen to need more users on your database, you can use `DbUser` Custom Resource

```
# -- Example
---
apiVersion: "kinda.rocks/v1beta1"
kind: DbUser
metadata:
  name: mysql-readwrite
  name: default
spec:
  secretName: mysql-readwrite-secret
  accessType: readWrite
  databaseRef: mysql-db
```

### How it's created?

When you apply a manifest like in the example above, db-operator will create a new user on a database instance, to which the wished database belongs.  Let's assume, you have a `DbInstance` **mysql-generic-server**, and a `Database` **mysql-db** that is using the **mysql-generic-server** as a server. When you create a user like in  the example, db-operator will try to find a `Database` specified in `spec.databaseRef` and will create a user using it. But since users are resources that are created per server, but not per database, the user will actually exist on the `DbInstance`.

As a username, ${metadata.namespace}-${metadata.name} of the `DbUser` will be used. That means that after creating a resource from the example manifest, you will have a `default-mysql-readwrite` user on your Mysql server. 

Users can only access databases that are deployed to the same namespace, and they, currently, cannot be assigned to multiple databases.

DbInstance creation will trigger a new secret creation as well. You will have a secret in the same namespace, the name of which will be taken from `spec.secretName`, it will contain all the data that is required to connect to the DB

### Access Types

`DbUser` supports two access types:
    - readWrite (SELECT, INSERT, UPDATE, DELETE)
    - readOnly (SELECT)

Read Write user can'r create and drop tables, because actions like this should be done only by the main user (the one created with the database)
