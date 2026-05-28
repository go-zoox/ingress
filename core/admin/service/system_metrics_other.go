//go:build !unix

package service

import "runtime"

func processCPUTimeSeconds() float64 {
	return 0
}

func processMemoryMB() float64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return float64(stats.HeapInuse) / (1024 * 1024)
}
