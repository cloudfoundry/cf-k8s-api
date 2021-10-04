# Image URL to use all building/pushing image targets
IMG ?= cloudfoundry/cf-k8s-api:latest

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.DEFAULT_GOAL := test

.PHONY: hnc-install test test-e2e kustomize docker-build fmt vet

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out -shuffle on

test-e2e:
	./scripts/deploy-on-kind.sh e2e
	KUBECONFIG="${HOME}/.kube/e2e.yml" API_SERVER_ROOT=http://localhost ROOT_NAMESPACE=cf-k8s-api-system go test -tags e2e -count 1 ./tests/e2e

run: fmt vet ## Run a controller from your host.
	go run ./main.go

generate: fmt vet
	go generate ./...

deploy: kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/base && $(KUSTOMIZE) edit set image cloudfoundry/cf-k8s-api=${IMG}
	$(KUSTOMIZE) build config/base | kubectl apply -f -

deploy-kind: kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/base && $(KUSTOMIZE) edit set image cloudfoundry/cf-k8s-api=${IMG}
	$(KUSTOMIZE) build config/overlays/kind | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/base | kubectl delete -f -

docker-build: ## Build docker image with the manager.
	docker build -t ${IMG} .

build-reference: kustomize
	cd config/base && $(KUSTOMIZE) edit set image cloudfoundry/cf-k8s-api=${IMG}
	$(KUSTOMIZE) build config/base -o reference/cf-k8s-api.yaml

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.2.0)

HNC_VERSION ?= v0.8.0
HNC_PLATFORM=$(shell go env GOHOSTOS)_$(shell go env GOHOSTARCH)
HNC_BIN=$(shell pwd)/bin
export PATH := $(HNC_BIN):$(PATH)
hnc-install:
	mkdir -p "$(HNC_BIN)"
	curl -L https://github.com/kubernetes-sigs/multi-tenancy/releases/download/hnc-$(HNC_VERSION)/kubectl-hns_$(HNC_PLATFORM) -o "$(HNC_BIN)/kubectl-hns"
	chmod +x "$(HNC_BIN)/kubectl-hns"

	kubectl label ns kube-system hnc.x-k8s.io/excluded-namespace=true --overwrite
	kubectl label ns kube-public hnc.x-k8s.io/excluded-namespace=true --overwrite
	kubectl label ns kube-node-lease hnc.x-k8s.io/excluded-namespace=true --overwrite
	kubectl apply -f https://github.com/kubernetes-sigs/multi-tenancy/releases/download/hnc-$(HNC_VERSION)/hnc-manager.yaml
	kubectl rollout status deployment/hnc-controller-manager -w -n hnc-system
	# Hierarchical namespace controller is quite asynchronous. There is no
	# guarantee that the operations below would succeed on first invocation,
	# so retry until they do.
	echo -n waiting for hns controller to be ready and servicing validating webhooks
	until kubectl create namespace ping-hnc; do echo -n .; sleep 0.5; done
	until kubectl hns create -n ping-hnc ping-hnc-child; do echo -n .; sleep 0.5; done
	until kubectl get namespace ping-hnc-child; do echo -n .; sleep 0.5; done
	until kubectl hns set --allowCascadingDeletion ping-hnc; do echo -n .; sleep 0.5; done
	until kubectl delete namespace ping-hnc --wait=false; do echo -n .; sleep 0.5; done
	echo

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
