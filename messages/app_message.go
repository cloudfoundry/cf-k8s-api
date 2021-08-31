package messages

type AppCreateMessage struct {
	Name          string        `json:"name" validate:"required"`
	Relationships Relationships `json:"relationships" validate:"required"`
	Lifecycle     Lifecycle     `json:"lifecycle"`
	Metadata      Metadata      `json:"metadata"`
}
