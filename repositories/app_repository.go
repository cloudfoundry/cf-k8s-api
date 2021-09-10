package repositories

import (
	workloadsv1alpha1 "code.cloudfoundry.org/cf-k8s-controllers/apis/workloads/v1alpha1"
	"context"
	"errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"time"

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
)

const (
	Kind       string = "CFApp"
	APIVersion string = "workloads.cloudfoundry.org/v1alpha1"
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

type AppEnvironmentVariablesRecord struct {
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
	cfApp := f.AppRecordToCFApp(appRecord)
	err := client.Create(context.Background(), &cfApp)
	if err != nil {
		return AppRecord{}, err
	}
	return f.CFAppToResponseApp(cfApp), err
}

func (f *AppRepo) AppRecordToCFApp(appRecord AppRecord) workloadsv1alpha1.CFApp {
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

func (f *AppRepo) CFAppToResponseApp(cfApp workloadsv1alpha1.CFApp) AppRecord {
	updatedAtTime := "2019-10-12T07:20:50.52Z"
	//updatedAtTime, _ := getTimeLastUpdatedTimestamp(&cfApp.ObjectMeta)

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
		CreatedAt: cfApp.CreationTimestamp.UTC().Format(time.RFC3339),
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

	return f.CFAppToResponseApp(apps[0]), nil
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

/*
func (f *AppRepo) CreateOrUpdateAppEnvironmentVariables(client client.Client, envVariables AppEnvironmentVariablesRecord) (AppEnvironmentVariablesRecord, error) {
	secretObj := f.appEnvVarsRecordToSecret(envVariables)
	err := client.Create(context.Background(), &cfApp)
	if err != nil {
		return AppRecord{}, err
	}
	return f.CFAppToResponseApp(cfApp), err
}

func (f *AppRepo) appEnvVarsRecordToSecret(envVariables AppEnvironmentVariablesRecord) corev1.Secret {
	labels := make(map[string]string, 1)
	appRequest.Metadata.Labels["apps.cloudfoundry.org/appGuid"] = envVariables.AppGUID
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        envVariables.AppGUID + "-env",
			Namespace:   envVariables.SpaceGUID,
			Labels:      appRequest.Metadata.Labels,
			Annotations: appRequest.Metadata.Annotations,
		},
		StringData: appRequest.EnvironmentVariables,
	}
}

*/
