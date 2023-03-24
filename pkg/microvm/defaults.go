package microvm

import (
	"github.com/weaveworks-liquidmetal/flintlock/api/types"
	"k8s.io/utils/pointer"
)

const (
	// KernelImage is the default MVM kernel image.
	KernelImage = "ghcr.io/weaveworks-liquidmetal/flintlock-kernel-arm:5.10.77"
	// CloudImage is the default MVM cloud image.
	CloudImage = "ghcr.io/weaveworks-liquidmetal/capmvm-kubernetes-arm:1.23.5"
)

func defaultMicroVM() *types.MicroVMSpec {
	return &types.MicroVMSpec{
		Vcpu:       2,    //nolint: gomnd // we don't care
		MemoryInMb: 2048, //nolint: gomnd // we don't care
		Kernel: &types.Kernel{
			Image:            KernelImage,
			Filename:         pointer.String("boot/image"),
			AddNetworkConfig: true,
		},
		RootVolume: &types.Volume{
			Id:         "root",
			IsReadOnly: false,
			Source: &types.VolumeSource{
				ContainerSource: pointer.String(CloudImage),
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
