package main

import (
	"log"

	"github.com/Microsoft/KubeGPU/crishim/pkg/app"
	"github.com/mindprince/gonvml"
)

func main() {
	// Loads the same "libnvidia-ml.so.1" shared library as
	// github.com/NVIDIA/gpu-monitoring-tools/bindings/go/nvml used in nvidiagpuplugin.
	// NOTE:
	// 1. cannot load inside plugin as plugin.Open resolve symbols with no lazy.
	// 2. cannot use go/nvml of nvidia here because duplicate sysmbols conflict.
	// 3. update vendor/github.com/mindprince/gonvml/bindings.go:120 to add RTLD_GLOBAL for sysmbols.
	log.Println("Loading NVML")
	if err := gonvml.Initialize(); err != nil {
		log.Fatalf("Failed to initialize NVML: %s.", err)
	}
	defer func() { log.Println("Shutdown of NVML returned:", gonvml.Shutdown()) }()

	// Add devices here
	// if err := device.DeviceManager.CreateAndAddDevice("nvidiagpu"); err != nil {
	// 	app.Die(fmt.Errorf("Adding device nvidiagpu fails with error %v", err))
	// }
	// run the app - parses all command line arguments
	app.RunApp()
}
