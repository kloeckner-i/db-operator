## 0.x.x -> 1.x.x

### Breaking changes
* Percona type instances won't be supported any longer. Please follow migration guide below before upgrading.


## Migrating Percona type instance to Generic type instance

1. Install proxysql on the cluster.

2. Configure proxysql to have same backend servers like you configured before in `percona.servers`.

Example Percona type instance Custom Resource

```YAML
apiVersion: kci.rocks/v1alpha1
kind: DbInstance
metadata:
  name: percona-instance
spec:
  adminSecretRef:
    Namespace: default
    Name: percona-instance-admin-password
  engine: mysql
  percona:
    servers:
    - host: pxc-0.default.svc.cluster.local
      port: 3306
      maxConn: 100
    - host: pxc-1.default.svc.cluster.local
      port: 3306
      maxConn: 100
    monitorUserSecretRef:
      Namespace: default
      Name: percona-instance-monitoruser-secret
```
In proxysql.cnf
```
mysql_servers =
(
  { address="pxc-0.default.svc.cluster.local", port=3306, hostgroup=10, max_connections=100 },
  { address="pxc-1.default.svc.cluster.local", port=3306, hostgroup=10, max_connections=100 },
)
```

3. Copy monitoring user secrets into proxysql configuration.

In proxysql.cnf
```
mysql_variables=
{
...
    monitor_username="<username from monitorUserSecret>"
    monitor_password="<password from monitorUserSecret>"
...
}
```

4. Create Generic type instance with proxysql as host.

```YAML
apiVersion: kci.rocks/v1alpha1
kind: DbInstance
metadata:
  name: new-generic-instance
spec:
  adminSecretRef:
    Name: <same as previous>
    Namespace: <same as previous>
  engine: <postgres or mysql>
  generic:
    host: <proxysql host>
    port: <proxysql sql port>
```

5. Update `Database` resources to point new generic instance.

```YAML
apiVersion: "kci.rocks/v1alpha1"
kind: "Database"
metadata:
  name: "example-db"
spec:
  instance: # Update here
...
```

6. Make sure `Databases` status are all true and then delete old percona instance.