module github.com/bigkevmcd/tekton-polling-operator

go 1.14

require (
	github.com/go-logr/logr v0.1.0
	github.com/golang/protobuf v1.4.3
	github.com/google/cel-go v0.6.0
	github.com/google/go-cmp v0.5.4
	github.com/operator-framework/operator-sdk v0.17.1
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/pipeline v0.20.0
	github.com/tektoncd/triggers v0.11.2
	go.uber.org/zap v1.16.0
	k8s.io/api v0.18.12
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v12.0.0+incompatible
	knative.dev/pkg v0.0.0-20210107022335-51c72e24c179
	sigs.k8s.io/controller-runtime v0.6.2
)

// Knative deps (release-0.20)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.13.4
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible
)

// Pin k8s deps to 0.18.8
replace (
	k8s.io/api => k8s.io/api v0.18.12
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.12
	k8s.io/client-go => k8s.io/client-go v0.18.12
	k8s.io/code-generator => k8s.io/code-generator v0.18.12
)
