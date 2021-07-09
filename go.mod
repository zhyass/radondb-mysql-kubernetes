module github.com/radondb/radondb-mysql-kubernetes

go 1.16

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/go-ini/ini v1.62.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/iancoleman/strcase v0.0.0-20190422225806-e506e3ef7365
	github.com/imdario/mergo v0.3.11
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.10.5
	github.com/presslabs/controller-util v0.3.0-alpha.2
	github.com/spf13/cobra v1.1.1
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/client-go v0.20.4
	k8s.io/klog/v2 v2.4.0
	sigs.k8s.io/controller-runtime v0.8.3
)
