package repositories_test

import (
	"context"
	"testing"
	"time"

	workloadsv1alpha1 "code.cloudfoundry.org/cf-k8s-controllers/apis/workloads/v1alpha1"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "code.cloudfoundry.org/cf-k8s-api/repositories"

	"github.com/sclevine/spec"
)

var _ = SuiteDescribe("Package Repository CreatePackage", testCreatePackage)
var _ = SuiteDescribe("Package Repository FetchPackage", testFetchPackage)
var _ = SuiteDescribe("Package Repository UpdatePackageSource", testUpdatePackageSourceRegistry)

func testCreatePackage(t *testing.T, when spec.G, it spec.S) {
	g := NewWithT(t)

	var (
		packageRepo   *PackageRepo
		client        client.Client
		packageCreate PackageCreateMessage
		ctx           context.Context
	)

	const (
		appGUID   = "the-app-guid"
		spaceGUID = "the-space-guid"
	)

	it.Before(func() {
		packageRepo = new(PackageRepo)

		var err error
		client, err = BuildCRClient(k8sConfig)
		g.Expect(err).NotTo(HaveOccurred())

		packageCreate = PackageCreateMessage{
			Type:      "bits",
			AppGUID:   appGUID,
			SpaceGUID: spaceGUID,
		}

		ctx = context.Background()
		g.Expect(
			k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: spaceGUID}}),
		).To(Succeed())

	})

	it.After(func() {
		g.Expect(
			k8sClient.Delete(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: spaceGUID}}),
		).To(Succeed())
	})

	it("creates a Package record", func() {
		returnedPackageRecord, err := packageRepo.CreatePackage(ctx, client, packageCreate)
		g.Expect(err).NotTo(HaveOccurred())

		packageGUID := returnedPackageRecord.GUID
		g.Expect(packageGUID).NotTo(BeEmpty())
		g.Expect(returnedPackageRecord.Type).To(Equal("bits"))
		g.Expect(returnedPackageRecord.AppGUID).To(Equal(appGUID))
		g.Expect(returnedPackageRecord.State).To(Equal("AWAITING_UPLOAD"))

		createdAt, err := time.Parse(time.RFC3339, returnedPackageRecord.CreatedAt)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(createdAt).To(BeTemporally("~", time.Now(), timeCheckThreshold*time.Second))

		updatedAt, err := time.Parse(time.RFC3339, returnedPackageRecord.CreatedAt)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(updatedAt).To(BeTemporally("~", time.Now(), timeCheckThreshold*time.Second))

		packageNSName := types.NamespacedName{Name: packageGUID, Namespace: spaceGUID}
		createdCFPackage := new(workloadsv1alpha1.CFPackage)
		g.Eventually(func() bool {
			err := k8sClient.Get(context.Background(), packageNSName, createdCFPackage)
			return err == nil
		}, 10*time.Second, 250*time.Millisecond).Should(BeTrue())

		g.Expect(createdCFPackage.Name).To(Equal(packageGUID))
		g.Expect(createdCFPackage.Namespace).To(Equal(spaceGUID))
		g.Expect(createdCFPackage.Spec.Type).To(Equal(workloadsv1alpha1.PackageType("bits")))
		g.Expect(createdCFPackage.Spec.AppRef.Name).To(Equal(appGUID))

		g.Expect(cleanupPackage(ctx, k8sClient, packageGUID, spaceGUID)).To(Succeed())
	})
}

func testFetchPackage(t *testing.T, when spec.G, it spec.S) {
	g := NewWithT(t)

	const (
		appGUID = "the-app-guid"
	)

	var (
		packageRepo *PackageRepo
		testClient  client.Client

		namespace1 *corev1.Namespace
		namespace2 *corev1.Namespace
	)

	it.Before(func() {
		namespace1Name := generateGUID()
		namespace1 = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace1Name}}
		g.Expect(k8sClient.Create(context.Background(), namespace1)).To(Succeed())

		namespace2Name := generateGUID()
		namespace2 = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace2Name}}
		g.Expect(k8sClient.Create(context.Background(), namespace2)).To(Succeed())

		packageRepo = new(PackageRepo)
		var err error
		testClient, err = BuildCRClient(k8sConfig)
		g.Expect(err).ToNot(HaveOccurred())
	})

	it.After(func() {
		g.Expect(k8sClient.Delete(context.Background(), namespace1)).To(Succeed())
		g.Expect(k8sClient.Delete(context.Background(), namespace2)).To(Succeed())
	})

	when("on the happy path", func() {
		var (
			package1GUID string
			package2GUID string
			package1     *workloadsv1alpha1.CFPackage
			package2     *workloadsv1alpha1.CFPackage
		)

		it.Before(func() {
			package1GUID = generateGUID()
			package2GUID = generateGUID()
			package1 = &workloadsv1alpha1.CFPackage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      package1GUID,
					Namespace: namespace1.Name,
				},
				Spec: workloadsv1alpha1.CFPackageSpec{
					Type: "bits",
					AppRef: corev1.LocalObjectReference{
						Name: appGUID,
					},
				},
			}
			g.Expect(k8sClient.Create(context.Background(), package1)).To(Succeed())

			package2 = &workloadsv1alpha1.CFPackage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      package2GUID,
					Namespace: namespace2.Name,
				},
				Spec: workloadsv1alpha1.CFPackageSpec{
					Type: "bits",
					AppRef: corev1.LocalObjectReference{
						Name: appGUID,
					},
				},
			}
			g.Expect(k8sClient.Create(context.Background(), package2)).To(Succeed())
		})

		it.After(func() {
			g.Expect(k8sClient.Delete(context.Background(), package1)).To(Succeed())
			g.Expect(k8sClient.Delete(context.Background(), package2)).To(Succeed())
		})

		it("can fetch the PackageRecord we're looking for", func() {
			record, err := packageRepo.FetchPackage(context.Background(), testClient, package2GUID)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(record.GUID).To(Equal(package2GUID))
			g.Expect(record.Type).To(Equal("bits"))
			g.Expect(record.AppGUID).To(Equal(appGUID))
			g.Expect(record.State).To(Equal("AWAITING_UPLOAD"))

			createdAt, err := time.Parse(time.RFC3339, record.CreatedAt)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(createdAt).To(BeTemporally("~", time.Now(), timeCheckThreshold*time.Second))

			updatedAt, err := time.Parse(time.RFC3339, record.CreatedAt)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(updatedAt).To(BeTemporally("~", time.Now(), timeCheckThreshold*time.Second))
		})
	})

	when("table-testing the State field", func() {
		var (
			cfPackage *workloadsv1alpha1.CFPackage
		)

		it.Before(func() {
			packageGUID := generateGUID()
			cfPackage = &workloadsv1alpha1.CFPackage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      packageGUID,
					Namespace: namespace1.Name,
				},
				Spec: workloadsv1alpha1.CFPackageSpec{
					Type: "bits",
					AppRef: corev1.LocalObjectReference{
						Name: appGUID,
					},
				},
			}
		})

		type testCase struct {
			description   string
			expectedState string
			setupFunc     func(cfPackage2 *workloadsv1alpha1.CFPackage)
		}

		cases := []testCase{
			{
				description:   "no source image is set",
				expectedState: "AWAITING_UPLOAD",
				setupFunc:     func(p *workloadsv1alpha1.CFPackage) { p.Spec.Source = workloadsv1alpha1.PackageSource{} },
			},
			{
				description:   "an source image is set",
				expectedState: "PROCESSING_UPLOAD",
				setupFunc:     func(p *workloadsv1alpha1.CFPackage) { p.Spec.Source.Registry.Image = "some-org/some-repo" },
			},
		}

		for _, tc := range cases {
			when(tc.description, func() {
				it("has state "+tc.expectedState, func() {
					tc.setupFunc(cfPackage)
					g.Expect(k8sClient.Create(context.Background(), cfPackage)).To(Succeed())
					defer func() { g.Expect(k8sClient.Delete(context.Background(), cfPackage)).To(Succeed()) }()

					record, err := packageRepo.FetchPackage(context.Background(), testClient, cfPackage.Name)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(record.State).To(Equal(tc.expectedState))
				})
			})
		}
	})

	when("duplicate Packages exist across namespaces with the same GUID", func() {
		var (
			packageGUID string
			cfPackage1  *workloadsv1alpha1.CFPackage
			cfPackage2  *workloadsv1alpha1.CFPackage
		)

		it.Before(func() {
			packageGUID = generateGUID()
			cfPackage1 = &workloadsv1alpha1.CFPackage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      packageGUID,
					Namespace: namespace1.Name,
				},
				Spec: workloadsv1alpha1.CFPackageSpec{
					Type: "bits",
					AppRef: corev1.LocalObjectReference{
						Name: appGUID,
					},
				},
			}
			g.Expect(k8sClient.Create(context.Background(), cfPackage1)).To(Succeed())

			cfPackage2 = &workloadsv1alpha1.CFPackage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      packageGUID,
					Namespace: namespace2.Name,
				},
				Spec: workloadsv1alpha1.CFPackageSpec{
					Type: "bits",
					AppRef: corev1.LocalObjectReference{
						Name: appGUID,
					},
				},
			}
			g.Expect(k8sClient.Create(context.Background(), cfPackage2)).To(Succeed())
		})

		it.After(func() {
			g.Expect(k8sClient.Delete(context.Background(), cfPackage1)).To(Succeed())
			g.Expect(k8sClient.Delete(context.Background(), cfPackage2)).To(Succeed())
		})

		it("returns an error", func() {
			_, err := packageRepo.FetchPackage(context.Background(), testClient, packageGUID)
			g.Expect(err).To(HaveOccurred())
			g.Expect(err).To(MatchError("duplicate packages exist"))
		})
	})

	when("no packages exist", func() {
		it("returns an error", func() {
			_, err := packageRepo.FetchPackage(context.Background(), testClient, "i don't exist")
			g.Expect(err).To(HaveOccurred())
			g.Expect(err).To(MatchError(NotFoundError{}))
		})
	})
}

func testUpdatePackageSourceRegistry(t *testing.T, when spec.G, it spec.S) {
	g := NewWithT(t)

	var (
		packageRepo       *PackageRepo
		client            client.Client
		existingCFPackage workloadsv1alpha1.CFPackage
		spaceGUID         string
		updateMessage     PackageUpdateSourceMessage
	)

	const (
		packageGUID               = "the-package-guid"
		appGUID                   = "the-app-guid"
		packageSourceImageRef     = "my-org/" + packageGUID
		packageRegistrySecretName = "image-pull-secret"
	)

	it.Before(func() {
		spaceGUID = generateGUID()
		packageRepo = new(PackageRepo)
		ctx := context.Background()

		var err error
		client, err = BuildCRClient(k8sConfig)
		g.Expect(err).NotTo(HaveOccurred())

		existingCFPackage = workloadsv1alpha1.CFPackage{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CFPackage",
				APIVersion: workloadsv1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      packageGUID,
				Namespace: spaceGUID,
			},
			Spec: workloadsv1alpha1.CFPackageSpec{
				Type:   "bits",
				AppRef: corev1.LocalObjectReference{Name: appGUID},
			},
		}

		updateMessage = PackageUpdateSourceMessage{
			GUID:               packageGUID,
			SpaceGUID:          spaceGUID,
			ImageRef:           packageSourceImageRef,
			RegistrySecretName: packageRegistrySecretName,
		}

		g.Expect(
			k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: spaceGUID}}),
		).To(Succeed())

		g.Expect(
			k8sClient.Create(ctx, &existingCFPackage),
		).To(Succeed())
	})

	it.After(func() {
		g.Expect(
			k8sClient.Delete(context.Background(), &existingCFPackage),
		).To(Succeed())

		g.Expect(
			k8sClient.Delete(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: spaceGUID}}),
		).To(Succeed())
	})

	it("returns an updated record", func() {
		returnedPackageRecord, err := packageRepo.UpdatePackageSource(context.Background(), client, updateMessage)
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(returnedPackageRecord.GUID).To(Equal(existingCFPackage.ObjectMeta.Name))
		g.Expect(returnedPackageRecord.Type).To(Equal(string(existingCFPackage.Spec.Type)))
		g.Expect(returnedPackageRecord.AppGUID).To(Equal(existingCFPackage.Spec.AppRef.Name))
		g.Expect(returnedPackageRecord.SpaceGUID).To(Equal(existingCFPackage.ObjectMeta.Namespace))
		g.Expect(returnedPackageRecord.State).To(Equal("PROCESSING_UPLOAD"))

		createdAt, err := time.Parse(time.RFC3339, returnedPackageRecord.CreatedAt)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(createdAt).To(BeTemporally("~", time.Now(), time.Second))

		updatedAt, err := time.Parse(time.RFC3339, returnedPackageRecord.CreatedAt)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(updatedAt).To(BeTemporally("~", time.Now(), time.Second))
	})

	it("updates only the Registry field of the existing CFPackage", func() {
		_, err := packageRepo.UpdatePackageSource(context.Background(), client, updateMessage)
		g.Expect(err).NotTo(HaveOccurred())

		packageNSName := types.NamespacedName{Name: packageGUID, Namespace: spaceGUID}
		createdCFPackage := new(workloadsv1alpha1.CFPackage)
		g.Eventually(func() bool {
			err := k8sClient.Get(context.Background(), packageNSName, createdCFPackage)
			return err == nil
		}, 10*time.Second, 250*time.Millisecond).Should(BeTrue())

		g.Expect(createdCFPackage.Name).To(Equal(existingCFPackage.ObjectMeta.Name))
		g.Expect(createdCFPackage.Namespace).To(Equal(existingCFPackage.ObjectMeta.Namespace))
		g.Expect(createdCFPackage.Spec.Type).To(Equal(existingCFPackage.Spec.Type))
		g.Expect(createdCFPackage.Spec.AppRef).To(Equal(existingCFPackage.Spec.AppRef))
		g.Expect(createdCFPackage.Spec.Source.Registry).To(Equal(workloadsv1alpha1.Registry{
			Image:            packageSourceImageRef,
			ImagePullSecrets: []corev1.LocalObjectReference{{Name: packageRegistrySecretName}},
		}))
	})
}

func cleanupPackage(ctx context.Context, k8sClient client.Client, packageGUID, namespace string) error {
	cfPackage := workloadsv1alpha1.CFPackage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      packageGUID,
			Namespace: namespace,
		},
	}
	return k8sClient.Delete(ctx, &cfPackage)
}
