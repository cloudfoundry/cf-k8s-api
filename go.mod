module code.cloudfoundry.org/cf-k8s-api

go 1.16

require (
	code.cloudfoundry.org/cf-k8s-controllers v0.0.0-20211007182811-671a78a1a8cb
	github.com/buildpacks/pack v0.21.1
	github.com/form3tech-oss/jwt-go v3.2.3+incompatible
	github.com/go-http-utils/headers v0.0.0-20181008091004-fed159eddc2a
	github.com/go-logr/logr v0.4.0
	github.com/go-playground/locales v0.14.0
	github.com/go-playground/universal-translator v0.18.0
	github.com/go-playground/validator/v10 v10.9.0
	github.com/google/go-containerregistry v0.6.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/go-uuid v1.0.2
	github.com/matt-royal/biloba v0.2.1
	github.com/maxbrunsfeld/counterfeiter/v6 v6.4.1
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.16.0
	github.com/pivotal/kpack v0.3.1
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	sigs.k8s.io/controller-runtime v0.10.2
	sigs.k8s.io/controller-tools v0.7.0
	sigs.k8s.io/hierarchical-namespaces v0.0.0-20210827200453-b03328e734e6
)
