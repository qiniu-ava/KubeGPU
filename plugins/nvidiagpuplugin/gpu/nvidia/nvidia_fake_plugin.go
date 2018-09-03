package nvidia

import (
	"github.com/Microsoft/KubeGPU/crishim/pkg/types"
	"github.com/NVIDIA/gpu-monitoring-tools/bindings/go/nvml"
)

type NvidiaFakePlugin struct {
	gInfo gpusInfo
}

func (np *NvidiaFakePlugin) GetGPUInfo() ([]*nvml.Device, error) {
	var devs []*nvml.Device
	for _, g := range np.gInfo.Gpus {
		mem := uint64(g.Memory.Global)
		bw := uint(g.PCI.Bandwidth)
		d := &nvml.Device{
			UUID:   g.ID,
			Path:   g.Path,
			Model:  &g.Model,
			Memory: &mem,
			PCI: nvml.PCIInfo{
				BusID:     g.PCI.BusID,
				Bandwidth: &bw,
			},
		}
		for _, tp := range g.Topology {
			d.Topology = append(d.Topology, nvml.P2PLink{
				BusID: tp.BusID,
				Link:  nvml.P2PLinkType(tp.Link),
			})
		}
		devs = append(devs, d)
	}
	return devs, nil
}

func NewFakeNvidiaGPUManager(info *gpusInfo) (types.Device, error) {
	plugin := &NvidiaFakePlugin{
		gInfo: *info,
	}
	return &NvidiaGPUManager{
		gpus: make(map[string]gpuInfo),
		np:   plugin,
	}, nil
}
