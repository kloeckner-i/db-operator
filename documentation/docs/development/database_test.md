---
icon: lucide/flask-conical
---

# Database tests

We have a pretty good test coverage of the Database packages. These tests need an accessible database instance to run.

Tests are not performing any clean-up logic, so once tests are executed, you'll have to clean the database up on your own. So it's recommended not to use any 'real' databases.

## Postgres

All the names of postgres-related test functions must start with `TestPostgres`, only then they will be executed as part of Postgres test suites in CI.

### Generic Postgres

All the postgres tests must be successfully executed against the regular self-hosted postgres instance. You can use the official docker image or use `helmfile` to prepare the environment.

First, start a `kind` cluster using the config from the repo:
```sh
make setup-test-e2e
# or without Makefile
kind create cluster --config=kind-config.yaml
```

The config will forward the `NodePort` used by the postgres service, so you can reach it from your machine.

Then set environment variables:

```sh
# cert-manager is not needed
export CERT_MANAGER_INSTALL=false
export POSTGRES_INSTALL=true
export POSTGRES_PORT=30432
export POSTGRES_USER=db-operator
export POSTGRES_PASSWORD=qwertyu9
```

Now you're ready to start the instance

```sh
helmfile apply
```

And run tests once it's up and running
```
go test -count=1 -tags tests -run "TestPostgres" ./... -v
```

After tests are executed, you can remove postgres like that:
```sh
helmfile destroy
```

You can also use any tag found here: <https://hub.docker.com/_/postgres/>

To do so, set the `POSTGRES_VERSION` variable to a tag name, for example `latest` or `18.3`

If you don't want to use `helmfile`, you can point the tests to any server using the following environment variables:

```sh
export POSTGRES_PORT=<Postgres port>
export POSTGRES_HOST=<Postgres host>
export POSTGRES_USER=<Admin username>
export POSTGRES_PASSWORD=<Admin password>
```
### Cloud-provider specific instances

Officially we do not support any cloud-provider integration, but we expect though, that it must be possible to use the DB Operator in any environment.
Different cloud providers have different sets of permissions, so we have scripts that are bootstrapping instances in a way, we think, cloud providers do, but since we don't have any proven information about how it's actually done on their sides, we are rather guessing here.

#### AZ Flexible

You can start an instance that should behave like an AZ Flexible Postgres by adding the following variable to the set that is described for generic instances:

```sh
POSTGRES_SIMULATE_AZ=true
```

Then run `helmfile apply`

#### AWS RDS

To configure a "RDS" instance, set the `POSTGRES_SIMULATE_AWS` variable to `true`, and run `helmfile apply`

Currently, not all the tests will pass with `RDS` configuration.

## MySQL

To test `MySQL` you will also need a running server.

You can use `kind` and `helmfile` to start a `Mariadb` server and run tests against it.

```sh
make setup-test-e2e
# cert-manager is not needed
export CERT_MANAGER_INSTALL=false
export MARIADB_INSTALL=true
export MYSQL_PORT=30306
export MYSQL_PASSWORD=qwertyu9
helmfile apply
```

And run tests once it's up and running
```
go test -count=1 -tags tests -run "TestMysql" ./... -v
```

After tests are executed, you can remove mysql:
```sh
helmfile destroy
```

You can also use any tag found here: <https://hub.docker.com/_/mariadb/>

To do so, set the `MARIADB_VERSION` variable to a tag name, for example `latest` or `11.4`

If you don't want to use `helmfile`, you can point the tests to any server using the following environment variables:

```sh
export MYSQL_PORT=<MySQL port>
export MYSQL_HOST=<MySQL host>
export MYSQL_PASSWORD=<Admin password>
```
