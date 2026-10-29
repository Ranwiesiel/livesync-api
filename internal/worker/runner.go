package worker

import (
	"context"
	"log"
	"time"
)

// thumbURLTTL sama dengan umur URL foto di handler: 24 jam.
const thumbURLTTL = 24 * time.Hour

// Run menjalankan n goroutine yang membaca JobQueue sampai ctx selesai.
func (p *Processor) Run(ctx context.Context, n int) {
	for i := 0; i < n; i++ {
		go p.loop(ctx)
	}
}

func (p *Processor) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-JobQueue:
			if !ok {
				return
			}
			p.handle(ctx, job)
		}
	}
}

// handle memproses satu job lalu menyimpan hasilnya ke database.
func (p *Processor) handle(ctx context.Context, job Job) {
	// Process sekarang menerima ID foto dan storage key.
	res, err := p.Process(ctx, job.Key)
	if err != nil {
		log.Printf("worker: gagal memproses %s: %v", job.Key, err)
		return
	}

	if p.update == nil {
		return
	}

	thumbURL, err := p.storage.PresignGet(ctx, res.ThumbKey, thumbURLTTL)
	if err != nil {
		log.Printf(
			"worker: gagal menandatangani %s: %v",
			res.ThumbKey,
			err,
		)
		return
	}

	// Gabungkan hasil pemrosesan gambar dan EXIF dalam satu update.
	fields := map[string]any{
		"thumbnail_url": thumbURL,
		"blur_hash":     res.BlurHash,
		"width":         res.Width,
		"height":        res.Height,
	}

	if res.Exif != nil {
		fields["exif"] = res.Exif
	}

	if err := p.update(ctx, job.PhotoID, fields); err != nil {
		log.Printf(
			"worker: gagal update foto %s: %v",
			job.PhotoID.Hex(),
			err,
		)
		return
	}

	log.Printf("worker: foto %s berhasil diproses", job.PhotoID.Hex())
}
