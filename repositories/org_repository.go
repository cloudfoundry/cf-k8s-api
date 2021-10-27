package repositories

import (
	"context"
	"fmt"
	"time"

	"code.cloudfoundry.org/cf-k8s-api/repositories/registry"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/hierarchical-namespaces/api/v1alpha2"
)

//+kubebuilder:rbac:groups=hnc.x-k8s.io,resources=subnamespaceanchors,verbs=list;create

const (
	OrgNameLabel   = "cloudfoundry.org/org-name"
	SpaceNameLabel = "cloudfoundry.org/space-name"
)

type OrgRecord struct {
	Name        string
	GUID        string
	Suspended   bool
	Labels      map[string]string
	Annotations map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SpaceRecord struct {
	Name             string
	GUID             string
	OrganizationGUID string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type OrgRepo struct {
	rootNamespace    string
	privilegedClient client.WithWatch
	timeout          time.Duration
}

func NewOrgRepo(rootNamespace string, privilegedClient client.WithWatch, timeout time.Duration) *OrgRepo {
	return &OrgRepo{
		rootNamespace:    rootNamespace,
		privilegedClient: privilegedClient,
		timeout:          timeout,
	}
}

func (r *OrgRepo) CreateOrg(ctx context.Context, org OrgRecord) (OrgRecord, error) {
	registrar := registry.NewRegistrar(r.privilegedClient, r.rootNamespace)
	registration, err := registrar.TryRegister(ctx, registry.OrgType, r.rootNamespace, org.Name)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return OrgRecord{}, fmt.Errorf("%s record %q already exists in namespace %s: %w", string(registry.OrgType), r.rootNamespace, org.Name, err)
		}

		return OrgRecord{}, fmt.Errorf("failed to register name: %w", err)
	}

	anchor := &v1alpha2.SubnamespaceAnchor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      org.GUID,
			Namespace: r.rootNamespace,
			Labels: map[string]string{
				OrgNameLabel: org.Name,
			},
		},
	}
	err = r.privilegedClient.Create(ctx, anchor)
	if err != nil {
		return OrgRecord{}, fmt.Errorf("failed to create subnamespaceanchor: %w", err)
	}

	err = registrar.SetOwnerRef(ctx, anchor, registration)
	if err != nil {
		// todo
	}

	timeoutCtx, cancelFn := context.WithTimeout(ctx, r.timeout)
	defer cancelFn()

	watch, err := r.privilegedClient.Watch(timeoutCtx, &v1alpha2.SubnamespaceAnchorList{},
		client.InNamespace(r.rootNamespace),
		client.MatchingFields{"metadata.name": org.GUID},
	)
	if err != nil {
		return OrgRecord{}, fmt.Errorf("failed to set up watch on subnamespaceanchors: %w", err)
	}

	stateOK := false
	for res := range watch.ResultChan() {
		obj, ok := res.Object.(*v1alpha2.SubnamespaceAnchor)
		if !ok {
			// should never happen, but avoids panic above
			continue
		}
		if obj.Status.State == v1alpha2.Ok {
			watch.Stop()
			stateOK = true
			break
		}
	}
	if !stateOK {
		return OrgRecord{}, fmt.Errorf("subnamespaceanchor did not get state 'ok' within timeout period %d ms", r.timeout.Milliseconds())
	}

	org.GUID = anchor.Name
	org.CreatedAt = anchor.CreationTimestamp.Time
	org.UpdatedAt = anchor.CreationTimestamp.Time

	return org, nil
}

func (r *OrgRepo) FetchOrgs(ctx context.Context, names []string) ([]OrgRecord, error) {
	subnamespaceAnchorList := &v1alpha2.SubnamespaceAnchorList{}

	options := []client.ListOption{client.InNamespace(r.rootNamespace)}
	if len(names) > 0 {
		namesRequirement, err := labels.NewRequirement(OrgNameLabel, selection.In, names)
		if err != nil {
			return nil, err
		}
		namesSelector := client.MatchingLabelsSelector{
			Selector: labels.NewSelector().Add(*namesRequirement),
		}
		options = append(options, namesSelector)
	}

	err := r.privilegedClient.List(ctx, subnamespaceAnchorList, options...)
	if err != nil {
		return nil, err
	}

	records := []OrgRecord{}
	for _, anchor := range subnamespaceAnchorList.Items {
		if anchor.Status.State != v1alpha2.Ok {
			continue
		}

		records = append(records, OrgRecord{
			Name:      anchor.Labels[OrgNameLabel],
			GUID:      anchor.Name,
			CreatedAt: anchor.CreationTimestamp.Time,
			UpdatedAt: anchor.CreationTimestamp.Time,
		})
	}

	return records, nil
}

func (r *OrgRepo) FetchSpaces(ctx context.Context, organizationGUIDs, names []string) ([]SpaceRecord, error) {
	subnamespaceAnchorList := &v1alpha2.SubnamespaceAnchorList{}

	err := r.privilegedClient.List(ctx, subnamespaceAnchorList)
	if err != nil {
		return nil, err
	}

	orgsFilter := toMap(organizationGUIDs)
	orgUIDs := map[string]struct{}{}
	for _, anchor := range subnamespaceAnchorList.Items {
		if anchor.Namespace != r.rootNamespace {
			continue
		}

		if !matchFilter(orgsFilter, anchor.Name) {
			continue
		}

		orgUIDs[anchor.Name] = struct{}{}
	}

	nameFilter := toMap(names)
	records := []SpaceRecord{}
	for _, anchor := range subnamespaceAnchorList.Items {
		if anchor.Status.State != v1alpha2.Ok {
			continue
		}

		spaceName := anchor.Labels[SpaceNameLabel]
		if !matchFilter(nameFilter, spaceName) {
			continue
		}

		if _, ok := orgUIDs[anchor.Namespace]; !ok {
			continue
		}

		records = append(records, SpaceRecord{
			Name:             spaceName,
			GUID:             anchor.Name,
			OrganizationGUID: anchor.Namespace,
			CreatedAt:        anchor.CreationTimestamp.Time,
			UpdatedAt:        anchor.CreationTimestamp.Time,
		})
	}

	return records, nil
}

func matchFilter(filter map[string]struct{}, value string) bool {
	if len(filter) == 0 {
		return true
	}

	_, ok := filter[value]
	return ok
}

func toMap(elements []string) map[string]struct{} {
	result := map[string]struct{}{}
	for _, element := range elements {
		result[element] = struct{}{}
	}

	return result
}
