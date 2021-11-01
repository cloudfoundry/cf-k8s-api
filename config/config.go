package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServerURL                 string `yaml:"serverURL"`
	ServerPort                int    `yaml:"serverPort"`
	RootNamespace             string `yaml:"rootNamespace"`
	PackageRegistryBase       string `yaml:"packageRegistryBase"`
	PackageRegistrySecretName string `yaml:"packageRegistrySecretName"`

	DefaultLifecycleConfig DefaultLifecycleConfig `yaml:"defaultLifecycleConfig"`

	AuthEnabled  bool         `yaml:"authEnabled"`
	RoleMappings RoleMappings `yaml:"roleMappings"`
}

type RoleMappings struct {
	AdminReadOnly              string `yaml:"admin_read_only"`
	Admin                      string `yaml:"admin"`
	GlobalAuditor              string `yaml:"global_auditor"`
	OrganizationAuditor        string `yaml:"organization_auditor"`
	OrganizationBillingManager string `yaml:"organization_billing_manager"`
	OrganizationUser           string `yaml:"organization_user"`
	SpaceAuditor               string `yaml:"space_auditor"`
	SpaceDeveloper             string `yaml:"space_developer"`
	SpaceManager               string `yaml:"space_manager"`
	SpaceSupporter             string `yaml:"space_supporter"`
}

// DefaultLifecycleConfig contains default values of the Lifecycle block of CFApps and Builds created by the Shim
type DefaultLifecycleConfig struct {
	Type            string `yaml:"type"`
	Stack           string `yaml:"stack"`
	StagingMemoryMB int    `yaml:"stagingMemoryMB"`
	StagingDiskMB   int    `yaml:"stagingDiskMB"`
}

func LoadFromPath(path string) (*Config, error) {
	var config Config

	items, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config dir %q: %w", path, err)
	}

	for _, item := range items {
		fileName := item.Name()
		if item.IsDir() || strings.HasPrefix(fileName, ".") {
			continue
		}

		configFile, err := os.Open(filepath.Join(path, fileName))
		if err != nil {
			return &config, fmt.Errorf("failed to open file: %w", err)
		}
		defer configFile.Close()

		decoder := yaml.NewDecoder(configFile)
		if err = decoder.Decode(&config); err != nil {
			return nil, fmt.Errorf("failed decoding %q: %w", item.Name(), err)
		}
	}

	return &config, nil
}
