# Enabling monitoring (Prometheus exporter)

The DB Operator creates a `Deployment` of a Prometheus exporter to expose database metrics easily. This exporter will execute queries against database and expose metrics at the endpoint.

## Limitation
* Only Postgres engine

In case monitoring is enabled for Mysql databases, this won't be activated and there will be an error logged like `Monitoring: db engine monitoring for mysql not implemented`, as this functionality is not ready yet.

## How to enable

Change `monitoring.enable` to **true** in the Database custom resource spec.

```YAML
apiVersion: "kci.rocks/v1alpha1"
kind: "Database"
metadata:
  name: "example-db"
spec:
...
  monitoring:
    enable: true
```

Then operator creates an exporter `Deployment` with prometheus scrape annotations
```YAML
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-db-pgexporter
spec:
...
  template:
    metadata:
      annotations:
        prometheus.io/port: "60000"
        prometheus.io/scrape: "true"
```
The default exporter container image is `wrouesnel/postgres_exporter:latest`.


## What metrics will be exposed

The queries executed by the exporter are set in helm by the key `config.monitoring.postgres.exporter.customQueries`. They can be changed according to the needs. The default one is the following.

```YAML
pg_stat_statements:
    query: "SELECT userid, pgss.dbid, pgdb.datname, queryid, query, calls, total_time, mean_time, rows FROM pg_stat_statements pgss LEFT JOIN (select oid as dbid, datname from pg_database) as pgdb on pgdb.dbid = pgss.dbid WHERE not queryid isnull ORDER BY mean_time desc limit 20"
    metrics:
        - userid:
            usage: "LABEL"
            description: "User ID"
        - dbid:
            usage: "LABEL"
            description: "database ID"
        - datname:
            usage: "LABEL"
            description: "database NAME"
        - queryid:
            usage: "LABEL"
            description: "Query unique Hash Code"
        - query:
            usage: "LABEL"
            description: "Query class"
        - calls:
            usage: "COUNTER"
            description: "Number of times executed"
        - total_time:
            usage: "COUNTER"
            description: "Total time spent in the statement, in milliseconds"
        - mean_time:
            usage: "GAUGE"
            description: "Mean time spent in the statement, in milliseconds"
        - rows:
            usage: "COUNTER"
            description: "Total number of rows retrieved or affected by the statement"
```



For more query examples see [here](https://github.com/wrouesnel/postgres_exporter/blob/master/queries.yaml).