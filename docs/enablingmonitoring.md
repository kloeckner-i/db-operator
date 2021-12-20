# Enabling monitoring (Prometheus exporter)

Enable Prometheus exporter to expose database metrics easily by setting value for the `db-instance` helm chart. This exporter will execute queries against database and expose metrics at the endpoint.

## Limitation
* Only Postgres engine

In case monitoring is enabled for Mysql databases, this won't be activated.

## How to enable

Set `monitoring.enable` value to **true** in the dbinstance helm value definition.
Example values.yaml for `db-instance` chart.
```YAML
dbinstances:
  db-production:
    engine: postgres
    adminUserSecretName: db-production-admin-secret
    backup:
      bucket: db-postgres-backup-production
    monitoring:
      enabled: true
...
```

Then the helm release creates an exporter [Deployment](../charts/db-instances/templates/postgres_exporter.yaml) with prometheus scrape annotations or service monitors if enabled.
The default exporter container image is `wrouesnel/postgres_exporter:latest`.

## Grafana dashboard

[This file](./dashboard.json) contains a dashboard configured for displaying the exported metrics.

It can be imported in Grafana by using the import dashboard functionality.