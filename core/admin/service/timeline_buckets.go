package service

import "time"

// truncateTime floors t to the nearest slot boundary.
func truncateTime(t time.Time, slot time.Duration) time.Time {
	if slot <= 0 {
		return t
	}
	return t.Truncate(slot)
}

// alignedTimelineEnd returns the end of the current bucket for a wall-clock aligned timeline.
func alignedTimelineEnd(anchor time.Time, slot time.Duration) time.Time {
	if slot <= 0 {
		return anchor
	}
	end := truncateTime(anchor, slot)
	if end.Before(anchor) {
		end = end.Add(slot)
	}
	return end
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
