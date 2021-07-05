# Configuration

DB operator configuration with default values.

```YAML
# DbInstance configuration
instance:
  google:
    # clientSecretName is the kubernetes secret name containing service account json key with Cloud SQL Client role
    # this will be used by cloud sql proxy for accessing database
    clientSecretName: "cloudsql-readonly-serviceaccount"
    proxy:
      nodeSelector: {}
      image: kloeckneri/db-auth-gateway:0.1.7
  generic: {}
backup:
  nodeSelector: {}
  postgres:
    image: kloeckneri/pgdump-gcs:latest
  mysql: {}
monitoring:
  # append as an ENV variable "PROMETHEUS_PUSH_GATEWAY" to the backup cronjob
  promPushGateway: ""
  nodeSelector: {}
  postgres:
    image: wrouesnel/postgres_exporter:latest
    queries: |-
      pg_stat_statements:
        metrics:
        - userid:
            description: User ID
            usage: LABEL
        - dbid:
            description: database ID
            usage: LABEL
        - datname:
            description: database NAME
            usage: LABEL
        - queryid:
            description: Query unique Hash Code
            usage: LABEL
        - query:
            description: Query class
            usage: LABEL
        - calls:
            description: Number of times executed
            usage: COUNTER
        - total_time:
            description: Total time spent in the statement, in milliseconds
            usage: COUNTER
        - mean_time:
            description: Mean time spent in the statement, in milliseconds
            usage: GAUGE
        - rows:
            description: Total number of rows retrieved or affected by the statement
            usage: COUNTER
        query: SELECT userid, pgss.dbid, pgdb.datname, queryid, query, calls, total_time,
          mean_time, rows FROM pg_stat_statements pgss LEFT JOIN (select oid as dbid, datname
          from pg_database) as pgdb on pgdb.dbid = pgss.dbid WHERE not queryid isnull ORDER
          BY mean_time desc limit 20
  mysql: {}
```