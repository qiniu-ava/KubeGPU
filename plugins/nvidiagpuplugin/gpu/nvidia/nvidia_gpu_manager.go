package nvidia

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	devtypes "github.com/Microsoft/KubeGPU/crishim/pkg/types"
	gputypes "github.com/Microsoft/KubeGPU/plugins/gpuplugintypes"
	"github.com/Microsoft/KubeGPU/types"
	"github.com/golang/glog"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1alpha"
)

type memoryInfo struct {
	Global int64 `json:"Global"`
}

type pciInfo struct {
	BusID     string `json:"BusID"`
	Bandwidth int64  `json:"Bandwidth"`
}

type topologyInfo struct {
	BusID string `json:"BusID"`
	Link  int32  `json:"Link"`
}

type gpuInfo struct {
	ID       string         `json:"UUID"`
	Model    string         `json:"Model"`
	Path     string         `json:"Path"`
	Memory   memoryInfo     `json:"Memory"`
	PCI      pciInfo        `json:"PCI"`
	Topology []topologyInfo `json:"Topology"`
	Found    bool           `json:"-"`
	Index    int            `json:"-"`
	InUse    bool           `json:"-"`
	TopoDone bool           `json:"-"`
	Name     string         `json:"-"`
}

type gpusInfo struct {
	Gpus []gpuInfo `json:"Devices"`
}

// NvidiaGPUManager manages nvidia gpu devices.
type NvidiaGPUManager struct {
	sync.Mutex
	np        NvidiaPlugin
	gpus      map[string]gpuInfo
	pathToID  map[string]string
	busIDToID map[string]string
	indexToID []string
	numGpus   int
}

// NewNvidiaGPUManager returns a GPUManager that manages local Nvidia GPUs.
// TODO: Migrate to use pod level cgroups and make it generic to all runtimes.
func NewNvidiaGPUManager() (devtypes.Device, error) {
	ngm := &NvidiaGPUManager{}
	return ngm, ngm.New()
}

func (ngm *NvidiaGPUManager) New() error {
	plugin := &NvidiaDockerPlugin{}
	ngm.gpus = make(map[string]gpuInfo)
	ngm.np = plugin
	return nil
}

func arrayContains(arr []int32, val int32) bool {
	for _, elem := range arr {
		if val == elem {
			return true
		}
	}
	return false
}

func (ngm *NvidiaGPUManager) GetName() string {
	return "nvidiagpu"
}

// topology discovery
func (ngm *NvidiaGPUManager) topologyDiscovery(links []int32, level int32) {
	for id, copy := range ngm.gpus {
		copy.TopoDone = false
		ngm.gpus[id] = copy
	}
	linkID := 0
	for _, id := range ngm.indexToID {
		copy := ngm.gpus[id]
		if !ngm.gpus[id].Found || ngm.gpus[id].TopoDone {
			continue
		}
		prefix := "gpugrp" + strconv.Itoa(int(level)) + "/" + strconv.Itoa(int(linkID))
		linkID++
		copy.Name = prefix + "/" + ngm.gpus[id].Name
		copy.TopoDone = true
		ngm.gpus[id] = copy
		for _, topolink := range ngm.gpus[id].Topology {
			if arrayContains(links, topolink.Link) {
				idOnLink := ngm.busIDToID[topolink.BusID]
				gpuOnLink := ngm.gpus[idOnLink]
				if gpuOnLink.Found {
					gpuOnLink.Name = prefix + "/" + gpuOnLink.Name
					gpuOnLink.TopoDone = true
					ngm.gpus[idOnLink] = gpuOnLink
				}
			}
		}
	}
}

// Initialize the GPU devices
func (ngm *NvidiaGPUManager) UpdateGPUInfo() error {
	ngm.Lock()
	defer ngm.Unlock()

	np := ngm.np
	devices, err := np.GetGPUInfo()
	if err != nil {
		return err
	}
	var gpus gpusInfo
	for _, d := range devices {
		info := gpuInfo{
			ID:   d.UUID,
			Path: d.Path,
			PCI:  pciInfo{BusID: d.PCI.BusID},
		}
		if d.Model != nil {
			info.Model = *d.Model
		}
		if d.Memory != nil {
			info.Memory = memoryInfo{Global: int64(*d.Memory)}
		}
		if d.PCI.Bandwidth != nil {
			info.PCI.Bandwidth = int64(*d.PCI.Bandwidth)
		}
		for _, tp := range d.Topology {
			info.Topology = append(info.Topology, topologyInfo{
				BusID: tp.BusID,
				Link:  int32(tp.Link),
			})
		}
		gpus.Gpus = append(gpus.Gpus, info)
	}

	// convert certain resources to correct units, such as memory and Bandwidth
	for i := range gpus.Gpus {
		gpus.Gpus[i].Memory.Global *= int64(1024) * int64(1024) // in units of MiB
		gpus.Gpus[i].PCI.Bandwidth *= int64(1000) * int64(1000) // in units of MB
	}

	for key := range ngm.gpus {
		copy := ngm.gpus[key]
		copy.Found = false
		ngm.gpus[key] = copy
	}
	// go over found GPUs and reassign
	ngm.pathToID = make(map[string]string)
	ngm.busIDToID = make(map[string]string)
	ngm.indexToID = make([]string, len(gpus.Gpus))
	for index, gpuFound := range gpus.Gpus {
		gpu, available := ngm.gpus[gpuFound.ID]
		if available {
			gpuFound.InUse = gpu.InUse
		}
		gpuFound.Found = true
		gpuFound.Index = index
		gpuFound.Name = "gpu/" + gpuFound.ID
		ngm.gpus[gpuFound.ID] = gpuFound
		ngm.pathToID[gpuFound.Path] = gpuFound.ID
		ngm.busIDToID[gpuFound.PCI.BusID] = gpuFound.ID
		ngm.indexToID[index] = gpuFound.ID
	}
	// set numGpus to number found -- not to len(ngm.gpus)
	ngm.numGpus = len(gpus.Gpus) // if ngm.numGpus <> len(ngm.gpus), then some gpus have gone missing

	// perform topology discovery to reassign name
	// more information regarding various "link types" can be found in https://github.com/nvidia/nvidia-docker/blob/master/src/nvml/nvml.go
	// const (
	// 	P2PLinkUnknown P2PLinkType = iota
	// 	P2PLinkCrossCPU
	// 	P2PLinkSameCPU
	// 	P2PLinkHostBridge
	// 	P2PLinkMultiSwitch
	// 	P2PLinkSingleSwitch
	// 	P2PLinkSameBoard
	// )
	// For topology levels, see https://docs.nvidia.com/deploy/pdf/NVML_API_Reference_Guide.pdf
	// NVML_TOPOLOGY_INTERNAL = 0 (translate to level 6)
	// NVML_TOPOLOGY_SINGLE = 10 (level 5)
	// NVML_TOPOLOGY_MULTIPLE = 20 (level 4)
	// NVML_TOPOLOGY_HOSTBRIDGE = 30 (level 3)
	// NVML_TOPOLOGY_CPU = 40 (level 2)
	// NVML_TOPOLOGY_SYSTEM = 50 (level 1)
	//
	// can have more levels if desired, but perhaps two levels are sufficient
	// link "5" discovery - put 6, 5, 4 in first group
	ngm.topologyDiscovery([]int32{6, 5, 4}, 0)
	// link "5, 3"" discovery - put all in higher group
	ngm.topologyDiscovery([]int32{6, 5, 4, 3, 2, 1}, 1)

	glog.V(5).Infof("updated gpus: %+v", ngm.gpus)
	return nil
}

func (ngm *NvidiaGPUManager) Start() error {
	_ = ngm.UpdateGPUInfo() // ignore error in updating, gpus stay at zero
	return nil
}

// Get how many GPU cards we have.
func (ngm *NvidiaGPUManager) UpdateNodeInfo(nodeInfo *types.NodeInfo) error {
	err := ngm.UpdateGPUInfo() // don't care about error, ignore it
	if err != nil {
		ngm.numGpus = 0
		return err
	}
	nodeInfo.Capacity[gputypes.NvidiaGPUResource] = int64(len(ngm.gpus))
	nodeInfo.Allocatable[gputypes.NvidiaGPUResource] = int64(len(ngm.gpus))
	for _, val := range ngm.gpus {
		if val.Found { // if currently discovered
			types.AddGroupResource(nodeInfo.Capacity, val.Name+"/memory", val.Memory.Global)
			types.AddGroupResource(nodeInfo.Allocatable, val.Name+"/memory", val.Memory.Global)
			types.AddGroupResource(nodeInfo.Capacity, val.Name+"/cards", int64(1))
			types.AddGroupResource(nodeInfo.Allocatable, val.Name+"/cards", int64(1))
		}
	}
	return nil
}

// AllocateGPU returns VolumeName, VolumeDriver, and list of Devices to use
func (ngm *NvidiaGPUManager) Allocate(pod *types.PodInfo, container *types.ContainerInfo) (*pluginapi.AllocateResponse, error) {
	ngm.Lock()
	defer ngm.Unlock()
	if container.AllocateFrom == nil || 0 == len(container.AllocateFrom) {
		return nil, nil
	}

	re := regexp.MustCompile(types.DeviceGroupPrefix + "/gpugrp1/.*/gpugrp0/.*/gpu/" + `(.*?)/cards`)
	resp := &pluginapi.AllocateResponse{Envs: make(map[string]string)}
	var devIDs []string
	for _, res := range container.AllocateFrom {
		glog.V(4).Infof("PodName: %v -- searching for device UID: %v", pod.Name, res)
		matches := re.FindStringSubmatch(string(res))
		if len(matches) >= 2 {
			id := matches[1]
			gpu, ok := ngm.gpus[id]
			if !ok || !gpu.Found {
				return nil, fmt.Errorf("device %s is not available", id)
			}
			devIDs = append(devIDs, id)
		}
	}
	resp.Envs["NVIDIA_VISIBLE_DEVICES"] = strings.Join(devIDs, ",")
	return resp, nil
}
