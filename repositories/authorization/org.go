package authorization

import (
	"context"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const subjectsNameIndex = ".index.subjects.name"

//counterfeiter:generate -o fake -fake-name IdentityInspector . IdentityInspector
type IdentityInspector interface {
	WhoAmI(token string) (string, error)
}

type Org struct {
	identityInspector IdentityInspector
	k8sClient         client.Client
}

func addIndexes(mgr manager.Manager) error {
	return nil
}

func populateBindingIndex(obj client.Object) []string {
	binding, ok := obj.(*rbacv1.RoleBinding)
	if !ok {
		return nil
	}

	names := []string{}
	for _, subject := range binding.Subjects {
		names = append(names, subject.Name)
	}

	return names
}

func NewOrg(client client.Client, identityInspector IdentityInspector) *Org {
	return &Org{
		k8sClient:         client,
		identityInspector: identityInspector,
	}
}

func (o *Org) SetupWithManager(mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&rbacv1.RoleBinding{},
		subjectsNameIndex,
		populateBindingIndex,
	); err != nil {
		return errors.Wrapf(err, "Failed to create index %q", subjectsNameIndex)
	}

	return nil
}

func (o *Org) GetAuthorizedOrgs(ctx context.Context, token string) ([]string, error) {
	// TODO: handle error
	userInfo, _ := o.identityInspector.WhoAmI(token)

	bindingList := rbacv1.RoleBindingList{}

	// TODO: test error
	err := o.k8sClient.List(ctx, &bindingList, client.MatchingFields{subjectsNameIndex: userInfo.Username})
	if err != nil {
		return nil, err
	}

	nsMap := map[string]bool{}

	for _, binding := range bindingList.Items {
		nsMap[binding.Namespace] = true
	}

	namespaces := []string{}
	for ns := range nsMap {
		namespaces = append(namespaces, ns)
	}

	return namespaces, nil
}
