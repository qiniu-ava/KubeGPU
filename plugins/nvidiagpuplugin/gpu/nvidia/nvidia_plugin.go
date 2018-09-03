package nvidia

import (
	"github.com/NVIDIA/gpu-monitoring-tools/bindings/go/nvml"
)

type NvidiaPlugin interface {
	GetGPUInfo() ([]*nvml.Device, error)
}
