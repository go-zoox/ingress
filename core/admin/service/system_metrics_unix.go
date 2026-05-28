//go:build unix

package service

import (
	"runtime"
	"syscall"
)

func processCPUTimeSeconds() float64 {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0
	}
	user := float64(ru.Utime.Sec) + float64(ru.Utime.Usec)/1e6
	sys := float64(ru.Stime.Sec) + float64(ru.Stime.Usec)/1e6
	return user + sys
}

func processMemoryMB() float64 {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return heapInuseMB()
	}
	switch runtime.GOOS {
	case "darwin":
		return float64(ru.Maxrss) / (1024 * 1024)
	default:
		return float64(ru.Maxrss) / 1024
	}
}

func heapInuseMB() float64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return float64(stats.HeapInuse) / (1024 * 1024)
}
