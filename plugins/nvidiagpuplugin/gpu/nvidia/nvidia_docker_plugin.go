package nvidia

import (
	"github.com/NVIDIA/gpu-monitoring-tools/bindings/go/nvml"
)

type NvidiaDockerPlugin struct {
}

func (ndp *NvidiaDockerPlugin) GetGPUInfo() ([]*nvml.Device, error) {
	n, err := nvml.GetDeviceCount()
	if err != nil {
		return nil, err
	}

	var devs []*nvml.Device
	for i := uint(0); i < n; i++ {
		d, err := nvml.NewDevice(i)
		if err != nil {
			return nil, err
		}
		devs = append(devs, d)
	}

	// NewDevice won't get topology info, generate by GetP2PLink
	if err := genTopology(devs); err != nil {
		return nil, e
	}
	return devs, nil
}

func genTopology(devs []*nvml.Device) error {
	for i, d1 := range devs {
		for j, d2 := range devs {
			if i != j {
				link, err := nvml.GetP2PLink(d1, d2)
				if err != nil {
					return err
				}
				d1.Topology = append(d1.Topology, nvml.P2PLink{BusID: d2.PCI.BusID, Link: link})
			}
		}
	}
	return nil
}
