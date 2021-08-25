package fetchers

import (
	"context"
	"errors"

	workloadsv1alpha1 "code.cloudfoundry.org/cf-k8s-controllers/api/v1alpha1"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AppsFetcher struct{}

type CFApp struct {
	GUID string
}

func (f *AppsFetcher) FetchApp(appGUID string, conf *rest.Config) (CFApp, error) {
	k8sClient, err := client.New(conf, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return CFApp{}, err
	}

	// TODO: Could look up namespace from guid => namespace cache to do Get
	appList := &workloadsv1alpha1.CFAppList{}
	err = k8sClient.List(context.Background(), appList)
	if err != nil {
		return CFApp{}, err
	}
	allApps := appList.Items

	matches := []workloadsv1alpha1.CFApp{}
	for i, app := range allApps {
		if app.Name == appGUID {
			matches = append(matches, allApps[i])
		}
	}

	if len(matches) == 0 {
		return CFApp{}, errors.New("not found")
	}
	if len(matches) > 1 {
		return CFApp{}, errors.New("duplicate apps exist")
	}

	return CFApp{GUID: matches[0].Name}, nil
}
