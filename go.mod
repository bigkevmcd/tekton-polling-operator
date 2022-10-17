module github.com/bigkevmcd/tekton-polling-operator

go 1.14

require (
	cloud.google.com/go/container v1.5.0 // indirect
	cloud.google.com/go/monitoring v1.6.0 // indirect
	cloud.google.com/go/trace v1.2.0 // indirect
	github.com/Azure/go-autorest/autorest v0.11.20 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.15 // indirect
	github.com/aws/aws-sdk-go v1.34.9 // indirect
	github.com/go-logr/logr v1.2.2
	github.com/golang/protobuf v1.5.2
	github.com/google/cel-go v0.10.1
	github.com/google/go-cmp v0.5.8
	github.com/google/go-containerregistry v0.8.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/onsi/ginkgo/v2 v2.1.3 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.1 // indirect
	github.com/tektoncd/pipeline v0.23.0
	golang.org/x/crypto v0.0.0-20220408190544-5352b0902921 // indirect
	golang.org/x/tools v0.1.11 // indirect
	k8s.io/api v0.24.2
	k8s.io/apimachinery v0.24.2
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.12.2
)

// Knative deps (release-0.20)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.13.4
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible
	github.com/go-logr/logr => github.com/go-logr/logr v0.3.0
	github.com/go-logr/zapr => github.com/go-logr/zapr v0.3.0
	github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.10.0
	golang.org/x/text => golang.org/x/text v0.3.3 // Required to fix CVE-2020-14040
	k8s.io/api => k8s.io/api v0.19.7
	k8s.io/client-go => k8s.io/client-go v0.19.7
)
