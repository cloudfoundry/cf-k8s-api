package repositories

import (
	"context"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=create
type RoleRecord struct {
	GUID      string
	CreatedAt time.Time
	UpdatedAt time.Time
	Type      string
	Space     string
	User      string
}

type RoleRepo struct {
	privilegedClient client.Client
}

func NewRoleRepo(privilegedClient client.Client) *RoleRepo {
	return &RoleRepo{
		privilegedClient: privilegedClient,
	}
}

func (r *RoleRepo) CreateSpaceRole(ctx context.Context, role RoleRecord) (RoleRecord, error) {
	return RoleRecord{}, nil
}
