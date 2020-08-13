// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package containerd_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/talos-systems/crypto/x509"

	"github.com/talos-systems/talos/internal/pkg/containers/cri/containerd"
	"github.com/talos-systems/talos/pkg/config"
	"github.com/talos-systems/talos/pkg/config/types/v1alpha1"
	"github.com/talos-systems/talos/pkg/constants"
)

type mockConfig struct {
	mirrors map[string]*v1alpha1.RegistryMirrorConfig
	config  map[string]*v1alpha1.RegistryConfig
}

// Mirrors implements the Registries interface.
func (c *mockConfig) Mirrors() map[string]config.RegistryMirrorConfig {
	mirrors := make(map[string]config.RegistryMirrorConfig, len(c.mirrors))

	for k, v := range c.mirrors {
		mirrors[k] = v
	}

	return mirrors
}

// Config implements the Registries interface.
func (c *mockConfig) Config() map[string]config.RegistryConfig {
	registries := make(map[string]config.RegistryConfig, len(c.config))

	for k, v := range c.config {
		registries[k] = v
	}

	return registries
}

type ConfigSuite struct {
	suite.Suite
}

func (suite *ConfigSuite) TestGenerateRegistriesConfig() {
	cfg := &mockConfig{
		mirrors: map[string]*v1alpha1.RegistryMirrorConfig{
			"docker.io": {
				MirrorEndpoints: []string{"https://registry-1.docker.io", "https://registry-2.docker.io"},
			},
		},
		config: map[string]*v1alpha1.RegistryConfig{
			"some.host:123": {
				RegistryAuth: &v1alpha1.RegistryAuthConfig{
					RegistryUsername:      "root",
					RegistryPassword:      "secret",
					RegistryAuth:          "auth",
					RegistryIdentityToken: "token",
				},
				RegistryTLS: &v1alpha1.RegistryTLSConfig{
					TLSInsecureSkipVerify: true,
					TLSCA:                 []byte("cacert"),
					TLSClientIdentity: &x509.PEMEncodedCertificateAndKey{
						Crt: []byte("clientcert"),
						Key: []byte("clientkey"),
					},
				},
			},
		},
	}

	files, err := containerd.GenerateRegistriesConfig(cfg)
	suite.Require().NoError(err)
	suite.Assert().Equal([]config.File{
		&v1alpha1.MachineFile{
			FileContent:     `cacert`,
			FilePermissions: 0o600,
			FilePath:        "/etc/cri/ca/some.host:123.crt",
			FileOp:          "create",
		},
		&v1alpha1.MachineFile{
			FileContent:     `clientcert`,
			FilePermissions: 0o600,
			FilePath:        "/etc/cri/client/some.host:123.crt",
			FileOp:          "create",
		},
		&v1alpha1.MachineFile{
			FileContent:     `clientkey`,
			FilePermissions: 0o600,
			FilePath:        "/etc/cri/client/some.host:123.key",
			FileOp:          "create",
		},
		&v1alpha1.MachineFile{
			FileContent: `[plugins]
  [plugins.cri]
    [plugins.cri.registry]
      [plugins.cri.registry.mirrors]
        [plugins.cri.registry.mirrors."docker.io"]
          endpoint = ["https://registry-1.docker.io", "https://registry-2.docker.io"]
      [plugins.cri.registry.configs]
        [plugins.cri.registry.configs."some.host:123"]
          [plugins.cri.registry.configs."some.host:123".auth]
            username = "root"
            password = "secret"
            auth = "auth"
            identitytoken = "token"
          [plugins.cri.registry.configs."some.host:123".tls]
            insecure_skip_verify = true
            ca_file = "/etc/cri/ca/some.host:123.crt"
            cert_file = "/etc/cri/client/some.host:123.crt"
            key_file = "/etc/cri/client/some.host:123.key"
`,
			FilePermissions: 0o644,
			FilePath:        constants.CRIContainerdConfig,
			FileOp:          "append",
		},
	}, files)
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}
