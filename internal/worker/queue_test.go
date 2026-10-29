package worker

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestEnqueueTidakMemblokirSaatPenuh(t *testing.T) {
	// Kosongkan antrean global setelah test selesai.
	defer func() {
		for len(JobQueue) > 0 {
			<-JobQueue
		}
	}()

	for i := 0; i < cap(JobQueue); i++ {
		if !Enqueue(Job{PhotoID: bson.NewObjectID(), Key: "uploads/x.jpg"}) {
			t.Fatalf("Enqueue gagal pada iterasi %d, antrean belum penuh", i)
		}
	}

	if Enqueue(Job{PhotoID: bson.NewObjectID(), Key: "uploads/y.jpg"}) {
		t.Fatal("Enqueue harus mengembalikan false saat antrean penuh")
	}
}
