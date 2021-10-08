package presenter

import (
	"net/url"
	"path"
)

type Lifecycle struct {
	Type string        `json:"type"`
	Data LifecycleData `json:"data"`
}

type LifecycleData struct {
	Buildpacks []string `json:"buildpacks"`
	Stack      string   `json:"stack"`
}

type Relationships map[string]Relationship

type Relationship struct {
	Data RelationshipData `json:"data"`
}

type RelationshipData struct {
	GUID string `json:"guid"`
}

type Metadata struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

type Link struct {
	HREF   string `json:"href,omitempty"`
	Method string `json:"method,omitempty"`
}

type PaginationData struct {
	TotalResults int     `json:"total_results"`
	TotalPages   int     `json:"total_pages"`
	First        PageRef `json:"first"`
	Last         PageRef `json:"last"`
	Next         *int    `json:"next"`
	Previous     *int    `json:"previous"`
}

type PageRef struct {
	HREF string `json:"href"`
}

type buildURL url.URL

func (u buildURL) appendPath(subpath ...string) buildURL {
	rest := path.Join(subpath...)
	if u.Path == "" {
		u.Path = rest
	} else {
		u.Path = path.Join(u.Path, rest)
	}

	return u
}

func (u buildURL) setQuery(rawQuery string) buildURL {
	u.RawQuery = rawQuery

	return u
}

func (u buildURL) build() string {
	native := url.URL(u)
	nativeP := &native

	return nativeP.String()
}
