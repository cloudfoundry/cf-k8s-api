package messages

type Lifecycle struct {
	Data LifecycleData `json:"data"`
}

type LifecycleData struct {
	Buildpacks []string `json:"buildpacks"`
	Stack      string   `json:"stack"`
}

type Relationships map[string]Relationship

type Relationship struct {
	GUID string `json:"guid" validate:"required"`
}

type Metadata struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}
