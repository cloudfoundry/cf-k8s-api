//go:build e2e
// +build e2e

package e2e_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"code.cloudfoundry.org/cf-k8s-api/repositories"
	"github.com/hashicorp/go-uuid"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	controllerruntime "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	hnsv1alpha2 "sigs.k8s.io/hierarchical-namespaces/api/v1alpha2"
)

type hierarchicalNamespace struct {
	label         string
	generatedName string
	createdAt     string
	uid           string
	children      []hierarchicalNamespace
}

var (
	suite             spec.Suite
	testServerAddress string
	g                 *WithT
	k8sClient         client.Client
	rootNamespace     string
	apiServerRoot     string
)

func Suite() spec.Suite {
	if suite == nil {
		suite = spec.New("E2E Tests")
	}

	return suite
}

func SuiteDescribe(desc string, f func(t *testing.T, when spec.G, it spec.S)) bool {
	return Suite()(desc, f)
}

func TestSuite(t *testing.T) {
	g = NewWithT(t)

	beforeSuite()
	defer afterSuite()

	suite.Run(t)
}

func beforeSuite() {
	apiServerRoot = mustHaveEnv("API_SERVER_ROOT")

	logf.SetLogger(zap.New(zap.WriteTo(os.Stderr), zap.UseDevMode(true)))

	hnsv1alpha2.AddToScheme(scheme.Scheme)

	config, err := controllerruntime.GetConfig()
	g.Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(config, client.Options{Scheme: scheme.Scheme})
	g.Expect(err).NotTo(HaveOccurred())

	rootNamespace = mustHaveEnv("ROOT_NAMESPACE")
	ensureServerIsUp()
}

func afterSuite() {
}

func mustHaveEnv(key string) string {
	val, ok := os.LookupEnv(key)
	g.Expect(ok).To(BeTrue(), "must set env var %q", key)

	return val
}

func ensureServerIsUp() {
	g.Eventually(func() (int, error) {
		resp, err := http.Get(apiServerRoot)
		if err != nil {
			return 0, err
		}

		resp.Body.Close()

		return resp.StatusCode, nil
	}, "30s").Should(Equal(http.StatusOK), "API Server at %s was not running after 30 seconds", apiServerRoot)
}

func generateGUID(prefix string) string {
	guid, err := uuid.GenerateUUID()
	g.Expect(err).NotTo(HaveOccurred())

	return fmt.Sprintf("%s-%s", prefix, guid[:6])
}

func waitForSubnamespaceAnchor(parent, name string) {
	g.Eventually(func() (bool, error) {
		anchor := &hnsv1alpha2.SubnamespaceAnchor{}
		err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: parent, Name: name}, anchor)
		if err != nil {
			return false, err
		}

		return anchor.Status.State == hnsv1alpha2.Ok, nil
	}, "30s").Should(BeTrue())
}

func waitForNamespaceDeletion(ns string) {
	g.Eventually(func() (bool, error) {
		err := k8sClient.Get(context.Background(), client.ObjectKey{Name: ns}, &corev1.Namespace{})
		if errors.IsNotFound(err) {
			return true, nil
		}

		fmt.Printf("err = %+v\n", err)

		return false, err
	}, "30s").Should(BeTrue())
}

func createHierarchicalNamespace(parentName, cfName, labelKey string) hierarchicalNamespace {
	ctx := context.Background()

	anchor := &hnsv1alpha2.SubnamespaceAnchor{}
	anchor.GenerateName = cfName
	anchor.Namespace = parentName
	anchor.Labels = map[string]string{labelKey: cfName}
	err := k8sClient.Create(ctx, anchor)
	g.Expect(err).NotTo(HaveOccurred())

	return hierarchicalNamespace{
		label:         cfName,
		generatedName: anchor.Name,
		uid:           string(anchor.UID),
		createdAt:     anchor.CreationTimestamp.Time.UTC().Format(time.RFC3339),
	}
}

func deleteSubnamespace(parent, name string) {
	ctx := context.Background()
	namesRequirement, err := labels.NewRequirement(repositories.OrgNameLabel, selection.Equals, []string{"our-org"})
	g.Expect(err).NotTo(HaveOccurred())
	err = k8sClient.DeleteAllOf(ctx, &hnsv1alpha2.SubnamespaceAnchor{}, client.InNamespace(rootNamespace), client.MatchingLabelsSelector{
		Selector: labels.NewSelector().Add(*namesRequirement),
	})
	g.Expect(err).NotTo(HaveOccurred())
}
