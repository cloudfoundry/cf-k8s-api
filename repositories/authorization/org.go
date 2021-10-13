package authorization

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//counterfeiter:generate -o fake -fake-name IdentityInspector . IdentityInspector
type IdentityInspector interface {
	WhoAmI(token string) (string, error)
}

type Org struct {
	identityInspector IdentityInspector
	k8sClient         client.Client
}

func NewOrg(k8sClient client.Client, identityInspector IdentityInspector) *Org {
	return &Org{
		k8sClient:         k8sClient,
		identityInspector: identityInspector,
	}
}

func (o *Org) GetAuthorizedOrgs(token string) ([]string, error) {
	return []string{}, nil
}
