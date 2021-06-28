module github.com/kloeckner-i/db-operator

go 1.15

require (
	bou.ke/monkey v1.0.2
	github.com/GoogleCloudPlatform/cloudsql-proxy v1.23.0
	github.com/go-logr/logr v0.3.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/kloeckner-i/can-haz-password v0.1.0
	github.com/lib/pq v1.10.2
	github.com/mitchellh/hashstructure v1.1.0
	github.com/prometheus/client_golang v1.7.1
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.7.0
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c
	google.golang.org/api v0.47.0
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.19.12
	k8s.io/apimachinery v0.19.12
	k8s.io/client-go v0.19.12
	sigs.k8s.io/controller-runtime v0.7.2
)
