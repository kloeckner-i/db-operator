module github.com/kloeckner-i/db-operator

go 1.13

require (
	bou.ke/monkey v1.0.1
	github.com/GoogleCloudPlatform/cloudsql-proxy v1.23.0
	github.com/go-openapi/spec v0.19.4
	github.com/go-sql-driver/mysql v1.6.0
	github.com/jcelliott/lumber v0.0.0-20160324203708-dd349441af25 // indirect
	github.com/kloeckner-i/can-haz-password v0.1.0
	github.com/lib/pq v1.10.2
	github.com/mitchellh/hashstructure v1.0.0
	github.com/operator-framework/operator-sdk v0.18.0
	github.com/prometheus/client_golang v1.5.1
	github.com/sdomino/scribble v0.0.0-20200707180004-3cc68461d505 // indirect
	github.com/sirupsen/logrus v1.5.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2 // indirect
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c
	google.golang.org/api v0.47.0
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20200121204235-bf4fb3bd569c
	sigs.k8s.io/controller-runtime v0.6.0
)

// Pinned to kubernetes-1.16.2
replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.18.2 // Required by prometheus-operator
)
