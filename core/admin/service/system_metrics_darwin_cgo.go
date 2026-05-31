//go:build darwin && cgo

package service

/*
#include <libproc.h>
#include <unistd.h>

static double ingressCurrentRSSMB(void) {
	struct proc_taskinfo ti;
	if (proc_pidinfo(getpid(), PROC_PIDTASKINFO, 0, &ti, sizeof(ti)) <= (int)0) {
		return -1.0;
	}
	return (double)ti.pti_resident_size / (1024.0 * 1024.0);
}
*/
import "C"

func darwinProcessRSSMB() (float64, bool) {
	v := float64(C.ingressCurrentRSSMB())
	if v < 0 {
		return 0, false
	}
	return v, true
}
