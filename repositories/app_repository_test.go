package repositories_test

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/types"
	"testing"
	"time"

	workloadsv1alpha1 "code.cloudfoundry.org/cf-k8s-controllers/apis/workloads/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"code.cloudfoundry.org/cf-k8s-api/repositories"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

var _ = SuiteDescribe("API Shim App Get", testAppGet)
var _ = SuiteDescribe("API Shim App Create", testAppCreate)

func testAppGet(t *testing.T, when spec.G, it spec.S) {
	g := NewWithT(t)

	const (
		cfAppGUID = "test-app-guid"
		namespace = "default"
	)

	when("multiple Apps exist", func() {
		var (
			cfApp1 *workloadsv1alpha1.CFApp
			cfApp2 *workloadsv1alpha1.CFApp
		)
		it.Before(func() {
			ctx := context.Background()
			cfApp1 = &workloadsv1alpha1.CFApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some-other-app",
					Namespace: namespace,
				},
				Spec: workloadsv1alpha1.CFAppSpec{
					Name:         "test-app1",
					DesiredState: "STOPPED",
					Lifecycle: workloadsv1alpha1.Lifecycle{
						Type: "buildpack",
						Data: workloadsv1alpha1.LifecycleData{
							Buildpacks: []string{},
							Stack:      "",
						},
					},
				},
			}
			g.Expect(k8sClient.Create(ctx, cfApp1)).To(Succeed())

			cfApp2 = &workloadsv1alpha1.CFApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfAppGUID,
					Namespace: namespace,
				},
				Spec: workloadsv1alpha1.CFAppSpec{
					Name:         "test-app2",
					DesiredState: "STOPPED",
					Lifecycle: workloadsv1alpha1.Lifecycle{
						Type: "buildpack",
						Data: workloadsv1alpha1.LifecycleData{
							Buildpacks: []string{"java"},
							Stack:      "",
						},
					},
				},
			}
			g.Expect(k8sClient.Create(ctx, cfApp2)).To(Succeed())
		})

		it.After(func() {
			ctx := context.Background()
			g.Expect(k8sClient.Delete(ctx, cfApp1)).To(Succeed())
			g.Expect(k8sClient.Delete(ctx, cfApp2)).To(Succeed())
		})

		it("can fetch the AppRecord CR we're looking for", func() {
			appRepo := repositories.AppRepo{}
			client, err := appRepo.ConfigureClient(k8sConfig)
			g.Expect(err).ToNot(HaveOccurred())

			app, err := appRepo.FetchApp(client, cfAppGUID)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(app.GUID).To(Equal(cfAppGUID))
			g.Expect(app.Name).To(Equal("test-app2"))
			g.Expect(app.SpaceGUID).To(Equal(namespace))
			g.Expect(app.State).To(Equal(repositories.DesiredState("STOPPED")))

			expectedLifecycle := repositories.Lifecycle{
				Data: repositories.LifecycleData{
					Buildpacks: []string{"java"},
					Stack:      "",
				},
			}
			g.Expect(app.Lifecycle).To(Equal(expectedLifecycle))
		})
	})

	when("duplicate Apps exist across namespaces with the same name", func() {
		var (
			cfApp1 *workloadsv1alpha1.CFApp
			cfApp2 *workloadsv1alpha1.CFApp
		)

		it.Before(func() {
			ctx := context.Background()
			g.Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "other-namespace"}})).To(Succeed())

			cfApp1 = &workloadsv1alpha1.CFApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfAppGUID,
					Namespace: namespace,
				},
				Spec: workloadsv1alpha1.CFAppSpec{
					Name:         "test-app1",
					DesiredState: "STOPPED",
					Lifecycle: workloadsv1alpha1.Lifecycle{
						Type: "buildpack",
						Data: workloadsv1alpha1.LifecycleData{
							Buildpacks: []string{},
							Stack:      "",
						},
					},
				},
			}
			g.Expect(k8sClient.Create(ctx, cfApp1)).To(Succeed())

			cfApp2 = &workloadsv1alpha1.CFApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfAppGUID,
					Namespace: "other-namespace",
				},
				Spec: workloadsv1alpha1.CFAppSpec{
					Name:         "test-app2",
					DesiredState: "STOPPED",
					Lifecycle: workloadsv1alpha1.Lifecycle{
						Type: "buildpack",
						Data: workloadsv1alpha1.LifecycleData{
							Buildpacks: []string{},
							Stack:      "",
						},
					},
				},
			}
			g.Expect(k8sClient.Create(ctx, cfApp2)).To(Succeed())
		})

		it.After(func() {
			ctx := context.Background()
			g.Expect(k8sClient.Delete(ctx, cfApp1)).To(Succeed())
			g.Expect(k8sClient.Delete(ctx, cfApp2)).To(Succeed())
		})

		it("returns an error", func() {
			appRepo := repositories.AppRepo{}
			client, err := appRepo.ConfigureClient(k8sConfig)
			g.Expect(err).ToNot(HaveOccurred())

			_, err = appRepo.FetchApp(client, cfAppGUID)
			g.Expect(err).To(HaveOccurred())
			g.Expect(err).To(MatchError("duplicate apps exist"))
		})
	})

	when("no Apps exist", func() {
		it("returns an error", func() {
			appRepo := repositories.AppRepo{}
			client, err := appRepo.ConfigureClient(k8sConfig)
			g.Expect(err).ToNot(HaveOccurred())

			_, err = appRepo.FetchApp(client, "i don't exist")
			g.Expect(err).To(HaveOccurred())
			g.Expect(err).To(MatchError("not found"))
		})
	})
}

func intializeAppCR(appName string, appGUID string, spaceGUID string) workloadsv1alpha1.CFApp {
	return workloadsv1alpha1.CFApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appGUID,
			Namespace: spaceGUID,
		},
		Spec: workloadsv1alpha1.CFAppSpec{
			Name:         appName,
			DesiredState: "STOPPED",
			Lifecycle: workloadsv1alpha1.Lifecycle{
				Type: "buildpack",
				Data: workloadsv1alpha1.LifecycleData{
					Buildpacks: []string{},
					Stack:      "",
				},
			},
		},
	}
}

func intializeAppRecord(appName string, appGUID string, spaceGUID string) repositories.AppRecord {
	return repositories.AppRecord{
		Name:      appName,
		GUID:      appGUID,
		SpaceGUID: spaceGUID,
		State:     "STOPPED",
		Lifecycle: repositories.Lifecycle{
			Type: "buildpack",
			Data: repositories.LifecycleData{
				Buildpacks: []string{},
				Stack:      "cflinuxfs3",
			},
		},
	}
}

func generateAppGUID() string {
	newUUID, err := uuid.NewUUID()
	if err != nil {
		errorMessage := fmt.Sprintf("could not generate a UUID %v", err)
		panic(errorMessage)
	}
	return newUUID.String()
}

func cleanupApp(k8sClient client.Client, appGUID, appNamespace string) error {
	app := workloadsv1alpha1.CFApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appGUID,
			Namespace: appNamespace,
		},
	}
	return k8sClient.Delete(context.Background(), &app)
}

func testAppCreate(t *testing.T, when spec.G, it spec.S) {
	g := NewWithT(t)

	const (
		defaultNamespace = "default"
	)

	when("creating an App record", func() {
		const (
			testAppName = "test-app-name"
		)

		when("space does not exist", func() {

			it("returns an unauthorized or not found err", func() {
				appRepo := repositories.AppRepo{}
				client, err := appRepo.ConfigureClient(k8sConfig)

				_, err = appRepo.FetchNamespace(client, "some-guid")
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError("Invalid space. Ensure that the space exists and you have access to it."))
			})

		})

		when("app does not already exist", func() {
			var (
				appRepo     repositories.AppRepo
				client      client.Client
				testAppGUID string
				appRecord   repositories.AppRecord
			)

			it.Before(func() {
				appRepo = repositories.AppRepo{}
				client, _ = appRepo.ConfigureClient(k8sConfig)
				testAppGUID = generateAppGUID()
				appRecord = intializeAppRecord(testAppName, testAppGUID, defaultNamespace)
			})

			it("returns false when checking if the App Exists", func() {
				exists, err := appRepo.AppExists(client, testAppGUID, defaultNamespace)
				g.Expect(exists).To(BeFalse())
				g.Expect(err).NotTo(HaveOccurred())
			})

			it("should create a new app successfully", func() {
				createdAppRecord, err := appRepo.CreateApp(client, appRecord)
				g.Expect(err).To(BeNil())
				g.Expect(createdAppRecord).NotTo(BeNil())

				cfAppLookupKey := types.NamespacedName{Name: testAppGUID, Namespace: defaultNamespace}
				createdCFApp := new(workloadsv1alpha1.CFApp)
				g.Eventually(func() string {
					err := k8sClient.Get(context.Background(), cfAppLookupKey, createdCFApp)
					if err != nil {
						return ""
					}
					return createdCFApp.Name
				}, 10*time.Second, 250*time.Millisecond).Should(Equal(testAppGUID))
				g.Expect(cleanupApp(k8sClient, testAppGUID, defaultNamespace)).To(Succeed())
			})

			when("an app is created with the repository", func() {
				var (
					creationTime time.Time
					createdAppRecord repositories.AppRecord
				)
				it.Before(func() {
					creationTime = time.Now()

					var err error
					createdAppRecord, err = appRepo.CreateApp(client, appRecord)
					g.Expect(err).To(BeNil())
				})
				it.After(func() {
					g.Expect(cleanupApp(k8sClient, testAppGUID, defaultNamespace)).To(Succeed())
				})

				it("should return a non-empty AppRecord", func() {
					g.Expect(createdAppRecord).NotTo(Equal(repositories.AppRecord{}))
				})

				it("should return an AppRecord with matching GUID, spaceGUID, and name", func() {
					g.Expect(createdAppRecord.GUID).To(Equal(testAppGUID), "App GUID in record did not match input")
					g.Expect(createdAppRecord.SpaceGUID).To(Equal(defaultNamespace), "App SpaceGUID in record did not match input")
					g.Expect(createdAppRecord.Name).To(Equal(testAppName), "App Name in record did not match input")
				})

				it("should return an AppRecord with CreatedAt and UpdatedAt fields that make sense", func() {
					testTime := time.Now()
					recordCreatedTime, err := time.Parse(time.RFC3339, createdAppRecord.CreatedAt)
					g.Expect(err).To(BeNil(), "There was an error converting the createAppRecord CreatedTime to string")
					recordUpdatedTime, err := time.Parse(time.RFC3339, createdAppRecord.UpdatedAt)
					g.Expect(err).To(BeNil(), "There was an error converting the createAppRecord UpdatedTime to string")
					fmt.Sprintf("%v,%v,%v,%v", creationTime, testTime, recordCreatedTime, recordUpdatedTime)


					g.Expect(recordCreatedTime.After(creationTime)).To(BeTrue(), "app record creation time was not after creation time")
					g.Expect(recordCreatedTime.After(creationTime)).To(BeTrue(), "app record creation time was not before test checking time")
				})

			})
		})

		when("the app already exists", func() {
			var (
				cfApp1 *workloadsv1alpha1.CFApp
				// appRecord1 repositories.AppRecord
				appRepo repositories.AppRepo
				client  client.Client
			)

			it.Before(func() {
				ctx := context.Background()
				cfApp1 = &workloadsv1alpha1.CFApp{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "some-other-app",
						Namespace: defaultNamespace,
					},
					Spec: workloadsv1alpha1.CFAppSpec{
						Name:         "test-app1",
						DesiredState: "STOPPED",
						Lifecycle: workloadsv1alpha1.Lifecycle{
							Type: "buildpack",
							Data: workloadsv1alpha1.LifecycleData{
								Buildpacks: []string{},
								Stack:      "",
							},
						},
					},
				}
				g.Expect(k8sClient.Create(ctx, cfApp1)).To(Succeed())

				appRepo = repositories.AppRepo{}
				client, _ = appRepo.ConfigureClient(k8sConfig)
			})

			it.After(func() {
				ctx := context.Background()
				g.Expect(k8sClient.Delete(ctx, cfApp1)).To(Succeed())
			})

			it("should return true when AppExists is called", func() {
				appRepo := repositories.AppRepo{}
				client, _ := appRepo.ConfigureClient(k8sConfig)

				exists, err := appRepo.AppExists(client, "some-other-app", "default")
				g.Expect(exists).To(BeTrue())
				g.Expect(err).NotTo(HaveOccurred())
			})

			it("should error when trying to create the same app again", func() {
				createdAppRecord, err := appRepo.CreateApp(client, appRepo.CFAppToResponseApp(*cfApp1))
				emptyAppRecord := repositories.AppRecord{}
				g.Expect(err).NotTo(BeNil())
				g.Expect(createdAppRecord).To(Equal(emptyAppRecord))
			})
		})
	})
}
