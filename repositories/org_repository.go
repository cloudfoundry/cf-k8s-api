package repositories

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/hierarchical-namespaces/api/v1alpha2"
)

const (
	OrgNameLabel   = "cloudfoundry.org/org-name"
	SpaceNameLabel = "cloudfoundry.org/space-name"
)

type OrgRecord struct {
	Name      string
	GUID      string
	Suspended bool
	CreatedAt time.Time
	UpdatedAt time.Time
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
	privilegedClient client.Client
}

func NewOrgRepo(rootNamespace string, privilegedClient client.Client) *OrgRepo {
	return &OrgRepo{
		rootNamespace:    rootNamespace,
		privilegedClient: privilegedClient,
	}
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
		records = append(records, OrgRecord{
			Name:      anchor.Labels[OrgNameLabel],
			GUID:      string(anchor.UID),
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
	orgUIDs := map[string]string{}
	for _, anchor := range subnamespaceAnchorList.Items {
		if anchor.Namespace != r.rootNamespace {
			continue
		}

		anchorUID := string(anchor.UID)
		if !matchFilter(orgsFilter, anchorUID) {
			continue
		}

		orgUIDs[anchor.Name] = anchorUID
	}

	nameFilter := toMap(names)
	records := []SpaceRecord{}
	for _, anchor := range subnamespaceAnchorList.Items {
		spaceName := anchor.Labels[SpaceNameLabel]
		if !matchFilter(nameFilter, spaceName) {
			continue
		}

		if _, ok := orgUIDs[anchor.Namespace]; !ok {
			continue
		}

		records = append(records, SpaceRecord{
			Name:             spaceName,
			GUID:             string(anchor.UID),
			OrganizationGUID: orgUIDs[anchor.Namespace],
			CreatedAt:        anchor.CreationTimestamp.Time,
			UpdatedAt:        anchor.CreationTimestamp.Time,
		})
	}

	return records, nil
}

func matchFilter(filter map[string]bool, value string) bool {
	if len(filter) == 0 {
		return true
	}

	_, ok := filter[value]
	return ok
}

func toMap(elements []string) map[string]bool {
	result := map[string]bool{}
	for _, element := range elements {
		result[element] = true
	}

	return result
}
