// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

/*
Package v1alpha1 configuration file contains all the options available for configuring a machine.

To generate a set of basic configuration files, run:
```bash
talosctl gen config --version v1alpha1 <cluster name> <cluster endpoint>
````

This will generate a machine config for each node type, and a talosconfig for the CLI.
*/
package v1alpha1

//go:generate docgen ./v1alpha1_types.go ./v1alpha1_types_doc.go Configuration

import (
	"net/url"
	"os"
	"strconv"
	"time"

	humanize "github.com/dustin/go-humanize"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/talos-systems/crypto/x509"
	yaml "gopkg.in/yaml.v3"

	"github.com/talos-systems/talos/pkg/machinery/config"
	"github.com/talos-systems/talos/pkg/machinery/config/types/v1alpha1/machine"
)

func init() {
	config.Register("v1alpha1", func(version string) (target interface{}) {
		target = &Config{}

		return target
	})
}

var (
	// Examples section.

	// this is using custom type to avoid generating full example with all the nested structs.
	configExample = struct {
		Version string `yaml:"version"`
		Persist bool
		Machine *yaml.Node
		Cluster *yaml.Node
	}{
		Version: "v1alpha1",
		Persist: true,
		Machine: &yaml.Node{Kind: yaml.ScalarNode, LineComment: "..."},
		Cluster: &yaml.Node{Kind: yaml.ScalarNode, LineComment: "..."},
	}

	machineConfigExample = struct {
		Type    string
		Install *InstallConfig
	}{
		Type:    machine.TypeControlPlane.String(),
		Install: machineInstallExample,
	}

	machineConfigRegistriesExample = &RegistriesConfig{
		RegistryMirrors: map[string]*RegistryMirrorConfig{
			"docker.io": {
				MirrorEndpoints: []string{"https://registry.local"},
			},
		},
		RegistryConfig: map[string]*RegistryConfig{
			"registry.local": {
				RegistryTLS: &RegistryTLSConfig{
					TLSClientIdentity: pemEncodedCertificateExample,
				},
				RegistryAuth: &RegistryAuthConfig{
					RegistryUsername: "username",
					RegistryPassword: "password",
				},
			},
		},
	}

	machineConfigRegistryMirrorsExample = map[string]*RegistryMirrorConfig{
		"ghcr.io": {
			MirrorEndpoints: []string{"https://registry.insecure", "https://ghcr.io/v2/"},
		},
	}

	machineConfigRegistryConfigExample = map[string]*RegistryConfig{
		"registry.insecure": {
			RegistryTLS: &RegistryTLSConfig{
				TLSInsecureSkipVerify: true,
			},
		},
	}

	machineConfigRegistryTLSConfigExample1 = &RegistryTLSConfig{
		TLSClientIdentity: pemEncodedCertificateExample,
	}

	machineConfigRegistryTLSConfigExample2 = &RegistryTLSConfig{
		TLSInsecureSkipVerify: true,
	}

	machineConfigRegistryAuthConfigExample = &RegistryAuthConfig{
		RegistryUsername: "username",
		RegistryPassword: "password",
	}

	pemEncodedCertificateExample *x509.PEMEncodedCertificateAndKey = &x509.PEMEncodedCertificateAndKey{
		Crt: []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJIekNCMHF..."),
		Key: []byte("LS0tLS1CRUdJTiBFRDI1NTE5IFBSSVZBVEUgS0VZLS0tLS0KTUM..."),
	}

	machineKubeletExample = &KubeletConfig{
		KubeletImage: (&KubeletConfig{}).Image(),
		KubeletExtraArgs: map[string]string{
			"--feature-gates": "ServerSideApply=true",
		},
	}

	kubeletImageExample = (&KubeletConfig{}).Image()

	machineNetworkConfigExample = &NetworkConfig{
		NetworkHostname: "worker-1",
		NetworkInterfaces: []*Device{
			{
				DeviceInterface: "eth0",
				DeviceCIDR:      "192.168.2.0/24",
				DeviceMTU:       1500,
				DeviceRoutes: []*Route{
					{
						RouteNetwork: "0.0.0.0/0",
						RouteGateway: "192.168.2.1",
						RouteMetric:  1024,
					},
				},
			},
		},
		NameServers: []string{"9.8.7.6", "8.7.6.5"},
	}

	machineDisksExample []*MachineDisk = []*MachineDisk{
		{
			DeviceName: "/dev/sdb",
			DiskPartitions: []*DiskPartition{
				{
					DiskMountPoint: "/var/mnt/extra",
				},
			},
		},
	}

	machineInstallExample = &InstallConfig{
		InstallDisk:            "/dev/sda",
		InstallExtraKernelArgs: []string{"console=ttyS1", "panic=10"},
		InstallImage:           "ghcr.io/talos-systems/installer:latest",
		InstallBootloader:      true,
		InstallWipe:            false,
	}

	machineFilesExample = []*MachineFile{
		{
			FileContent:     "...",
			FilePermissions: 0o666,
			FilePath:        "/tmp/file.txt",
			FileOp:          "append",
		},
	}

	machineEnvExamples = []Env{
		{
			"GRPC_GO_LOG_VERBOSITY_LEVEL": "99",
			"GRPC_GO_LOG_SEVERITY_LEVEL":  "info",
			"https_proxy":                 "http://SERVER:PORT/",
		},
		{
			"GRPC_GO_LOG_SEVERITY_LEVEL": "error",
			"https_proxy":                "https://USERNAME:PASSWORD@SERVER:PORT/",
		},
		{
			"https_proxy": "http://DOMAIN\\USERNAME:PASSWORD@SERVER:PORT/",
		},
	}

	machineTimeExample = &TimeConfig{
		TimeServers: []string{"time.cloudflare.com"},
	}

	machineSysctlsExample map[string]string = map[string]string{
		"kernel.domainname":   "talos.dev",
		"net.ipv4.ip_forward": "0",
	}

	clusterConfigExample = struct {
		ControlPlane *ControlPlaneConfig   `yaml:"controlPlane"`
		ClusterName  string                `yaml:"clusterName"`
		Network      *ClusterNetworkConfig `yaml:"network"`
	}{
		ControlPlane: clusterControlPlaneExample,
		ClusterName:  "talos.local",
		Network:      clusterNetworkExample,
	}

	clusterControlPlaneExample = &ControlPlaneConfig{
		Endpoint: &Endpoint{
			&url.URL{
				Host:   "1.2.3.4",
				Scheme: "https",
			},
		},
		LocalAPIServerPort: 443,
	}

	clusterNetworkExample = &ClusterNetworkConfig{
		CNI: &CNIConfig{
			CNIName: "flannel",
		},
		DNSDomain:     "cluster.local",
		PodSubnet:     []string{"10.244.0.0/16"},
		ServiceSubnet: []string{"10.96.0.0/12"},
	}

	clusterAPIServerExample = &APIServerConfig{
		ContainerImage: (&APIServerConfig{}).Image(),
		ExtraArgsConfig: map[string]string{
			"--feature-gates":                    "ServerSideApply=true",
			"--http2-max-streams-per-connection": "32",
		},
		CertSANs: []string{
			"1.2.3.4",
			"4.5.6.7",
		},
	}

	clusterControllerManagerExample = &ControllerManagerConfig{
		ContainerImage: (&ControllerManagerConfig{}).Image(),
		ExtraArgsConfig: map[string]string{
			"--feature-gates": "ServerSideApply=true",
		},
	}

	clusterProxyExample = &ProxyConfig{
		ContainerImage: (&ProxyConfig{}).Image(),
		ExtraArgsConfig: map[string]string{
			"--proxy-mode": "iptables",
		},
		ModeConfig: "ipvs",
	}

	clusterSchedulerConfig = &SchedulerConfig{
		ContainerImage: (&SchedulerConfig{}).Image(),
		ExtraArgsConfig: map[string]string{
			"--feature-gates": "AllBeta=true",
		},
	}

	clusterEtcdConfig = &EtcdConfig{
		ContainerImage: (&EtcdConfig{}).Image(),
		EtcdExtraArgs: map[string]string{
			"--election-timeout": "5000",
		},
		RootCA: pemEncodedCertificateExample,
	}

	clusterPodCheckpointerExample = &PodCheckpointer{
		PodCheckpointerImage: "...",
	}

	clusterCoreDNSExample = &CoreDNS{
		CoreDNSImage: (&CoreDNS{}).Image(),
	}

	clusterAdminKubeconfigExample = AdminKubeconfigConfig{
		AdminKubeconfigCertLifetime: time.Hour,
	}

	kubeletExtraMountsExample = []specs.Mount{
		{
			Source:      "/var/lib/example",
			Destination: "/var/lib/example",
			Type:        "bind",
			Options: []string{
				"rshared",
				"rw",
			},
		},
	}

	networkConfigExtraHostsExample = []*ExtraHost{
		{
			HostIP: "192.168.1.100",
			HostAliases: []string{
				"example",
				"example.domain.tld",
			},
		},
	}

	networkConfigRoutesExample = []*Route{
		{
			RouteNetwork: "0.0.0.0/0",
			RouteGateway: "10.5.0.1",
		},
		{
			RouteNetwork: "10.2.0.0/16",
			RouteGateway: "10.2.0.1",
		},
	}

	networkConfigBondExample = &Bond{
		BondMode:       "802.3ad",
		BondLACPRate:   "fast",
		BondInterfaces: []string{"eth0", "eth1"},
	}

	networkConfigDHCPOptionsExample = &DHCPOptions{
		DHCPRouteMetric: 1024,
	}

	clusterCustomCNIExample = &CNIConfig{
		CNIName: "custom",
		CNIUrls: []string{
			"https://raw.githubusercontent.com/cilium/cilium/v1.8/install/kubernetes/quick-install.yaml",
		},
	}
)

// Config defines the v1alpha1 configuration file.
//
//  examples:
//     - value: configExample
type Config struct {
	//   description: |
	//     Indicates the schema used to decode the contents.
	//   values:
	//     - "v1alpha1"
	ConfigVersion string `yaml:"version"`
	//   description: |
	//     Enable verbose logging to the console.
	//   values:
	//     - true
	//     - yes
	//     - false
	//     - no
	ConfigDebug bool `yaml:"debug"`
	//   description: |
	//     Indicates whether to pull the machine config upon every boot.
	//   values:
	//     - true
	//     - yes
	//     - false
	//     - no
	ConfigPersist bool `yaml:"persist"`
	//   description: |
	//     Provides machine specific configuration options.
	MachineConfig *MachineConfig `yaml:"machine"`
	//   description: |
	//     Provides cluster specific configuration options.
	ClusterConfig *ClusterConfig `yaml:"cluster"`
}

// MachineConfig represents the machine-specific config values.
//
//  examples:
//     - value: machineConfigExample
type MachineConfig struct {
	//   description: |
	//     Defines the role of the machine within the cluster.
	//
	//     #### Init
	//
	//     Init node type designates the first control plane node to come up.
	//     You can think of it like a bootstrap node.
	//     This node will perform the initial steps to bootstrap the cluster -- generation of TLS assets, starting of the control plane, etc.
	//
	//     #### Control Plane
	//
	//     Control Plane node type designates the node as a control plane member.
	//     This means it will host etcd along with the Kubernetes master components such as API Server, Controller Manager, Scheduler.
	//
	//     #### Worker
	//
	//     Worker node type designates the node as a worker node.
	//     This means it will be an available compute node for scheduling workloads.
	//   values:
	//     - "init"
	//     - "controlplane"
	//     - "join"
	MachineType string `yaml:"type"`
	//   description: |
	//     The `token` is used by a machine to join the PKI of the cluster.
	//     Using this token, a machine will create a certificate signing request (CSR), and request a certificate that will be used as its' identity.
	//   examples:
	//     - name: example token
	//       value: "\"328hom.uqjzh6jnn2eie9oi\""
	MachineToken string `yaml:"token"` // Warning: It is important to ensure that this token is correct since a machine's certificate has a short TTL by default.
	//   description: |
	//     The root certificate authority of the PKI.
	//     It is composed of a base64 encoded `crt` and `key`.
	//   examples:
	//     - value: pemEncodedCertificateExample
	//       name: machine CA example
	MachineCA *x509.PEMEncodedCertificateAndKey `yaml:"ca,omitempty"`
	//   description: |
	//     Extra certificate subject alternative names for the machine's certificate.
	//     By default, all non-loopback interface IPs are automatically added to the certificate's SANs.
	//   examples:
	//     - name: Uncomment this to enable SANs.
	//       value: '[]string{"10.0.0.10", "172.16.0.10", "192.168.0.10"}'
	MachineCertSANs []string `yaml:"certSANs"`
	//   description: |
	//     Used to provide additional options to the kubelet.
	//   examples:
	//     - name: Kubelet definition example.
	//       value: machineKubeletExample
	MachineKubelet *KubeletConfig `yaml:"kubelet,omitempty"`
	//   description: |
	//     Provides machine specific network configuration options.
	//   examples:
	//     - name: Network definition example.
	//       value: machineNetworkConfigExample
	MachineNetwork *NetworkConfig `yaml:"network,omitempty"`
	//   description: |
	//     Used to partition, format and mount additional disks.
	//     Since the rootfs is read only with the exception of `/var`, mounts are only valid if they are under `/var`.
	//     Note that the partitioning and formating is done only once, if and only if no existing partitions are found.
	//     If `size:` is omitted, the partition is sized to occupy the full disk.
	//   examples:
	//     - name: MachineDisks list example.
	//       value: machineDisksExample
	MachineDisks []*MachineDisk `yaml:"disks,omitempty"` // Note: `size` is in units of bytes.
	//   description: |
	//     Used to provide instructions for installations.
	//   examples:
	//     - name: MachineInstall config usage example.
	//       value: machineInstallExample
	MachineInstall *InstallConfig `yaml:"install,omitempty"`
	//   description: |
	//     Allows the addition of user specified files.
	//     The value of `op` can be `create`, `overwrite`, or `append`.
	//     In the case of `create`, `path` must not exist.
	//     In the case of `overwrite`, and `append`, `path` must be a valid file.
	//     If an `op` value of `append` is used, the existing file will be appended.
	//     Note that the file contents are not required to be base64 encoded.
	//   examples:
	//      - name: MachineFiles usage example.
	//        value: machineFilesExample
	MachineFiles []*MachineFile `yaml:"files,omitempty"` // Note: The specified `path` is relative to `/var`.
	//   description: |
	//     The `env` field allows for the addition of environment variables.
	//     All environment variables are set on PID 1 in addition to every service.
	//   values:
	//     - "`GRPC_GO_LOG_VERBOSITY_LEVEL`"
	//     - "`GRPC_GO_LOG_SEVERITY_LEVEL`"
	//     - "`http_proxy`"
	//     - "`https_proxy`"
	//     - "`no_proxy`"
	//   examples:
	//     - name: Environment variables definition examples.
	//       value: machineEnvExamples[0]
	//     - value: machineEnvExamples[1]
	//     - value: machineEnvExamples[2]
	MachineEnv Env `yaml:"env,omitempty"`
	//   description: |
	//     Used to configure the machine's time settings.
	//   examples:
	//     - name: Example configuration for cloudflare ntp server.
	//       value: machineTimeExample
	MachineTime *TimeConfig `yaml:"time,omitempty"`
	//   description: |
	//     Used to configure the machine's sysctls.
	//   examples:
	//     - name: MachineSysctls usage example.
	//       value: machineSysctlsExample
	MachineSysctls map[string]string `yaml:"sysctls,omitempty"`
	//   description: |
	//     Used to configure the machine's container image registry mirrors.
	//
	//     Automatically generates matching CRI configuration for registry mirrors.
	//
	//     The `mirrors` section allows to redirect requests for images to non-default registry,
	//     which might be local registry or caching mirror.
	//
	//     The `config` section provides a way to authenticate to the registry with TLS client
	//     identity, provide registry CA, or authentication information.
	//     Authentication information has same meaning with the corresponding field in `.docker/config.json`.
	//
	//     See also matching configuration for [CRI containerd plugin](https://github.com/containerd/cri/blob/master/docs/registry.md).
	//   examples:
	//     - value: machineConfigRegistriesExample
	MachineRegistries RegistriesConfig `yaml:"registries,omitempty"`
}

// ClusterConfig represents the cluster-wide config values.
//
//  examples:
//     - value: clusterConfigExample
type ClusterConfig struct {
	//   description: |
	//     Provides control plane specific configuration options.
	//   examples:
	//     - name: Setting controlplane endpoint address to 1.2.3.4 and port to 443 example.
	//       value: clusterControlPlaneExample
	ControlPlane *ControlPlaneConfig `yaml:"controlPlane"`
	//   description: |
	//     Configures the cluster's name.
	ClusterName string `yaml:"clusterName,omitempty"`
	//   description: |
	//     Provides cluster specific network configuration options.
	//   examples:
	//     - name: Configuring with flannel CNI and setting up subnets.
	//       value:  clusterNetworkExample
	ClusterNetwork *ClusterNetworkConfig `yaml:"network,omitempty"`
	//   description: |
	//     The [bootstrap token](https://kubernetes.io/docs/reference/access-authn-authz/bootstrap-tokens/) used to join the cluster.
	//   examples:
	//     - name: Bootstrap token example (do not use in production!).
	//       value: '"wlzjyw.bei2zfylhs2by0wd"'
	BootstrapToken string `yaml:"token,omitempty"`
	//   description: |
	//     The key used for the [encryption of secret data at rest](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/).
	//   examples:
	//     - name: Decryption secret example (do not use in production!).
	//       value: '"z01mye6j16bspJYtTB/5SFX8j7Ph4JXxM2Xuu4vsBPM="'
	ClusterAESCBCEncryptionSecret string `yaml:"aescbcEncryptionSecret"`
	//   description: |
	//     The base64 encoded root certificate authority used by Kubernetes.
	//   examples:
	//     - name: ClusterCA example.
	//       value: pemEncodedCertificateExample
	ClusterCA *x509.PEMEncodedCertificateAndKey `yaml:"ca,omitempty"`
	//   description: |
	//     API server specific configuration options.
	//   examples:
	//     - value: clusterAPIServerExample
	APIServerConfig *APIServerConfig `yaml:"apiServer,omitempty"`
	//   description: |
	//     Controller manager server specific configuration options.
	//   examples:
	//     - value: clusterControllerManagerExample
	ControllerManagerConfig *ControllerManagerConfig `yaml:"controllerManager,omitempty"`
	//   description: |
	//     Kube-proxy server-specific configuration options
	//   examples:
	//     - value: clusterProxyExample
	ProxyConfig *ProxyConfig `yaml:"proxy,omitempty"`
	//   description: |
	//     Scheduler server specific configuration options.
	//   examples:
	//     - value: clusterSchedulerConfig
	SchedulerConfig *SchedulerConfig `yaml:"scheduler,omitempty"`
	//   description: |
	//     Etcd specific configuration options.
	//   examples:
	//     - value: clusterEtcdConfig
	EtcdConfig *EtcdConfig `yaml:"etcd,omitempty"`
	//   description: |
	//     Pod Checkpointer specific configuration options.
	//   examples:
	//     - value: clusterPodCheckpointerExample
	PodCheckpointerConfig *PodCheckpointer `yaml:"podCheckpointer,omitempty"`
	//   description: |
	//     Core DNS specific configuration options.
	//   examples:
	//     - value: clusterCoreDNSExample
	CoreDNSConfig *CoreDNS `yaml:"coreDNS,omitempty"`
	//   description: |
	//     A list of urls that point to additional manifests.
	//     These will get automatically deployed by bootkube.
	//   examples:
	//     - value: >
	//        []string{
	//         "https://www.example.com/manifest1.yaml",
	//         "https://www.example.com/manifest2.yaml",
	//        }
	ExtraManifests []string `yaml:"extraManifests,omitempty"`
	//   description: |
	//     A map of key value pairs that will be added while fetching the ExtraManifests.
	//   examples:
	//     - value: >
	//         map[string]string{
	//           "Token": "1234567",
	//           "X-ExtraInfo": "info",
	//         }
	ExtraManifestHeaders map[string]string `yaml:"extraManifestHeaders,omitempty"`
	//   description: |
	//     Settings for admin kubeconfig generation.
	//     Certificate lifetime can be configured.
	//   examples:
	//     - value: clusterAdminKubeconfigExample
	AdminKubeconfigConfig AdminKubeconfigConfig `yaml:"adminKubeconfig,omitempty"`
	//   description: |
	//     Indicates if master nodes are schedulable.
	//   values:
	//     - true
	//     - yes
	//     - false
	//     - no
	AllowSchedulingOnMasters bool `yaml:"allowSchedulingOnMasters,omitempty"`
}

// KubeletConfig represents the kubelet config values.
type KubeletConfig struct {
	//   description: |
	//     The `image` field is an optional reference to an alternative kubelet image.
	//   examples:
	//     - value: kubeletImageExample
	KubeletImage string `yaml:"image,omitempty"`
	//   description: |
	//     The `extraArgs` field is used to provide additional flags to the kubelet.
	//   examples:
	//     - value: >
	//         map[string]string{
	//           "key": "value",
	//         }
	KubeletExtraArgs map[string]string `yaml:"extraArgs,omitempty"`
	//   description: |
	//     The `extraMounts` field is used to add additional mounts to the kubelet container.
	//   examples:
	//     - value: kubeletExtraMountsExample
	KubeletExtraMounts []specs.Mount `yaml:"extraMounts,omitempty"`
}

// NetworkConfig represents the machine's networking config values.
type NetworkConfig struct {
	//   description: |
	//     Used to statically set the hostname for the machine.
	NetworkHostname string `yaml:"hostname,omitempty"`
	//   description: |
	//     `interfaces` is used to define the network interface configuration.
	//     By default all network interfaces will attempt a DHCP discovery.
	//     This can be further tuned through this configuration parameter.
	//   examples:
	//     - value: machineNetworkConfigExample.NetworkInterfaces
	NetworkInterfaces []*Device `yaml:"interfaces,omitempty"`
	//   description: |
	//     Used to statically set the nameservers for the machine.
	//     Defaults to `1.1.1.1` and `8.8.8.8`
	//   examples:
	//     - value: '[]string{"8.8.8.8", "1.1.1.1"}'
	NameServers []string `yaml:"nameservers,omitempty"`
	//   description: |
	//     Allows for extra entries to be added to the `/etc/hosts` file
	//   examples:
	//     - value: networkConfigExtraHostsExample
	ExtraHostEntries []*ExtraHost `yaml:"extraHostEntries,omitempty"`
}

// InstallConfig represents the installation options for preparing a node.
type InstallConfig struct {
	//   description: |
	//     The disk used for installations.
	//   examples:
	//     - value: '"/dev/sda"'
	//     - value: '"/dev/nvme0"'
	InstallDisk string `yaml:"disk,omitempty"`
	//   description: |
	//     Allows for supplying extra kernel args via the bootloader.
	//   examples:
	//     - value: '[]string{"talos.platform=metal", "reboot=k"}'
	InstallExtraKernelArgs []string `yaml:"extraKernelArgs,omitempty"`
	//   description: |
	//     Allows for supplying the image used to perform the installation.
	//     Image reference for each Talos release can be found on
	//     [GitHub releases page](https://github.com/talos-systems/talos/releases).
	//   examples:
	//     - value: '"ghcr.io/talos-systems/installer:latest"'
	InstallImage string `yaml:"image,omitempty"`
	//   description: |
	//     Indicates if a bootloader should be installed.
	//   values:
	//     - true
	//     - yes
	//     - false
	//     - no
	InstallBootloader bool `yaml:"bootloader,omitempty"`
	//   description: |
	//     Indicates if the installation disk should be wiped at installation time.
	//     Defaults to `true`.
	//   values:
	//     - true
	//     - yes
	//     - false
	//     - no
	InstallWipe bool `yaml:"wipe"`
}

// TimeConfig represents the options for configuring time on a machine.
type TimeConfig struct {
	//   description: |
	//     Indicates if the time service is disabled for the machine.
	//     Defaults to `false`.
	TimeDisabled bool `yaml:"disabled"`
	//   description: |
	//     Specifies time (NTP) servers to use for setting the system time.
	//     Defaults to `pool.ntp.org`
	TimeServers []string `yaml:"servers,omitempty"` // This parameter only supports a single time server.
}

// RegistriesConfig represents the image pull options.
type RegistriesConfig struct {
	//   description: |
	//     Specifies mirror configuration for each registry.
	//     This setting allows to use local pull-through caching registires,
	//     air-gapped installations, etc.
	//
	//     Registry name is the first segment of image identifier, with 'docker.io'
	//     being default one.
	//     To catch any registry names not specified explicitly, use '*'.
	//   examples:
	//     - value: machineConfigRegistryMirrorsExample
	RegistryMirrors map[string]*RegistryMirrorConfig `yaml:"mirrors,omitempty"`
	//   description: |
	//     Specifies TLS & auth configuration for HTTPS image registries.
	//     Mutual TLS can be enabled with 'clientIdentity' option.
	//
	//     TLS configuration can be skipped if registry has trusted
	//     server certificate.
	//   examples:
	//     - value: machineConfigRegistryConfigExample
	RegistryConfig map[string]*RegistryConfig `yaml:"config,omitempty"`
}

// PodCheckpointer represents the pod-checkpointer config values.
type PodCheckpointer struct {
	//   description: |
	//     The `image` field is an override to the default pod-checkpointer image.
	PodCheckpointerImage string `yaml:"image,omitempty"`
}

// CoreDNS represents the CoreDNS config values.
type CoreDNS struct {
	//   description: |
	//     The `image` field is an override to the default coredns image.
	CoreDNSImage string `yaml:"image,omitempty"`
}

// Endpoint represents the endpoint URL parsed out of the machine config.
type Endpoint struct {
	*url.URL
}

// UnmarshalYAML is a custom unmarshaller for `Endpoint`.
func (e *Endpoint) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var endpoint string

	if err := unmarshal(&endpoint); err != nil {
		return err
	}

	url, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	*e = Endpoint{url}

	return nil
}

// MarshalYAML is a custom unmarshaller for `Endpoint`.
func (e *Endpoint) MarshalYAML() (interface{}, error) {
	return e.URL.String(), nil
}

// ControlPlaneConfig represents the control plane configuration options.
type ControlPlaneConfig struct {
	//   description: |
	//     Endpoint is the canonical controlplane endpoint, which can be an IP address or a DNS hostname.
	//     It is single-valued, and may optionally include a port number.
	//   examples:
	//     - value: '"https://1.2.3.4:6443"'
	//     - value: '"https://cluster1.internal:6443"'
	Endpoint *Endpoint `yaml:"endpoint"`
	//   description: |
	//     The port that the API server listens on internally.
	//     This may be different than the port portion listed in the endpoint field above.
	//     The default is `6443`.
	LocalAPIServerPort int `yaml:"localAPIServerPort,omitempty"`
}

// APIServerConfig represents the kube apiserver configuration options.
type APIServerConfig struct {
	//   description: |
	//     The container image used in the API server manifest.
	ContainerImage string `yaml:"image,omitempty"`
	//   description: |
	//     Extra arguments to supply to the API server.
	ExtraArgsConfig map[string]string `yaml:"extraArgs,omitempty"`
	//   description: |
	//     Extra certificate subject alternative names for the API server's certificate.
	CertSANs []string `yaml:"certSANs,omitempty"`
}

// ControllerManagerConfig represents the kube controller manager configuration options.
type ControllerManagerConfig struct {
	//   description: |
	//     The container image used in the controller manager manifest.
	ContainerImage string `yaml:"image,omitempty"`
	//   description: |
	//     Extra arguments to supply to the controller manager.
	ExtraArgsConfig map[string]string `yaml:"extraArgs,omitempty"`
}

// ProxyConfig represents the kube proxy configuration options.
type ProxyConfig struct {
	//   description: |
	//     The container image used in the kube-proxy manifest.
	ContainerImage string `yaml:"image,omitempty"`
	//   description: |
	//     proxy mode of kube-proxy.
	//     The default is 'iptables'.
	ModeConfig string `yaml:"mode,omitempty"`
	//   description: |
	//     Extra arguments to supply to kube-proxy.
	ExtraArgsConfig map[string]string `yaml:"extraArgs,omitempty"`
}

// SchedulerConfig represents the kube scheduler configuration options.
type SchedulerConfig struct {
	//   description: |
	//     The container image used in the scheduler manifest.
	ContainerImage string `yaml:"image,omitempty"`
	//   description: |
	//     Extra arguments to supply to the scheduler.
	ExtraArgsConfig map[string]string `yaml:"extraArgs,omitempty"`
}

// EtcdConfig represents the etcd configuration options.
type EtcdConfig struct {
	//   description: |
	//     The container image used to create the etcd service.
	ContainerImage string `yaml:"image,omitempty"`
	//   description: |
	//     The `ca` is the root certificate authority of the PKI.
	//     It is composed of a base64 encoded `crt` and `key`.
	//   examples:
	//     - value: pemEncodedCertificateExample
	RootCA *x509.PEMEncodedCertificateAndKey `yaml:"ca"`
	//   description: |
	//     Extra arguments to supply to etcd.
	//     Note that the following args are not allowed:
	//
	//     - `name`
	//     - `data-dir`
	//     - `initial-cluster-state`
	//     - `listen-peer-urls`
	//     - `listen-client-urls`
	//     - `cert-file`
	//     - `key-file`
	//     - `trusted-ca-file`
	//     - `peer-client-cert-auth`
	//     - `peer-cert-file`
	//     - `peer-trusted-ca-file`
	//     - `peer-key-file`
	//   examples:
	//     - values: >
	//         map[string]string{
	//           "initial-cluster": "https://1.2.3.4:2380",
	//           "advertise-client-urls": "https://1.2.3.4:2379",
	//         }
	EtcdExtraArgs map[string]string `yaml:"extraArgs,omitempty"`
}

// ClusterNetworkConfig represents kube networking configuration options.
type ClusterNetworkConfig struct {
	//   description: |
	//     The CNI used.
	//     Composed of "name" and "url".
	//     The "name" key only supports options of "flannel" or "custom".
	//     URLs is only used if name is equal to "custom".
	//     URLs should point to the set of YAML files to be deployed.
	//     An empty struct or any other name will default to bootkube's flannel.
	//   examples:
	//     - value: clusterCustomCNIExample
	CNI *CNIConfig `yaml:"cni,omitempty"`
	//   description: |
	//     The domain used by Kubernetes DNS.
	//     The default is `cluster.local`
	//   examples:
	//     - value: '"cluser.local"'
	DNSDomain string `yaml:"dnsDomain"`
	//   description: |
	//     The pod subnet CIDR.
	//   examples:
	//     -  value: >
	//          []string{"10.244.0.0/16"}
	PodSubnet []string `yaml:"podSubnets"`
	//   description: |
	//     The service subnet CIDR.
	//   examples:
	//   examples:
	//     -  value: >
	//          []string{"10.96.0.0/12"}
	ServiceSubnet []string `yaml:"serviceSubnets"`
}

// CNIConfig represents the CNI configuration options.
type CNIConfig struct {
	//   description: |
	//     Name of CNI to use.
	CNIName string `yaml:"name"`
	//   description: |
	//     URLs containing manifests to apply for the CNI.
	CNIUrls []string `yaml:"urls,omitempty"`
}

// AdminKubeconfigConfig contains admin kubeconfig settings.
type AdminKubeconfigConfig struct {
	//   description: |
	//     Admin kubeconfig certificate lifetime (default is 1 year).
	//     Field format accepts any Go time.Duration format ('1h' for one hour, '10m' for ten minutes).
	AdminKubeconfigCertLifetime time.Duration `yaml:"certLifetime,omitempty"`
}

// MachineDisk represents the options available for partitioning, formatting, and
// mounting extra disks.
type MachineDisk struct {
	//   description: The name of the disk to use.
	DeviceName string `yaml:"device,omitempty"`
	//   description: A list of partitions to create on the disk.
	DiskPartitions []*DiskPartition `yaml:"partitions,omitempty"`
}

// DiskSize partition size in bytes.
type DiskSize uint64

// MarshalYAML write as human readable string.
func (ds DiskSize) MarshalYAML() (interface{}, error) {
	if ds%DiskSize(1000) == 0 {
		bytesString := humanize.Bytes(uint64(ds))
		// ensure that stringifying bytes as human readable string
		// doesn't lose precision
		parsed, err := humanize.ParseBytes(bytesString)
		if err == nil && parsed == uint64(ds) {
			return bytesString, nil
		}
	}

	return uint64(ds), nil
}

// UnmarshalYAML read from human readable string.
func (ds *DiskSize) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var size string

	if err := unmarshal(&size); err != nil {
		return err
	}

	s, err := humanize.ParseBytes(size)
	if err != nil {
		return err
	}

	*ds = DiskSize(s)

	return nil
}

// DiskPartition represents the options for a disk partition.
type DiskPartition struct {
	//   description: |
	//     This size of partition: either bytes or human readable representation.
	//   examples:
	//     - name: Human readable representation.
	//       value: DiskSize(100000000)
	//     - name: Precise value in bytes.
	//       value: 1024 * 1024 * 1024
	DiskSize DiskSize `yaml:"size,omitempty"`
	//   description:
	//     Where to mount the partition.
	DiskMountPoint string `yaml:"mountpoint,omitempty"`
}

// Env represents a set of environment variables.
type Env = map[string]string

// FileMode represents file's permissions.
type FileMode os.FileMode

// String convert file mode to octal string.
func (fm FileMode) String() string {
	return "0o" + strconv.FormatUint(uint64(fm), 8)
}

// MarshalYAML encodes as an octal value.
func (fm FileMode) MarshalYAML() (interface{}, error) {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!int",
		Value: fm.String(),
	}, nil
}

// MachineFile represents a file to write to disk.
type MachineFile struct {
	//   description: The contents of the file.
	FileContent string `yaml:"content"`
	//   description: The file's permissions in octal.
	FilePermissions FileMode `yaml:"permissions"`
	//   description: The path of the file.
	FilePath string `yaml:"path"`
	//   description: The operation to use
	//   values:
	//     - create
	//     - append
	//     - overwrite
	FileOp string `yaml:"op"`
}

// ExtraHost represents a host entry in /etc/hosts.
type ExtraHost struct {
	//   description: The IP of the host.
	HostIP string `yaml:"ip"`
	//   description: The host alias.
	HostAliases []string `yaml:"aliases"`
}

// Device represents a network interface.
type Device struct {
	//   description: The interface name.
	//   examples:
	//     - value: '"eth0"'
	DeviceInterface string `yaml:"interface"`
	//   description: |
	//     Assigns a static IP address to the interface.
	//     This should be in proper CIDR notation.
	//
	//     > Note: This option is mutually exclusive with DHCP option.
	//   examples:
	//     - value: '"10.5.0.0/16"'
	DeviceCIDR string `yaml:"cidr,omitempty"`
	//   description: |
	//     A list of routes associated with the interface.
	//     If used in combination with DHCP, these routes will be appended to routes returned by DHCP server.
	//   examples:
	//     - value: networkConfigRoutesExample
	DeviceRoutes []*Route `yaml:"routes,omitempty"`
	//   description: Bond specific options.
	//   examples:
	//     - value: networkConfigBondExample
	DeviceBond *Bond `yaml:"bond,omitempty"`
	//   description: VLAN specific options.
	DeviceVlans []*Vlan `yaml:"vlans,omitempty"`
	//   description: |
	//     The interface's MTU.
	//     If used in combination with DHCP, this will override any MTU settings returned from DHCP server.
	DeviceMTU int `yaml:"mtu"`
	//   description: |
	//     Indicates if DHCP should be used to configure the interface.
	//     The following DHCP options are supported:
	//
	//     - `OptionClasslessStaticRoute`
	//     - `OptionDomainNameServer`
	//     - `OptionDNSDomainSearchList`
	//     - `OptionHostName`
	//
	//     > Note: This option is mutually exclusive with CIDR.
	//     >
	//     > Note: To configure an interface with *only* IPv6 SLAAC addressing, CIDR should be set to "" and DHCP to false
	//     > in order for Talos to skip configuration of addresses.
	//     > All other options will still apply.
	//   examples:
	//     - value: true
	DeviceDHCP bool `yaml:"dhcp,omitempty"`
	//   description: Indicates if the interface should be ignored (skips configuration).
	DeviceIgnore bool `yaml:"ignore,omitempty"`
	//   description: |
	//     Indicates if the interface is a dummy interface.
	//     `dummy` is used to specify that this interface should be a virtual-only, dummy interface.
	DeviceDummy bool `yaml:"dummy,omitempty"`
	//   description: |
	//     DHCP specific options.
	//     `dhcp` *must* be set to true for these to take effect.
	//   examples:
	//     - value: networkConfigDHCPOptionsExample
	DeviceDHCPOptions *DHCPOptions `yaml:"dhcpOptions,omitempty"`
}

// DHCPOptions contains options for configuring the DHCP settings for a given interface.
type DHCPOptions struct {
	//   description: The priority of all routes received via DHCP.
	DHCPRouteMetric uint32 `yaml:"routeMetric"`
}

// Bond contains the various options for configuring a bonded interface.
type Bond struct {
	//   description: The interfaces that make up the bond.
	BondInterfaces []string `yaml:"interfaces"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondARPIPTarget []string `yaml:"arpIPTarget,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondMode string `yaml:"mode"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondHashPolicy string `yaml:"xmitHashPolicy,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondLACPRate string `yaml:"lacpRate,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondADActorSystem string `yaml:"adActorSystem,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondARPValidate string `yaml:"arpValidate,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondARPAllTargets string `yaml:"arpAllTargets,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondPrimary string `yaml:"primary,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondPrimaryReselect string `yaml:"primaryReselect,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondFailOverMac string `yaml:"failOverMac,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondADSelect string `yaml:"adSelect,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondMIIMon uint32 `yaml:"miimon,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondUpDelay uint32 `yaml:"updelay,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondDownDelay uint32 `yaml:"downdelay,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondARPInterval uint32 `yaml:"arpInterval,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondResendIGMP uint32 `yaml:"resendIgmp,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondMinLinks uint32 `yaml:"minLinks,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondLPInterval uint32 `yaml:"lpInterval,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondPacketsPerSlave uint32 `yaml:"packetsPerSlave,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondNumPeerNotif uint8 `yaml:"numPeerNotif,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondTLBDynamicLB uint8 `yaml:"tlbDynamicLb,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondAllSlavesActive uint8 `yaml:"allSlavesActive,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondUseCarrier bool `yaml:"useCarrier,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondADActorSysPrio uint16 `yaml:"adActorSysPrio,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondADUserPortKey uint16 `yaml:"adUserPortKey,omitempty"`
	//   description: |
	//     A bond option.
	//     Please see the official kernel documentation.
	BondPeerNotifyDelay uint32 `yaml:"peerNotifyDelay,omitempty"`
}

// Vlan represents vlan settings for a device.
type Vlan struct {
	//   description: The CIDR to use.
	VlanCIDR string `yaml:"cidr"`
	//   description: A list of routes associated with the VLAN.
	VlanRoutes []*Route `yaml:"routes"`
	//   description: Indicates if DHCP should be used.
	VlanDHCP bool `yaml:"dhcp"`
	//   description: The VLAN's ID.
	VlanID uint16 `yaml:"vlanId"`
}

// Route represents a network route.
type Route struct {
	//   description: The route's network.
	RouteNetwork string `yaml:"network"`
	//   description: The route's gateway.
	RouteGateway string `yaml:"gateway"`
	//   description: The optional metric for the route.
	RouteMetric uint32 `yaml:"metric,omitempty"`
}

// RegistryMirrorConfig represents mirror configuration for a registry.
type RegistryMirrorConfig struct {
	//   description: |
	//     List of endpoints (URLs) for registry mirrors to use.
	//     Endpoint configures HTTP/HTTPS access mode, host name,
	//     port and path (if path is not set, it defaults to `/v2`).
	MirrorEndpoints []string `yaml:"endpoints"`
}

// RegistryConfig specifies auth & TLS config per registry.
type RegistryConfig struct {
	//   description: |
	//     The TLS configuration for the registry.
	//   examples:
	//     - value: machineConfigRegistryTLSConfigExample1
	//     - value: machineConfigRegistryTLSConfigExample2
	RegistryTLS *RegistryTLSConfig `yaml:"tls,omitempty"`
	//   description: The auth configuration for this registry.
	//   examples:
	//     - value: machineConfigRegistryAuthConfigExample
	RegistryAuth *RegistryAuthConfig `yaml:"auth,omitempty"`
}

// RegistryAuthConfig specifies authentication configuration for a registry.
type RegistryAuthConfig struct {
	//   description: |
	//     Optional registry authentication.
	//     The meaning of each field is the same with the corresponding field in .docker/config.json.
	RegistryUsername string `yaml:"username,omitempty"`
	//   description: |
	//     Optional registry authentication.
	//     The meaning of each field is the same with the corresponding field in .docker/config.json.
	RegistryPassword string `yaml:"password,omitempty"`
	//   description: |
	//     Optional registry authentication.
	//     The meaning of each field is the same with the corresponding field in .docker/config.json.
	RegistryAuth string `yaml:"auth,omitempty"`
	//   description: |
	//     Optional registry authentication.
	//     The meaning of each field is the same with the corresponding field in .docker/config.json.
	RegistryIdentityToken string `yaml:"identityToken,omitempty"`
}

// RegistryTLSConfig specifies TLS config for HTTPS registries.
type RegistryTLSConfig struct {
	//   description: |
	//     Enable mutual TLS authentication with the registry.
	//     Client certificate and key should be base64-encoded.
	//   examples:
	//     - value: pemEncodedCertificateExample
	TLSClientIdentity *x509.PEMEncodedCertificateAndKey `yaml:"clientIdentity,omitempty"`
	//   description: |
	//     CA registry certificate to add the list of trusted certificates.
	//     Certificate should be base64-encoded.
	TLSCA Base64Bytes `yaml:"ca,omitempty"`
	//   description: |
	//     Skip TLS server certificate verification (not recommended).
	TLSInsecureSkipVerify bool `yaml:"insecureSkipVerify,omitempty"`
}
