//go:build unix

package service

import (
	"bufio"
	"bytes"
	"os"
	"runtime"
	"strconv"
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
	if runtime.GOOS == "linux" {
		if mb, ok := linuxProcessRSSMB(); ok {
			return mb
		}
	}
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return heapInuseMB()
	}
	if runtime.GOOS == "darwin" {
		return float64(ru.Maxrss) / (1024 * 1024)
	}
	return heapInuseMB()
}

// linuxProcessRSSMB reads current VmRSS from /proc/self/status (kilobytes).
// getrusage(2) ru_maxrss on Linux is peak RSS since start, not current usage.
func linuxProcessRSSMB() (float64, bool) {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0, false
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Bytes()
		if !bytes.HasPrefix(line, []byte("VmRSS:")) {
			continue
		}
		fields := bytes.Fields(line)
		if len(fields) < 2 {
			return 0, false
		}
		kb, err := strconv.ParseFloat(string(fields[1]), 64)
		if err != nil {
			return 0, false
		}
		return kb / 1024, true
	}
	return 0, false
}

func heapInuseMB() float64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return float64(stats.HeapInuse) / (1024 * 1024)
}
