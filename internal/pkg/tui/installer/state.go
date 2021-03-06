// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package installer contains terminal UI based talos interactive installer parts.
package installer

import (
	"context"
	"fmt"

	"github.com/dustin/go-humanize"

	"github.com/talos-systems/talos/pkg/images"
	machineapi "github.com/talos-systems/talos/pkg/machinery/api/machine"
	"github.com/talos-systems/talos/pkg/machinery/config/types/v1alpha1"
	"github.com/talos-systems/talos/pkg/machinery/config/types/v1alpha1/machine"
	"github.com/talos-systems/talos/pkg/machinery/constants"
)

// cniPresets defines custom CNI presets.
var cniPresets = map[string]*machineapi.CNIConfig{
	"cilium": {
		Name: "custom",
		Urls: []string{
			"https://raw.githubusercontent.com/cilium/cilium/v1.8/install/kubernetes/quick-install.yaml",
		},
	},
}

// NewState creates new installer state.
func NewState(ctx context.Context, conn *Connection) (*State, error) {
	opts := &machineapi.GenerateConfigurationRequest{
		ConfigVersion: "v1alpha1",
		MachineConfig: &machineapi.MachineConfig{
			Type:              machineapi.MachineConfig_MachineType(machine.TypeInit),
			NetworkConfig:     &machineapi.NetworkConfig{},
			KubernetesVersion: constants.DefaultKubernetesVersion,
			InstallConfig: &machineapi.InstallConfig{
				InstallImage: images.DefaultInstallerImage,
			},
		},
		ClusterConfig: &machineapi.ClusterConfig{
			Name:         "talos-default",
			ControlPlane: &machineapi.ControlPlaneConfig{},
			ClusterNetwork: &machineapi.ClusterNetworkConfig{
				DnsDomain: "cluster.local",
			},
		},
	}

	if conn.ExpandingCluster() {
		opts.ClusterConfig.ControlPlane.Endpoint = fmt.Sprintf("https://%s:%d", conn.bootstrapEndpoint, constants.DefaultControlPlanePort)
	} else {
		opts.ClusterConfig.ControlPlane.Endpoint = fmt.Sprintf("https://%s:%d", conn.nodeEndpoint, constants.DefaultControlPlanePort)
	}

	installDiskOptions := []interface{}{
		NewTableHeaders("device name", "model name", "size"),
	}

	disks, err := conn.Disks()
	if err != nil {
		return nil, err
	}

	for i, disk := range disks.Disks {
		if i == 0 {
			opts.MachineConfig.InstallConfig.InstallDisk = disk.DeviceName
		}

		installDiskOptions = append(installDiskOptions, disk.DeviceName, disk.Model, humanize.Bytes(disk.Size))
	}

	var machineTypes []interface{}

	if conn.ExpandingCluster() {
		machineTypes = []interface{}{
			"worker", machineapi.MachineConfig_MachineType(machine.TypeJoin),
			"control plane", machineapi.MachineConfig_MachineType(machine.TypeControlPlane),
		}
		opts.MachineConfig.Type = machineapi.MachineConfig_MachineType(machine.TypeControlPlane)
	} else {
		machineTypes = []interface{}{
			"control plane", machineapi.MachineConfig_MachineType(machine.TypeInit),
		}
	}

	state := &State{
		cni:  constants.DefaultCNI,
		conn: conn,
		opts: opts,
	}

	networkConfigItems := []*Item{
		NewItem(
			"hostname",
			v1alpha1.NetworkConfigDoc.Describe("hostname", true),
			&opts.MachineConfig.NetworkConfig.Hostname,
		),
		NewItem(
			"dns domain",
			v1alpha1.ClusterNetworkConfigDoc.Describe("dnsDomain", true),
			&opts.ClusterConfig.ClusterNetwork.DnsDomain,
		),
	}

	if !conn.ExpandingCluster() {
		networkConfigItems = append(networkConfigItems, NewItem(
			"type",
			v1alpha1.ClusterNetworkConfigDoc.Describe("cni", true),
			&state.cni,
			NewTableHeaders("CNI", "description"),
			constants.DefaultCNI, "CNI used by Talos by default",
			"cilium", "Cillium 1.8 installed through quick-install.yaml",
		))
	}

	state.pages = []*Page{
		NewPage("Installer Params",
			NewItem(
				"image",
				v1alpha1.InstallConfigDoc.Describe("image", true),
				&opts.MachineConfig.InstallConfig.InstallImage,
			),
			NewItem(
				"install disk",
				v1alpha1.InstallConfigDoc.Describe("disk", true),
				&opts.MachineConfig.InstallConfig.InstallDisk,
				installDiskOptions...,
			),
		),
		NewPage("Machine Config",
			NewItem(
				"machine type",
				v1alpha1.MachineConfigDoc.Describe("type", true),
				&opts.MachineConfig.Type,
				machineTypes...,
			),
			NewItem(
				"cluster name",
				v1alpha1.ClusterConfigDoc.Describe("clusterName", true),
				&opts.ClusterConfig.Name,
			),
			NewItem(
				"control plane endpoint",
				v1alpha1.ControlPlaneConfigDoc.Describe("endpoint", true),
				&opts.ClusterConfig.ControlPlane.Endpoint,
			),
			NewItem(
				"kubernetes version",
				"Kubernetes version to install.",
				&opts.MachineConfig.KubernetesVersion,
			),
		),
		NewPage("Network Config",
			networkConfigItems...,
		),
	}

	return state, nil
}

// State installer state.
type State struct {
	pages []*Page
	opts  *machineapi.GenerateConfigurationRequest
	conn  *Connection
	cni   string
}

// GenConfig returns current config encoded in yaml.
func (s *State) GenConfig() (*machineapi.GenerateConfigurationResponse, error) {
	// configure custom cni from the preset
	if customCNI, ok := cniPresets[s.cni]; ok {
		s.opts.ClusterConfig.ClusterNetwork.CniConfig = customCNI
	}

	return s.conn.GenerateConfiguration(s.opts)
}
