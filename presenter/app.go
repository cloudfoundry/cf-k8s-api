package presenter

import (
	"net/url"

	"code.cloudfoundry.org/cf-k8s-api/repositories"
)

const (
	appsBase = "/v3/apps"
)

type AppResponse struct {
	Name  string `json:"name"`
	GUID  string `json:"guid"`
	State string `json:"state"`

	CreatedAt     string        `json:"created_at"`
	UpdatedAt     string        `json:"updated_at"`
	Relationships Relationships `json:"relationships"`
	Lifecycle     Lifecycle     `json:"lifecycle"`
	Metadata      Metadata      `json:"metadata"`
	Links         AppLinks      `json:"links"`
}

type AppLinks struct {
	Self                 Link `json:"self"`
	Space                Link `json:"space"`
	Processes            Link `json:"processes"`
	Packages             Link `json:"packages"`
	EnvironmentVariables Link `json:"environment_variables"`
	CurrentDroplet       Link `json:"current_droplet"`
	Droplets             Link `json:"droplets"`
	Tasks                Link `json:"tasks"`
	StartAction          Link `json:"start"`
	StopAction           Link `json:"stop"`
	Revisions            Link `json:"revisions"`
	DeployedRevisions    Link `json:"deployed_revisions"`
	Features             Link `json:"features"`
}

type AppListResponse struct {
	PaginationData PaginationData `json:"pagination"`
	Resources      []AppResponse  `json:"resources"`
}

func ForApp(responseApp repositories.AppRecord, baseURL url.URL) AppResponse {
	return AppResponse{
		Name:      responseApp.Name,
		GUID:      responseApp.GUID,
		State:     string(responseApp.State),
		CreatedAt: responseApp.CreatedAt,
		UpdatedAt: responseApp.UpdatedAt,
		Relationships: Relationships{
			"space": Relationship{
				Data: &RelationshipData{
					GUID: responseApp.SpaceGUID,
				},
			},
		},
		Lifecycle: Lifecycle{
			Type: responseApp.Lifecycle.Type,
			Data: LifecycleData{
				Buildpacks: responseApp.Lifecycle.Data.Buildpacks,
				Stack:      responseApp.Lifecycle.Data.Stack,
			},
		},
		Metadata: Metadata{
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		Links: AppLinks{
			Self: Link{
				HREF: buildURL(baseURL).appendPath(appsBase, responseApp.GUID).build(),
			},
			Space: Link{
				HREF: buildURL(baseURL).appendPath(spacesBase, responseApp.SpaceGUID).build(),
			},
			Processes: Link{
				HREF: buildURL(baseURL).appendPath(appsBase, responseApp.GUID, "processes").build(),
			},
			Packages: Link{
				HREF: buildURL(baseURL).appendPath(appsBase, responseApp.GUID, "packages").build(),
			},
			EnvironmentVariables: Link{
				HREF: buildURL(baseURL).appendPath(appsBase, responseApp.GUID, "environment_variables").build(),
			},
			CurrentDroplet: Link{
				HREF: buildURL(baseURL).appendPath(appsBase, responseApp.GUID, "droplets/current").build(),
			},
			Droplets: Link{
				HREF: buildURL(baseURL).appendPath(appsBase, responseApp.GUID, "droplets").build(),
			},
			Tasks: Link{
				HREF: buildURL(baseURL).appendPath(appsBase, responseApp.GUID, "tasks").build(),
			},
			StartAction: Link{
				HREF:   buildURL(baseURL).appendPath(appsBase, responseApp.GUID, "actions/start").build(),
				Method: "POST",
			},
			StopAction: Link{
				HREF:   buildURL(baseURL).appendPath(appsBase, responseApp.GUID, "actions/stop").build(),
				Method: "POST",
			},
			Revisions: Link{
				HREF: buildURL(baseURL).appendPath(appsBase, responseApp.GUID, "revisions").build(),
			},
			DeployedRevisions: Link{
				HREF: buildURL(baseURL).appendPath(appsBase, responseApp.GUID, "revisions/deployed").build(),
			},
			Features: Link{
				HREF: buildURL(baseURL).appendPath(appsBase, responseApp.GUID, "features").build(),
			},
		},
	}
}

func ForAppList(appRecordList []repositories.AppRecord, baseURL url.URL) AppListResponse {
	appResponses := make([]AppResponse, 0, len(appRecordList))
	for _, app := range appRecordList {
		appResponses = append(appResponses, ForApp(app, baseURL))
	}

	appListResponse := AppListResponse{
		PaginationData: PaginationData{
			TotalResults: len(appResponses),
			TotalPages:   1,
			First: PageRef{
				HREF: buildURL(baseURL).appendPath(appsBase).setQuery("page=1").build(),
			},
			Last: PageRef{
				HREF: buildURL(baseURL).appendPath(appsBase).setQuery("page=1").build(),
			},
		},
		Resources: appResponses,
	}

	return appListResponse
}

type CurrentDropletResponse struct {
	Relationship `json:",inline"`
	Links        CurrentDropletLinks `json:"links"`
}

type CurrentDropletLinks struct {
	Self    Link `json:"self"`
	Related Link `json:"related"`
}

func ForCurrentDroplet(record repositories.CurrentDropletRecord, baseURL url.URL) CurrentDropletResponse {
	return CurrentDropletResponse{
		Relationship: Relationship{
			Data: &RelationshipData{
				GUID: record.DropletGUID,
			},
		},
		Links: CurrentDropletLinks{
			Self: Link{
				HREF: buildURL(baseURL).appendPath(appsBase, record.AppGUID, "relationships/current_droplet").build(),
			},
			Related: Link{
				HREF: buildURL(baseURL).appendPath(appsBase, record.AppGUID, "droplets/current").build(),
			},
		},
	}
}
