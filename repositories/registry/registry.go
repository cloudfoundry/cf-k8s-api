package registry

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	coordv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type RegistryType string

const OrgType RegistryType = "Org"

type Registrar struct {
	client            client.Client
	registryNamespace string
}

func NewRegistrar(client client.Client, namespace string) *Registrar {
	return &Registrar{
		client:            client,
		registryNamespace: namespace,
	}
}

func (l *Registrar) TryRegister(ctx context.Context, registryType RegistryType, namespace, name string) (string, error) {
	leaseName := getLeaseName(registryType, namespace, name)
	holder := "cf-name-registry"
	now := metav1.NewMicroTime(time.Now())

	lease := coordv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      leaseName,
			Namespace: namespace,
			Labels: map[string]string{
				"type": string(registryType),
				"name": name,
			},
		},
		Spec: coordv1.LeaseSpec{
			HolderIdentity: &holder,
			AcquireTime:    &now,
		},
	}

	err := l.client.Create(ctx, &lease)
	if err != nil {
		return "", err
	}

	return lease.Name, nil
}

func (l *Registrar) SetOwnerRef(ctx context.Context, owner client.Object, registrationCode string) error {
	lease := &coordv1.Lease{}
	lease.Namespace = l.registryNamespace
	lease.Name = registrationCode
	leaseCopy := lease.DeepCopy()

	err := controllerutil.SetOwnerReference(owner, leaseCopy, scheme.Scheme)
	if err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	err = l.client.Patch(ctx, leaseCopy, client.MergeFrom(lease))
	if err != nil {
		return fmt.Errorf("failed to patch lease with owner ref: %w", err)
	}

	fmt.Printf("leaseCopy = %+v\n", leaseCopy)
	return nil
}

func getLeaseName(registryType RegistryType, namespace, name string) string {
	plain := []byte(string(registryType) + "::" + namespace + "::" + name)
	sum := sha256.Sum256(plain)

	return fmt.Sprintf("r-%x", sum)
}
