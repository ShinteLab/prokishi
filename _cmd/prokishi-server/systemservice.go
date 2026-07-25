package main

import (
	"context"
	"shinte/prokishi/registry"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/process"
)

type CpuInfo struct {
	Total     float64   `json:"total"`
	PerCPU    []float64 `json:"perCpu"`
	ModelName string    `json:"modelName"`
	Cores     int       `json:"cores"`
}

type ProcessCpuInfo struct {
	ConnID     string  `json:"connId"`
	EngineName string  `json:"engineName"`
	PID        int     `json:"pid"`
	CpuPercent float64 `json:"cpuPercent"`
}

type SystemInfo struct {
	Cpu       *CpuInfo         `json:"cpu"`
	Processes []ProcessCpuInfo `json:"processes"`
}

type SystemService struct {
	registry *registry.Registry
}

func (s *SystemService) GetSystemInfo() (*SystemInfo, error) {
	total, err := cpu.Percent(time.Second, false)
	if err != nil {
		return nil, err
	}
	perCPU, err := cpu.Percent(0, true)
	if err != nil {
		return nil, err
	}

	cpuInfo := &CpuInfo{
		PerCPU: perCPU,
		Cores:  len(perCPU),
	}
	if len(total) > 0 {
		cpuInfo.Total = total[0]
	}
	cpuInfos, err := cpu.Info()
	if err == nil && len(cpuInfos) > 0 {
		cpuInfo.ModelName = cpuInfos[0].ModelName
	}

	var procs []ProcessCpuInfo
	if s.registry != nil {
		conns := s.registry.ListConnections()
		for _, conn := range conns {
			if conn.PID <= 0 {
				continue
			}
			p, err := process.NewProcess(int32(conn.PID))
			if err != nil {
				continue
			}
			raw, err := p.CPUPercentWithContext(context.Background())
			if err != nil {
				continue
			}
			numCPU := runtime.NumCPU()
			pct := raw
			if numCPU > 0 {
				pct = raw / float64(numCPU)
			}
			name := conn.EngineName
			if name == "" {
				name = conn.EnginePath
			}
			procs = append(procs, ProcessCpuInfo{
				ConnID:     conn.ID,
				EngineName: name,
				PID:        conn.PID,
				CpuPercent: pct,
			})
		}
	}

	return &SystemInfo{
		Cpu:       cpuInfo,
		Processes: procs,
	}, nil
}
