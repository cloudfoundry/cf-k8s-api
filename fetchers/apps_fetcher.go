package fetchers

import (
	"context"
	"errors"

	workloadsv1alpha1 "code.cloudfoundry.org/cf-k8s-controllers/api/v1alpha1"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AppsFetcher struct {
	Config *rest.Config
	client client.Client
}

type CFApp struct {
	GUID string
}

func (f *AppsFetcher) ConfigureClient(config *rest.Config) (client.Client, error) {
	client, err := client.New(config, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (f *AppsFetcher) FetchApp(client client.Client, appGUID string) (CFApp, error) {
	// TODO: Could look up namespace from guid => namespace cache to do Get
	appList := &workloadsv1alpha1.CFAppList{}
	err := client.List(context.Background(), appList)
	if err != nil {
		return CFApp{}, err
	}
	allApps := appList.Items
	matches := f.FilterAppsByName(allApps, appGUID)

	return f.ReturnApps(matches)
}

func (f *AppsFetcher) ReturnApps(apps []workloadsv1alpha1.CFApp) (CFApp, error) {
	if len(apps) == 0 {
		return CFApp{}, errors.New("not found")
	}
	if len(apps) > 1 {
		return CFApp{}, errors.New("duplicate apps exist")
	}

	return CFApp{GUID: apps[0].Name}, nil
}

func (f *AppsFetcher) FilterAppsByName(apps []workloadsv1alpha1.CFApp, name string) []workloadsv1alpha1.CFApp {
	filtered := []workloadsv1alpha1.CFApp{}
	for i, app := range apps {
		if app.Name == name {
			filtered = append(filtered, apps[i])
		}
	}
	return filtered
}
