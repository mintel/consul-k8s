package vault

import (
	"testing"

	"github.com/hashicorp/consul-k8s/acceptance/framework/consul"
	"github.com/hashicorp/consul-k8s/acceptance/framework/helpers"
	"github.com/hashicorp/consul-k8s/acceptance/framework/k8s"
	"github.com/hashicorp/consul-k8s/acceptance/framework/logger"
	"github.com/hashicorp/consul-k8s/acceptance/framework/vault"
)

// TestVault installs Vault, bootstraps it with secrets, policies, and Kube Auth Method.
// It then configures Consul to use vault as the backend and checks that it works.
func TestVault_SnapshotAgent(t *testing.T) {
	cfg := suite.Config()
	ctx := suite.Environment().DefaultContext(t)
	ns := ctx.KubectlOptions(t).Namespace

	consulReleaseName := helpers.RandomName()
	vaultReleaseName := helpers.RandomName()

	vaultCluster := vault.NewVaultCluster(t, ctx, cfg, vaultReleaseName, nil)
	vaultCluster.Create(t, ctx)
	// Vault is now installed in the cluster.

	// Now fetch the Vault client so we can create the policies and secrets.
	vaultClient := vaultCluster.VaultClient(t)

	configureGossipVaultSecret(t, vaultClient)

	createConnectCAPolicy(t, vaultClient, "dc1")
	if cfg.EnableEnterprise {
		configureEnterpriseLicenseVaultSecret(t, vaultClient, cfg)
	}

	configureSnapshotAgentConfigVaultSecret(t, vaultClient)

	configureKubernetesAuthRoles(t, vaultClient, consulReleaseName, ns, "kubernetes", "dc1", cfg)

	configurePKICA(t, vaultClient)
	certPath := configurePKICertificates(t, vaultClient, consulReleaseName, ns, "dc1")

	vaultCASecret := vault.CASecretName(vaultReleaseName)

	consulHelmValues := map[string]string{
		"server.extraVolumes[0].type": "secret",
		"server.extraVolumes[0].name": vaultCASecret,
		"server.extraVolumes[0].load": "false",

		"connectInject.enabled":  "true",
		"connectInject.replicas": "1",
		"controller.enabled":     "true",

		"global.secretsBackend.vault.enabled":          "true",
		"global.secretsBackend.vault.consulServerRole": "consul-server",
		"global.secretsBackend.vault.consulClientRole": "consul-client",
		"global.secretsBackend.vault.consulCARole":     "consul-ca",

		"global.secretsBackend.vault.ca.secretName": vaultCASecret,
		"global.secretsBackend.vault.ca.secretKey":  "tls.crt",

		"global.secretsBackend.vault.connectCA.address":             vaultCluster.Address(),
		"global.secretsBackend.vault.connectCA.rootPKIPath":         "connect_root",
		"global.secretsBackend.vault.connectCA.intermediatePKIPath": "dc1/connect_inter",

		"global.acls.manageSystemACLs":       "true",
		"global.tls.enabled":                 "true",
		"global.gossipEncryption.secretName": "consul/data/secret/gossip",
		"global.gossipEncryption.secretKey":  "gossip",

		// "ingressGateways.enabled":               "true",
		// "ingressGateways.defaults.replicas":     "1",
		// "terminatingGateways.enabled":           "true",
		// "terminatingGateways.defaults.replicas": "1",

		"server.serverCert.secretName": certPath,
		"global.tls.caCert.secretName": "pki/cert/ca",
		"global.tls.enableAutoEncrypt": "true",

		// // For sync catalog, it is sufficient to check that the deployment is running and ready
		// // because we only care that get-auto-encrypt-client-ca init container was able
		// // to talk to the Consul server using the CA from Vault. For this reason,
		// // we don't need any services to be synced in either direction.
		// "syncCatalog.enabled":  "true",
		// "syncCatalog.toConsul": "false",
		// "syncCatalog.toK8S":    "false",

		"client.snapshotAgent.enabled":    "true",
		"client.snapshotAgent.secretName": "consul/data/secret/snapshotagentconfig",
		"client.snapshotAgent.secretKey":  "snapshotagentconfig",
	}

	if cfg.EnableEnterprise {
		consulHelmValues["global.enterpriseLicense.secretName"] = "consul/data/secret/enterpriselicense"
		consulHelmValues["global.enterpriseLicense.secretKey"] = "enterpriselicense"
	}

	logger.Log(t, "Installing Consul")
	consulCluster := consul.NewHelmCluster(t, consulHelmValues, ctx, cfg, consulReleaseName)
	consulCluster.Create(t)

	// Deploy two services and check that they can talk to each other.
	logger.Log(t, "creating static-server and static-client deployments")
	k8s.DeployKustomize(t, ctx.KubectlOptions(t), cfg.NoCleanupOnFailure, cfg.DebugDirectory, "../fixtures/cases/static-server-inject")
	if cfg.EnableTransparentProxy {
		k8s.DeployKustomize(t, ctx.KubectlOptions(t), cfg.NoCleanupOnFailure, cfg.DebugDirectory, "../fixtures/cases/static-client-tproxy")
	} else {
		k8s.DeployKustomize(t, ctx.KubectlOptions(t), cfg.NoCleanupOnFailure, cfg.DebugDirectory, "../fixtures/cases/static-client-inject")
	}
	helpers.Cleanup(t, cfg.NoCleanupOnFailure, func() {
		k8s.KubectlDeleteK(t, ctx.KubectlOptions(t), "../fixtures/bases/intention")
	})
	k8s.KubectlApplyK(t, ctx.KubectlOptions(t), "../fixtures/bases/intention")

	logger.Log(t, "checking that connection is successful")
	if cfg.EnableTransparentProxy {
		k8s.CheckStaticServerConnectionSuccessful(t, ctx.KubectlOptions(t), staticClientName, "http://static-server")
	} else {
		k8s.CheckStaticServerConnectionSuccessful(t, ctx.KubectlOptions(t), staticClientName, "http://localhost:1234")
	}
}
