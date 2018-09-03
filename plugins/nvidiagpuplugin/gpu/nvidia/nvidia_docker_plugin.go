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
		d, err := nvml.NewDeviceLite(i)
		if err != nil {
			return nil, err
		}
		devs = append(devs, d)
	}

	return devs, nil
}
