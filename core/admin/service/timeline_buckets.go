package service

import "time"

// truncateTime floors t to the nearest slot boundary.
func truncateTime(t time.Time, slot time.Duration) time.Time {
	if slot <= 0 {
		return t
	}
	return t.Truncate(slot)
}

// timelineWindowStart aligns bucket boundaries with filterEntriesInWindow (anchor - window).
func timelineWindowStart(anchor time.Time, window, slot time.Duration) time.Time {
	return truncateTime(anchor.Add(-window), slot)
}

func formatTimelineLabel(bucketStart time.Time, slot time.Duration) string {
	switch {
	case slot < time.Minute:
		return bucketStart.Format("15:04:05")
	case slot < time.Hour:
		return bucketStart.Format("15:04")
	default:
		return bucketStart.Format("01-02 15:04")
	}
}
