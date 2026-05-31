//go:build darwin && !cgo

package service

import "runtime"

// darwinProcessRSSMB approximates current in-use Go memory when cgo/libproc is unavailable.
// ru_maxrss from getrusage is peak RSS since start and overstates vs Activity Monitor.
func darwinProcessRSSMB() (float64, bool) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	used := stats.Alloc + stats.StackInuse + stats.MSpanInuse + stats.MCacheInuse
	if used == 0 {
		return 0, false
	}
	return float64(used) / (1024 * 1024), true
}
