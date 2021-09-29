package payloads

type BuildCreate struct {
	Package         *RelationshipData `json:"package" validate:"required"`
	StagingMemoryMB *int              `json:"staging_memory_in_mb"`
	StagingDiskMB   *int              `json:"staging_disk_in_mb"`
	Lifecycle       *Lifecycle        `json:"lifecycle"`
	Metadata        Metadata          `json:"metadata"`
}
