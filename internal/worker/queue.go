package worker

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Job adalah satu foto yang menunggu diproses.
type Job struct {
	PhotoID   bson.ObjectID
	EventID   bson.ObjectID
	Key       string
	Uploader  string
	CreatedAt time.Time
}

// JobQueue menampung tugas. Buffered supaya request HTTP tidak menunggu worker.
var JobQueue = make(chan Job, 100)

// Enqueue memasukkan job tanpa memblokir request. Nilai false berarti antrean
// penuh; pemanggil yang memutuskan cara menanganinya.
func Enqueue(job Job) bool {
	select {
	case JobQueue <- job:
		return true
	default:
		return false
	}
}
