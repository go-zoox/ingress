//go:build unix && !darwin

package service

func darwinProcessRSSMB() (float64, bool) {
	return 0, false
}
