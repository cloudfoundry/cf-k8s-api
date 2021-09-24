package presenter

import (
	neturl "net/url"
	"path"
	"time"

	"code.cloudfoundry.org/cf-k8s-api/repositories"
)

const (
	// TODO: repetition with handler endpoint?
	orgsBase   = "/v3/organizations"
	spacesBase = "/v3/spaces"
)

type OrgListResponse struct {
	Pagination PaginationData `json:"pagination"`
	Resources  []OrgResponse  `json:"resources"`
}

type OrgResponse struct {
	Name string `json:"name"`
	GUID string `json:"guid"`

	CreatedAt     string        `json:"created_at"`
	UpdatedAt     string        `json:"updated_at"`
	Suspended     bool          `json:"suspended"`
	Relationships Relationships `json:"relationships"`
	Metadata      Metadata      `json:"metadata"`
	Links         OrgLinks      `json:"links"`
}

type OrgLinks struct {
	Self          *Link `json:"self"`
	Domains       *Link `json:"domains,omitempty"`
	DefaultDomain *Link `json:"default_domain,omitempty"`
	Quota         *Link `json:"quota,omitempty"`
}

type SpaceListResponse struct {
	Pagination PaginationData  `json:"pagination"`
	Resources  []SpaceResponse `json:"resources"`
}

type SpaceResponse struct {
	Name          string        `json:"name"`
	GUID          string        `json:"guid"`
	CreatedAt     string        `json:"created_at"`
	UpdatedAt     string        `json:"updated_at"`
	Links         SpaceLinks    `json:"links"`
	Metadata      Metadata      `json:"metadata"`
	Relationships Relationships `json:"relationships"`
}

type SpaceLinks struct {
	Self         *Link `json:"self"`
	Organization *Link `json:"organization"`
}

func ForOrgList(orgs []repositories.OrgRecord, apiBaseURL string) OrgListResponse {
	baseURL, _ := neturl.Parse(apiBaseURL)
	baseURL.Path = orgsBase
	baseURL.RawQuery = "page=1"

	selfLink, _ := neturl.Parse(apiBaseURL)

	orgResponses := []OrgResponse{}
	for _, org := range orgs {
		selfLink.Path = path.Join(orgsBase, org.GUID)
		orgResponses = append(orgResponses, OrgResponse{
			Name:      org.Name,
			GUID:      org.GUID,
			CreatedAt: org.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt: org.CreatedAt.UTC().Format(time.RFC3339),
			Metadata: Metadata{
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
			Relationships: Relationships{},
			Links: OrgLinks{
				Self: &Link{
					HREF: selfLink.String(),
				},
			},
		})
	}

	return OrgListResponse{
		Pagination: PaginationData{
			TotalResults: len(orgs),
			TotalPages:   1,
			First: PageRef{
				HREF: prefixedLinkURL(apiBaseURL, "v3/organizations?page=1"),
			},
			Last: PageRef{
				HREF: prefixedLinkURL(apiBaseURL, "v3/organizations?page=1"),
			},
		},
		Resources: orgResponses,
	}
}

func ForSpaceList(spaces []repositories.SpaceRecord, apiBaseURL string) SpaceListResponse {
	baseURL, _ := neturl.Parse(apiBaseURL)
	baseURL.Path = spacesBase
	baseURL.RawQuery = "page=1"

	selfLink, _ := neturl.Parse(apiBaseURL)
	orgLink, _ := neturl.Parse(apiBaseURL)

	spaceResponses := []SpaceResponse{}
	for _, space := range spaces {
		selfLink.Path = path.Join(spacesBase, space.GUID)
		orgLink.Path = path.Join(orgsBase, space.OrganizationGUID)
		spaceResponses = append(spaceResponses, SpaceResponse{
			Name:      space.Name,
			GUID:      space.GUID,
			CreatedAt: space.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt: space.CreatedAt.UTC().Format(time.RFC3339),
			Metadata: Metadata{
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
			Relationships: Relationships{
				"organization": Relationship{
					Data: RelationshipData{
						GUID: space.OrganizationGUID,
					},
				},
			},
			Links: SpaceLinks{
				Self: &Link{
					HREF: selfLink.String(),
				},
				Organization: &Link{
					HREF: orgLink.String(),
				},
			},
		})
	}

	return SpaceListResponse{
		Pagination: PaginationData{
			TotalResults: len(spaces),
			TotalPages:   1,
			First: PageRef{
				HREF: prefixedLinkURL(apiBaseURL, "v3/spaces?page=1"),
			},
			Last: PageRef{
				HREF: prefixedLinkURL(apiBaseURL, "v3/spaces?page=1"),
			},
		},
		Resources: spaceResponses,
	}
}
