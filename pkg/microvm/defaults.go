package microvm

import (
	"github.com/weaveworks-liquidmetal/flintlock/api/types"
	"k8s.io/utils/pointer"
)

const (
	// KernelImage is the default MVM kernel image.
	KernelImage = "ghcr.io/weaveworks-liquidmetal/firecracker-kernel-bin-arm:5.10.77"
	// ModulesImage is the default MVM kernel image.
	ModulesImage = "ghcr.io/weaveworks-liquidmetal/firecracker-kernel-modules:5.10.77"
	// OSImage is the default MVM OS image.
	OSImage = "ghcr.io/weaveworks-liquidmetal/action-runner-arm:2.303.0"

	kernelFilename = "boot/image"
	modulesPath    = "/lib/modules/5.10.77"
)

func defaultMicroVM() *types.MicroVMSpec {
	return &types.MicroVMSpec{
		Vcpu:       2,    //nolint: gomnd // we don't care
		MemoryInMb: 2048, //nolint: gomnd // we don't care
		Kernel: &types.Kernel{
			Image:            KernelImage,
			Filename:         pointer.String(kernelFilename),
			AddNetworkConfig: true,
		},
		RootVolume: &types.Volume{
			Id:         "root",
			IsReadOnly: false,
			Source: &types.VolumeSource{
				ContainerSource: pointer.String(OSImage),
			},
		},
		AdditionalVolumes: []*types.Volume{
			{
				Id:         "modules",
				IsReadOnly: false,
				Source: &types.VolumeSource{
					ContainerSource: pointer.String(ModulesImage),
				},
				MountPoint: pointer.String(modulesPath),
			},
		},
		Interfaces: []*types.NetworkInterface{
			{
				DeviceId: "eth1",
				Type:     0,
			},
		},
	}
}
