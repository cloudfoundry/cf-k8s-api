package repositories

import (
	workloadsv1alpha1 "code.cloudfoundry.org/cf-k8s-controllers/apis/workloads/v1alpha1"
	"context"
	"errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AppRepo struct{}

const (
	StartedState DesiredState = "STARTED"
	StoppedState DesiredState = "STOPPED"

	Kind            string = "CFApp"
	APIVersion      string = "workloads.cloudfoundry.org/v1alpha1"
	TimestampFormat string = time.RFC3339
	CFAppGUIDLabel  string = "apps.cloudfoundry.org/appGuid"
)

type AppRecord struct {
	Name      string
	GUID      string
	SpaceGUID string
	State     DesiredState
	Lifecycle Lifecycle
	CreatedAt string
	UpdatedAt string
}

type DesiredState string

type Lifecycle struct {
	Type string
	Data LifecycleData
}

type LifecycleData struct {
	Buildpacks []string
	Stack      string
}

type SpaceRecord struct {
	Name             string
	OrganizationGUID string
}

type AppEnvVarsRecord struct {
	AppGUID              string
	SpaceGUID            string
	EnvironmentVariables map[string]string
}

// TODO: Make a general ConfigureClient function / config and client generating package
func (f *AppRepo) ConfigureClient(config *rest.Config) (client.Client, error) {
	client, err := client.New(config, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (f *AppRepo) FetchApp(client client.Client, appGUID string) (AppRecord, error) {
	// TODO: Could look up namespace from guid => namespace cache to do Get
	appList := &workloadsv1alpha1.CFAppList{}
	err := client.List(context.Background(), appList)
	if err != nil {
		return AppRecord{}, err
	}
	allApps := appList.Items
	matches := f.filterAppsByName(allApps, appGUID)

	return f.returnApps(matches)
}

func (f *AppRepo) getAppCR(client client.Client, appGUID string, namespace string) (*workloadsv1alpha1.CFApp, error) {
	app := &workloadsv1alpha1.CFApp{}
	err := client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      appGUID,
	}, app)
	return app, err
}

func (f *AppRepo) AppExists(client client.Client, appGUID string, namespace string) (bool, error) {
	_, err := f.getAppCR(client, appGUID, namespace)
	if err != nil {
		switch errtype := err.(type) {
		case *k8serrors.StatusError:
			reason := errtype.Status().Reason
			if reason == metav1.StatusReasonNotFound {
				return false, nil
			}
		default:
			return true, err
		}
	}
	return true, nil
}

func (f *AppRepo) CreateApp(client client.Client, appRecord AppRecord) (AppRecord, error) {
	cfApp := f.appRecordToCFApp(appRecord)
	err := client.Create(context.Background(), &cfApp)
	if err != nil {
		return AppRecord{}, err
	}
	return f.cfAppToResponseApp(cfApp), err
}

func (f *AppRepo) appRecordToCFApp(appRecord AppRecord) workloadsv1alpha1.CFApp {
	return workloadsv1alpha1.CFApp{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      appRecord.GUID,
			Namespace: appRecord.SpaceGUID,
		},
		Spec: workloadsv1alpha1.CFAppSpec{
			Name:         appRecord.Name,
			DesiredState: workloadsv1alpha1.DesiredState(appRecord.State),
			Lifecycle: workloadsv1alpha1.Lifecycle{
				Type: workloadsv1alpha1.LifecycleType(appRecord.Lifecycle.Type),
				Data: workloadsv1alpha1.LifecycleData{
					Buildpacks: appRecord.Lifecycle.Data.Buildpacks,
					Stack:      appRecord.Lifecycle.Data.Stack,
				},
			},
		},
	}
}

func (f *AppRepo) cfAppToResponseApp(cfApp workloadsv1alpha1.CFApp) AppRecord {
	updatedAtTime, _ := getTimeLastUpdatedTimestamp(&cfApp.ObjectMeta)

	return AppRecord{
		GUID:      cfApp.Name,
		Name:      cfApp.Spec.Name,
		SpaceGUID: cfApp.Namespace,
		State:     DesiredState(cfApp.Spec.DesiredState),
		Lifecycle: Lifecycle{
			Data: LifecycleData{
				Buildpacks: cfApp.Spec.Lifecycle.Data.Buildpacks,
				Stack:      cfApp.Spec.Lifecycle.Data.Stack,
			},
		},
		CreatedAt: cfApp.CreationTimestamp.UTC().Format(TimestampFormat),
		UpdatedAt: updatedAtTime,
	}
}

func (f *AppRepo) returnApps(apps []workloadsv1alpha1.CFApp) (AppRecord, error) {
	if len(apps) == 0 {
		return AppRecord{}, NotFoundError{Err: errors.New("not found")}
	}
	if len(apps) > 1 {
		return AppRecord{}, errors.New("duplicate apps exist")
	}

	return f.cfAppToResponseApp(apps[0]), nil
}

func (f *AppRepo) filterAppsByName(apps []workloadsv1alpha1.CFApp, name string) []workloadsv1alpha1.CFApp {
	filtered := []workloadsv1alpha1.CFApp{}
	for i, app := range apps {
		if app.Name == name {
			filtered = append(filtered, apps[i])
		}
	}
	return filtered
}

func (f *AppRepo) FetchNamespace(client client.Client, nsGUID string) (SpaceRecord, error) {
	namespace := &v1.Namespace{}
	err := client.Get(context.Background(), types.NamespacedName{Name: nsGUID}, namespace)
	if err != nil {
		switch errtype := err.(type) {
		case *k8serrors.StatusError:
			reason := errtype.Status().Reason
			if reason == metav1.StatusReasonNotFound || reason == metav1.StatusReasonUnauthorized {
				return SpaceRecord{}, PermissionDeniedOrNotFoundError{Err: err}
			}
		}
		return SpaceRecord{}, err
	}
	return f.v1NamespaceToSpaceRecord(namespace), nil
}

func (f *AppRepo) v1NamespaceToSpaceRecord(namespace *v1.Namespace) SpaceRecord {
	//TODO How do we derive Organization GUID here?
	return SpaceRecord{
		Name:             namespace.Name,
		OrganizationGUID: "",
	}
}

func (f *AppRepo) CreateAppEnvironmentVariables(client client.Client, envVariables AppEnvVarsRecord) (AppEnvVarsRecord, error) {
	secretObj := f.appEnvVarsRecordToSecret(envVariables)
	err := client.Create(context.Background(), &secretObj)
	if err != nil {
		return AppEnvVarsRecord{}, err
	}
	return f.appEnvVarsSecretToRecord(secretObj), nil
}

var staticCFApp workloadsv1alpha1.CFApp

func (f *AppRepo) GenerateEnvSecretName(appGUID string) string {
	return appGUID + "-env"
}
func (f *AppRepo) extractAppGUIDFromEnvSecretName(envSecretName string) string {
	return strings.Trim(envSecretName, "-env")
}

func (f *AppRepo) appEnvVarsRecordToSecret(envVars AppEnvVarsRecord) corev1.Secret {
	labels := make(map[string]string, 1)
	labels[CFAppGUIDLabel] = envVars.AppGUID
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.GenerateEnvSecretName(envVars.AppGUID),
			Namespace: envVars.SpaceGUID,
			Labels:    labels,
		},
		StringData: envVars.EnvironmentVariables,
	}
}

func (f *AppRepo) appEnvVarsSecretToRecord(envVars corev1.Secret) AppEnvVarsRecord {
	return AppEnvVarsRecord{
		AppGUID:              f.extractAppGUIDFromEnvSecretName(envVars.Name),
		SpaceGUID:            envVars.Namespace,
		EnvironmentVariables: envVars.StringData,
	}
}
