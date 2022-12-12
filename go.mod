module github.com/bigkevmcd/tekton-polling-operator

go 1.14

require (
	cloud.google.com/go/container v1.9.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.13.12 // indirect
	github.com/Azure/go-autorest/autorest v0.11.27 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.20 // indirect
	github.com/aws/aws-sdk-go v1.43.16 // indirect
	github.com/benbjohnson/clock v1.3.0 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/emicklei/go-restful/v3 v3.9.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-logr/logr v1.2.3
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/golang/protobuf v1.5.2
	github.com/google/cel-go v0.12.5
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/go-cmp v0.5.9
	github.com/google/go-containerregistry v0.8.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2 // indirect
	github.com/onsi/ginkgo/v2 v2.2.0 // indirect
	github.com/onsi/gomega v1.20.2 // indirect
	github.com/prometheus/client_golang v1.13.0 // indirect
	github.com/rogpeppe/go-internal v1.8.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/pipeline v0.23.0
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.23.0 // indirect
	golang.org/x/crypto v0.0.0-20221012134737-56aed061732a // indirect
	golang.org/x/sys v0.0.0-20221013171732-95e765b1cc43 // indirect
	golang.org/x/term v0.0.0-20220919170432-7a66f970e087 // indirect
	golang.org/x/time v0.0.0-20220922220347-f3bd1da661af // indirect
	k8s.io/api v0.25.3
	k8s.io/apiextensions-apiserver v0.25.3 // indirect
	k8s.io/apimachinery v0.25.3
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20221012153701-172d655c2280 // indirect
	k8s.io/utils v0.0.0-20221012122500-cfd413dd9e85 // indirect
	sigs.k8s.io/controller-runtime v0.13.0
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
