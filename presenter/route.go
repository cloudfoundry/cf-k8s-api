package presenter

import (
	"fmt"
	"net/url"

	"code.cloudfoundry.org/cf-k8s-api/repositories"
)

const (
	routesBase  = "/v3/routes"
	domainsBase = "/v3/domains"
)

type RouteResponse struct {
	GUID         string                     `json:"guid"`
	Protocol     string                     `json:"protocol"`
	Port         *int                       `json:"port"`
	Host         string                     `json:"host"`
	Path         string                     `json:"path"`
	URL          string                     `json:"url"`
	Destinations []routeDestinationResponse `json:"destinations"`

	CreatedAt     string        `json:"created_at"`
	UpdatedAt     string        `json:"updated_at"`
	Relationships Relationships `json:"relationships"`
	Metadata      Metadata      `json:"metadata"`
	Links         routeLinks    `json:"links"`
}

type RouteDestinationsResponse struct {
	Destinations []routeDestinationResponse `json:"destinations"`
	Links        routeDestinationsLinks     `json:"links"`
}

type routeDestinationResponse struct {
	GUID     string              `json:"guid"`
	App      routeDestinationApp `json:"app"`
	Weight   *int                `json:"weight"`
	Port     int                 `json:"port"`
	Protocol string              `json:"protocol"`
}

type routeDestinationApp struct {
	AppGUID string                     `json:"guid"`
	Process routeDestinationAppProcess `json:"process"`
}

type routeDestinationAppProcess struct {
	Type string `json:"type"`
}

type routeLinks struct {
	Self         Link `json:"self"`
	Space        Link `json:"space"`
	Domain       Link `json:"domain"`
	Destinations Link `json:"destinations"`
}

type routeDestinationsLinks struct {
	Self  Link `json:"self"`
	Route Link `json:"route"`
}

func ForRoute(route repositories.RouteRecord, baseURL url.URL) RouteResponse {
	destinations := make([]routeDestinationResponse, len(route.Destinations))
	for _, destinationRecord := range route.Destinations {
		destinations = append(destinations, ForDestination(destinationRecord))
	}
	return RouteResponse{
		GUID:      route.GUID,
		Protocol:  route.Protocol,
		Host:      route.Host,
		Path:      route.Path,
		URL:       routeURL(route),
		CreatedAt: route.CreatedAt,
		UpdatedAt: route.UpdatedAt,
		Relationships: Relationships{
			"space": Relationship{
				Data: RelationshipData{
					GUID: route.SpaceGUID,
				},
			},
			"domain": Relationship{
				Data: RelationshipData{
					GUID: route.DomainRef.GUID,
				},
			},
		},
		Destinations: destinations,
		Metadata: Metadata{
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		Links: routeLinks{
			Self: Link{
				HREF: buildURL(baseURL).appendPath(routesBase, route.GUID).build(),
			},
			Space: Link{
				HREF: buildURL(baseURL).appendPath(spacesBase, route.SpaceGUID).build(),
			},
			Domain: Link{
				HREF: buildURL(baseURL).appendPath(domainsBase, route.DomainRef.GUID).build(),
			},
			Destinations: Link{
				HREF: buildURL(baseURL).appendPath(routesBase, route.GUID, "destinations").build(),
			},
		},
	}
}

func ForDestination(destination repositories.Destination) routeDestinationResponse {
	return routeDestinationResponse{
		GUID: destination.GUID,
		App: routeDestinationApp{
			AppGUID: destination.AppGUID,
			Process: routeDestinationAppProcess{
				Type: destination.ProcessType,
			},
		},
		Weight:   nil,
		Port:     destination.Port,
		Protocol: "http1",
	}
}

func ForRouteDestinations(route repositories.RouteRecord, baseURL url.URL) RouteDestinationsResponse {
	destinations := []routeDestinationResponse{}
	for _, destinationRecord := range route.Destinations {
		destinations = append(destinations, ForDestination(destinationRecord))
	}
	return RouteDestinationsResponse{
		Destinations: destinations,
		Links: routeDestinationsLinks{
			Self: Link{
				HREF: buildURL(baseURL).appendPath(routesBase, route.GUID, "destinations").build(),
			},
			Route: Link{
				HREF: buildURL(baseURL).appendPath(routesBase, route.GUID).build(),
			},
		},
	}
}

func routeURL(route repositories.RouteRecord) string {
	if route.Host != "" {
		return fmt.Sprintf("%s.%s%s", route.Host, route.DomainRef.Name, route.Path)
	} else {
		return fmt.Sprintf("%s%s", route.DomainRef.Name, route.Path)
	}
}
