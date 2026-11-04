
package websocket

import (
	"sync"
	"time"
)

const (
	reactionBucketCapacity = 5
	reactionRefillRate     = 5.0 // token per detik
)

// TokenBucket mengatur jumlah pesan yang dapat dikirim klien.
type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	lastRefill time.Time
}

// NewTokenBucket membuat limiter dengan bucket penuh.
func NewTokenBucket() *TokenBucket {
	return &TokenBucket{
		tokens:     reactionBucketCapacity,
		lastRefill: time.Now(),
	}
}

// Allow mengembalikan true jika klien masih memiliki token.
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()

	// Isi kembali token berdasarkan waktu yang telah berlalu.
	tb.tokens += elapsed * reactionRefillRate

	// Batasi jumlah token sesuai kapasitas bucket.
	if tb.tokens > reactionBucketCapacity {
		tb.tokens = reactionBucketCapacity
	}

	tb.lastRefill = now

	// Tidak ada token tersisa.
	if tb.tokens < 1 {
		return false
	}

	// Gunakan satu token untuk satu reaksi.
	tb.tokens--

	return true
}
