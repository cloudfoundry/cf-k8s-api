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
var _ = SuiteDescribe("API Shim App Secret Create/Update", testEnvSecretCreate)

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
		var (
			testAppGUID    string
			emptyAppRecord = repositories.AppRecord{}
		)
		it.Before(func() {
			testAppGUID = generateAppGUID()
		})

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
				appRepo   repositories.AppRepo
				client    client.Client
				appRecord repositories.AppRecord
			)

			it.Before(func() {
				appRepo = repositories.AppRepo{}
				client, _ = appRepo.ConfigureClient(k8sConfig)
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
					beforeCreationTime time.Time
					createdAppRecord   repositories.AppRecord
				)
				it.Before(func() {
					beforeCreationTime = time.Now().UTC().AddDate(0, 0, -1)

					var err error
					createdAppRecord, err = appRepo.CreateApp(client, appRecord)
					g.Expect(err).To(BeNil())
				})
				it.After(func() {
					g.Expect(cleanupApp(k8sClient, testAppGUID, defaultNamespace)).To(Succeed())
				})

				it("should return a non-empty AppRecord", func() {
					g.Expect(createdAppRecord).NotTo(Equal(emptyAppRecord))
				})

				it("should return an AppRecord with matching GUID, spaceGUID, and name", func() {
					g.Expect(createdAppRecord.GUID).To(Equal(testAppGUID), "App GUID in record did not match input")
					g.Expect(createdAppRecord.SpaceGUID).To(Equal(defaultNamespace), "App SpaceGUID in record did not match input")
					g.Expect(createdAppRecord.Name).To(Equal(testAppName), "App Name in record did not match input")
				})

				it("should return an AppRecord with CreatedAt and UpdatedAt fields that make sense", func() {
					afterTestTime := time.Now().UTC().AddDate(0, 0, 1)
					recordCreatedTime, err := time.Parse(repositories.TimestampFormat, createdAppRecord.CreatedAt)
					g.Expect(err).To(BeNil(), "There was an error converting the createAppRecord CreatedTime to string")
					recordUpdatedTime, err := time.Parse(repositories.TimestampFormat, createdAppRecord.UpdatedAt)
					g.Expect(err).To(BeNil(), "There was an error converting the createAppRecord UpdatedTime to string")

					g.Expect(recordCreatedTime.After(beforeCreationTime)).To(BeTrue(), "app record creation time was not after the expected creation time")
					g.Expect(recordCreatedTime.Before(afterTestTime)).To(BeTrue(), "app record creation time was not before the expected testing time")

					g.Expect(recordUpdatedTime.After(beforeCreationTime)).To(BeTrue(), "app record updated time was not after the expected creation time")
					g.Expect(recordUpdatedTime.Before(afterTestTime)).To(BeTrue(), "app record updated time was not before the expected testing time")
				})

			})
		})

		when("the app already exists", func() {
			var (
				appCR   workloadsv1alpha1.CFApp
				appRepo repositories.AppRepo
				client  client.Client
				ctx     context.Context
			)

			it.Before(func() {
				ctx = context.Background()
				appCR = intializeAppCR(testAppName, testAppGUID, defaultNamespace)

				g.Expect(k8sClient.Create(ctx, &appCR)).To(Succeed())

				appRepo = repositories.AppRepo{}
				client, _ = appRepo.ConfigureClient(k8sConfig)
			})

			it.After(func() {
				g.Expect(k8sClient.Delete(ctx, &appCR)).To(Succeed())
			})

			it("should eventually return true when AppExists is called", func() {
				g.Eventually(func() bool {
					exists, _ := appRepo.AppExists(client, testAppGUID, defaultNamespace)
					return exists
				}, 10*time.Second, 250*time.Millisecond).Should(BeTrue())
				exists, err := appRepo.AppExists(client, testAppGUID, defaultNamespace)
				g.Expect(exists).To(BeTrue())
				g.Expect(err).NotTo(HaveOccurred())
			})

			it("should error when trying to create the same app again", func() {
				appRecord := intializeAppRecord(testAppName, testAppGUID, defaultNamespace)
				createdAppRecord, err := appRepo.CreateApp(client, appRecord)
				g.Expect(err).NotTo(BeNil())
				g.Expect(createdAppRecord).To(Equal(emptyAppRecord))
			})
		})
	})
}

func generateAppEnvSecretName(appGUID string) string {
	return appGUID + "-env"
}

func testEnvSecretCreate(t *testing.T, when spec.G, it spec.S) {
	g := NewWithT(t)

	const (
		defaultNamespace = "default"
	)

	when("an envSecret is created for a CFApp with the Repo", func() {
		const (
			testAppName = "test-app-name"
		)
		var (
			ctx                      context.Context
			appRepo                  repositories.AppRepo
			client                   client.Client
			testAppGUID              string
			testAppEnvSecretName     string
			testAppEnvSecret         repositories.AppEnvVarsRecord
			returnedAppEnvVarsRecord repositories.AppEnvVarsRecord
			returnedErr              error
		)
		it.Before(func() {
			ctx = context.Background()
			appRepo = repositories.AppRepo{}
			client, _ = appRepo.ConfigureClient(k8sConfig)
			testAppGUID = generateAppGUID()
			testAppEnvSecretName = generateAppEnvSecretName(testAppGUID)
			testAppEnvSecret = repositories.AppEnvVarsRecord{
				AppGUID:   testAppGUID,
				SpaceGUID: defaultNamespace,
			}

			returnedAppEnvVarsRecord, returnedErr = appRepo.CreateAppEnvironmentVariables(client, testAppEnvSecret)

		})

		it.After(func() {
			lookupSecretK8sResource := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testAppEnvSecretName,
					Namespace: defaultNamespace,
				},
			}
			g.Expect(client.Delete(ctx, &lookupSecretK8sResource)).To(Succeed(), "Could not clean up the created App Env Secret")
		})

		it("returns a record matching the input and no error", func() {
			g.Expect(returnedAppEnvVarsRecord).To(Equal(testAppEnvSecret))
			g.Expect(returnedErr).To(BeNil())
		})

		when("examining the created Secret in the k8s api", func() {
			var createdCFAppSecret corev1.Secret
			it.Before(func() {
				cfAppSecretLookupKey := types.NamespacedName{Name: testAppEnvSecretName, Namespace: defaultNamespace}
				createdCFAppSecret = corev1.Secret{}
				g.Eventually(func() bool {
					err := client.Get(ctx, cfAppSecretLookupKey, &createdCFAppSecret)
					if err != nil {
						return false
					}
					return true
				}, 10*time.Second, 250*time.Millisecond).Should(BeTrue(), "could not find the secret created by the repo")
			})
			it("is not empty", func() {
				g.Expect(createdCFAppSecret).ToNot(Equal(corev1.Secret{}))
			})
			it("has a Name that is derived from the CFApp", func() {
				g.Expect(createdCFAppSecret.Name).To(Equal(testAppEnvSecretName))
			})
			it("has a label that matches the CFApp GUID", func() {
				labelValue, exists := createdCFAppSecret.Labels[repositories.CFAppGUIDLabel]
				g.Expect(exists).To(BeTrue(), "label for envSecret AppGUID not found")
				g.Expect(labelValue).To(Equal(testAppGUID))
			})
		})

		it("returns an error if the secret already exists", func() {
			testAppEnvSecret.EnvironmentVariables = map[string]string{"foo": "foo", "bar": "bar"}

			returnedUpdatedAppEnvVarsRecord, returnedUpdatedErr := appRepo.CreateAppEnvironmentVariables(client, testAppEnvSecret)
			fmt.Printf("%+v\n%v\n", returnedUpdatedAppEnvVarsRecord, returnedUpdatedErr)
			g.Expect(returnedUpdatedErr).ToNot(BeNil())
			g.Expect(returnedUpdatedAppEnvVarsRecord).To(Equal(repositories.AppEnvVarsRecord{}))
		})

	})
}
