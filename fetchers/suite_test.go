package fetchers_test

import (
	"os"
	"path/filepath"
	"testing"

	workloadsv1alpha1 "code.cloudfoundry.org/cf-k8s-controllers/api/v1alpha1"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	suite             spec.Suite
	testEnv           *envtest.Environment
	k8sClient         client.Client
	k8sConfig         *rest.Config
	testServerAddress string
)

func Suite() spec.Suite {
	if suite == nil {
		suite = spec.New("API Shim")
	}

	return suite
}

func SuiteDescribe(desc string, f func(t *testing.T, when spec.G, it spec.S)) bool {
	return Suite()(desc, f)
}

func TestSuite(t *testing.T) {
	Suite().Before(func(t *testing.T) {
		g := NewWithT(t)
		//logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
		logf.SetLogger(zap.New(zap.WriteTo(os.Stderr), zap.UseDevMode(true)))

		testEnv = &envtest.Environment{
			CRDDirectoryPaths:     []string{filepath.Join("fixtures", "vendor", "cf-k8s-controllers", "config", "crd", "bases")},
			ErrorIfCRDPathMissing: true,
		}

		var err error
		k8sConfig, err = testEnv.Start()
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(k8sConfig).NotTo(BeNil())

		err = workloadsv1alpha1.AddToScheme(scheme.Scheme)
		g.Expect(err).NotTo(HaveOccurred())

		k8sClient, err = client.New(k8sConfig, client.Options{Scheme: scheme.Scheme})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(k8sClient).NotTo(BeNil())
	})

	Suite().After(func(t *testing.T) {
		g := NewWithT(t)
		err := testEnv.Stop()
		g.Expect(err).NotTo(HaveOccurred())
	})

	suite.Run(t)
}
