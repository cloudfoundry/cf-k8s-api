package presenter

import (
	"encoding/json"
	"net/http"
	"net/url"

	"code.cloudfoundry.org/cf-k8s-api/repositories"
)

const (
	processesBase = "/v3/processes"
)

type ProcessResponse struct {
	GUID          string                     `json:"guid"`
	Type          string                     `json:"type"`
	Command       string                     `json:"command"`
	Instances     int                        `json:"instances"`
	MemoryMB      int64                      `json:"memory_in_mb"`
	DiskQuotaMB   int64                      `json:"disk_in_mb"`
	HealthCheck   ProcessResponseHealthCheck `json:"health_check"`
	Relationships Relationships              `json:"relationships"`
	Metadata      Metadata                   `json:"metadata"`
	CreatedAt     string                     `json:"created_at"`
	UpdatedAt     string                     `json:"updated_at"`
	Links         ProcessLinks               `json:"links"`
}

type ProcessLinks struct {
	Self  Link `json:"self"`
	Scale Link `json:"scale"`
	App   Link `json:"app"`
	Space Link `json:"space"`
	Stats Link `json:"stats"`
}

type ProcessResponseHealthCheck struct {
	Type string                         `json:"type"`
	Data ProcessResponseHealthCheckData `json:"data"`
}

type ProcessResponseHealthCheckData struct {
	Type              string `json:"-"`
	Timeout           int64  `json:"timeout"`
	InvocationTimeout int64  `json:"invocation_timeout"`
	HTTPEndpoint      string `json:"endpoint"`
}

func (h ProcessResponseHealthCheckData) MarshalJSON() ([]byte, error) {
	timeout := &(h.Timeout)
	if *timeout == 0 {
		timeout = nil
	}
	invocationTimeout := &(h.InvocationTimeout)
	if *invocationTimeout == 0 {
		invocationTimeout = nil
	}

	switch h.Type {
	case "http":
		return json.Marshal(ProcessResponseHTTPHealthCheckData{
			Timeout:           timeout,
			InvocationTimeout: invocationTimeout,
			HTTPEndpoint:      h.HTTPEndpoint,
		})
	case "port":
		return json.Marshal(ProcessResponsePortHealthCheckData{
			Timeout:           timeout,
			InvocationTimeout: invocationTimeout,
		})
	case "process":
		return json.Marshal(ProcessResponseProcessHealthCheckData{
			Timeout: timeout,
		})
	default:
		return json.Marshal(h)
	}
}

type ProcessResponseHTTPHealthCheckData struct {
	Timeout           *int64 `json:"timeout"`
	InvocationTimeout *int64 `json:"invocation_timeout"`
	HTTPEndpoint      string `json:"endpoint"`
}

type ProcessResponsePortHealthCheckData struct {
	Timeout           *int64 `json:"timeout"`
	InvocationTimeout *int64 `json:"invocation_timeout"`
}

type ProcessResponseProcessHealthCheckData struct {
	Timeout *int64 `json:"timeout"`
}

type ProcessListResponse struct {
	PaginationData PaginationData    `json:"pagination"`
	Resources      []ProcessResponse `json:"resources"`
}

func ForProcess(responseProcess repositories.ProcessRecord, baseURL url.URL) ProcessResponse {
	return ProcessResponse{
		GUID:        responseProcess.GUID,
		Type:        responseProcess.Type,
		Command:     responseProcess.Command,
		Instances:   responseProcess.Instances,
		MemoryMB:    responseProcess.MemoryMB,
		DiskQuotaMB: responseProcess.DiskQuotaMB,
		HealthCheck: ProcessResponseHealthCheck{
			Type: string(responseProcess.HealthCheck.Type),
			Data: ProcessResponseHealthCheckData{
				Type:              string(responseProcess.HealthCheck.Type),
				Timeout:           responseProcess.HealthCheck.Data.TimeoutSeconds,
				InvocationTimeout: responseProcess.HealthCheck.Data.InvocationTimeoutSeconds,
				HTTPEndpoint:      responseProcess.HealthCheck.Data.HTTPEndpoint,
			},
		},
		Relationships: map[string]Relationship{
			"app": {
				Data: &RelationshipData{
					GUID: responseProcess.AppGUID,
				},
			},
		},
		Metadata: Metadata{
			Labels:      responseProcess.Labels,
			Annotations: responseProcess.Annotations,
		},
		CreatedAt: responseProcess.CreatedAt,
		UpdatedAt: responseProcess.UpdatedAt,
		Links: ProcessLinks{
			Self: Link{
				HREF: buildURL(baseURL).appendPath(processesBase, responseProcess.GUID).build(),
			},
			Scale: Link{
				HREF:   buildURL(baseURL).appendPath(processesBase, responseProcess.GUID, "actions", "scale").build(),
				Method: http.MethodPost,
			},
			App: Link{
				HREF: buildURL(baseURL).appendPath(appsBase, responseProcess.AppGUID).build(),
			},
			Space: Link{
				HREF: buildURL(baseURL).appendPath(spacesBase, responseProcess.SpaceGUID).build(),
			},
			Stats: Link{
				HREF: buildURL(baseURL).appendPath(processesBase, responseProcess.GUID, "stats").build(),
			},
		},
	}
}

func ForProcessList(processRecordList []repositories.ProcessRecord, baseURL url.URL, appGUID string) ProcessListResponse {
	processResponses := make([]ProcessResponse, 0, len(processRecordList))
	for _, process := range processRecordList {
		processResponse := ForProcess(process, baseURL)
		processResponse.Command = "[PRIVATE DATA HIDDEN IN LISTS]"
		processResponses = append(processResponses, processResponse)
	}

	pageHREF := buildURL(baseURL).appendPath(appsBase, appGUID, "processes").setQuery("page=1").build()
	processListResponse := ProcessListResponse{
		PaginationData: PaginationData{
			TotalResults: len(processResponses),
			TotalPages:   1,
			First: PageRef{
				HREF: pageHREF,
			},
			Last: PageRef{
				HREF: pageHREF,
			},
		},
		Resources: processResponses,
	}

	return processListResponse
}
