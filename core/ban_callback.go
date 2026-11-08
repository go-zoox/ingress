package core

// BanHitCallback is notified on every runtime ban match.
//
// The admin package implements this to count hits in memory; the request path
// must never write to the database, so counting is fire-and-forget and flushed
// on a timer.
type BanHitCallback interface {
	OnBanHit(banID uint)
}
