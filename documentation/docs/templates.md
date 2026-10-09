---
icon: lucide/form
---

# Templates

The **DB Operator** can generate custom templated entries in `Secrets` and `ConfigMaps` using database-related data. This feature uses **Go templates** and can be used in the following way in Databases:

```yaml
---
kind: Database
spec:
  credentials:
    templates:
      - name: USER_AND_PASSWORD
        template: "{{ .Username }}-{{ .Password }}"
        secret: true
```

And in DbUsers:
```yaml
---
kind: DbUser
spec:
  credentials:
    templates:
      - name: USER_AND_PASSWORD
        template: "{{ .Username }}-{{ .Password }}"
        secret: true
```

## Available fields and functions

### Fields
These fields are available for templating:

- **Protocol**: Depends on the db engine. Possible values are `mysql`/`postgresql`
- **Hostname**: The same value as for the db host in the connection configmap
- **Port**: The same value as for the db port in the connection configmap
- **Database**: The same value as for the db name in the creds secret
- **Username**: The same value as for the dab user in the creds secret
- **Password**: The same value as for the password in the creds secret

So to create a full connection string, you could use a template like that:

```
"{{ .Protocol }}://{{ .Username }}:{{ .Password }}@{{ .Hostname }}:{{ .Port }}/{{ .Database }}"
```

### Functions

You can also use one of the available functions to retrieve data directly from the data sources accessible to the operator.

- **Secret**: Query data from the Secret
- **ConfigMap**: Query data from the ConfigMap
- **Query**: Get data directly from the database

#### Secret/ConfigMap

When using `Secret` and `ConfigMap` you can query the previously created secret to template a new one, e.g.:

```yaml
spec:
  credentials:
    templates:
      - name: TMPL_1
        template: "test"
        secret: false
      - name: TMPL_2
        template: "{{ .ConfigMap \"TMPL_1\"}}"
        secret: true
      - name: TMPL_3
        template: "{{ .Secret \"TMPL_2\"}}"
        secret: false
```

This will produce the following entries:

- TMPL_1: test (hardcoded, stored in the ConfigMap)
- TMPL_2: test (read from the ConfigMap and stored in the Secret)
- TMPL_3: test (read from the Secret and stored in the ConfigMap)

#### Query

When using `Query` you need to make sure that you query returns only one value. For example:

```yaml
...
    templates:
      - name: POSTGRES_VERSION
        secret: false
        template: "{{ .Query \"SHOW server_version;\" }}"
```

### InstanceVars

For values shared across multiple databases on the same instance, you can define **instance variables**. Once configured on the `DbInstance`, these variables can also be referenced in `Database` and `DbUser` templates.

```yaml
kind: DbInstance
spec:
  instanceVars:
    TEST_KEY: TEST_VALUE
```

These variables can be accessed by the `Database`/`DbUser` templates like this:

```yaml
...
    templates:
      - name: DB_INSTANCE_VAR
        secret: false
        template: "{{ .InstanceVar 'TEST_KEY' }}"
...
```

Then the secret that is created by the operator should contain the following entry: `DB_INSTANCE_VAR: TEST_VALUE`.

When the value is changed on the instance level, it should also trigger reconciliation of the databases and hence the values should also be updated in the target secrets.
