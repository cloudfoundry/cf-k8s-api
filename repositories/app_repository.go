package repositories

import (
	"context"
	"errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	workloadsv1alpha1 "code.cloudfoundry.org/cf-k8s-controllers/api/v1alpha1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AppRepo struct{}

const (
	StartedState DesiredState = "STARTED"
	StoppedState DesiredState = "STOPPED"
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

func cfAppToResponseApp(cfApp workloadsv1alpha1.CFApp) AppRecord {
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
	}
}

func (f *AppRepo) returnApps(apps []workloadsv1alpha1.CFApp) (AppRecord, error) {
	if len(apps) == 0 {
		return AppRecord{}, NotFoundError{Err: errors.New("not found")}
	}
	if len(apps) > 1 {
		return AppRecord{}, errors.New("duplicate apps exist")
	}

	return cfAppToResponseApp(apps[0]), nil
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
