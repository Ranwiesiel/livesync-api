package worker

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"path"
	"strings"

	"github.com/bbrks/go-blurhash"
	"github.com/disintegration/imaging"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/Ranwiesiel/livesync-api/internal/models"
	"github.com/Ranwiesiel/livesync-api/internal/services"
)

const (
	// ThumbnailWidth adalah lebar thumbnail; tinggi mengikuti aspect ratio.
	ThumbnailWidth = 400

	jpegQuality = 82

	// Komponen BlurHash.
	blurHashX = 4
	blurHashY = 3
)

// Batas ukuran masukan, supaya file raksasa tidak menghabiskan RAM.
const (
	// maxInputBytes membatasi ukuran file asli yang diproses (20 MB).
	maxInputBytes = 20 << 20

	// maxPixels membatasi jumlah piksel; 50 MP setara foto 8660x5773.
	maxPixels = 50_000_000
)

// tooLarge melaporkan apakah dimensi gambar melewati batas piksel.
func tooLarge(width, height int) bool {
	return width*height > maxPixels
}

// transform mengubah gambar asli menjadi thumbnail JPEG dan BlurHash.
// Fungsi ini tidak menyentuh jaringan maupun disk.
func transform(original []byte) ([]byte, string, int, int, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(original))
	if err != nil {
		return nil, "", 0, 0, fmt.Errorf("worker: baca header gambar: %w", err)
	}
	if tooLarge(cfg.Width, cfg.Height) {
		return nil, "", 0, 0, fmt.Errorf(
			"worker: gambar terlalu besar (%dx%d, maks %d piksel)",
			cfg.Width, cfg.Height, maxPixels)
	}

	img, err := imaging.Decode(
		bytes.NewReader(original),
		imaging.AutoOrientation(true),
	)
	if err != nil {
		return nil, "", 0, 0, fmt.Errorf(
			"worker: decode gambar: %w",
			err,
		)
	}

	resized := imaging.Resize(img, ThumbnailWidth, 0, imaging.Lanczos)

	var buf bytes.Buffer
	if err := imaging.Encode(
		&buf,
		resized,
		imaging.JPEG,
		imaging.JPEGQuality(jpegQuality),
	); err != nil {
		return nil, "", 0, 0, fmt.Errorf(
			"worker: encode jpeg: %w",
			err,
		)
	}

	hash, err := blurhash.Encode(blurHashX, blurHashY, resized)
	if err != nil {
		return nil, "", 0, 0, fmt.Errorf(
			"worker: blurhash: %w",
			err,
		)
	}

	bounds := resized.Bounds()

	return buf.Bytes(), hash, bounds.Dx(), bounds.Dy(), nil
}

// thumbKey menaruh thumbnail di prefix terpisah, mengikuti path file aslinya.
// Prefix terpisah penting supaya aturan lifecycle bisa berbeda dari foto asli
// dan thumbnail tidak lolos validasi sebagai foto asli.
func thumbKey(originalKey string) string {
	k := strings.TrimPrefix(originalKey, "uploads/")

	dir := path.Dir(k)
	name := strings.TrimSuffix(path.Base(k), path.Ext(k))

	if dir == "." {
		return "thumbnails/thumb_" + name + ".jpg"
	}

	return "thumbnails/" + dir + "/thumb_" + name + ".jpg"
}

// Result adalah hasil pemrosesan satu foto.
type Result struct {
	ThumbKey  string
	BlurHash  string
	Width     int
	Height    int
	SizeBytes int
	Exif      *models.EXIFMetadata
}

// Processor memproses gambar asli dari R2 menjadi thumbnail.
type Processor struct {
	storage *services.R2Storage
	update  UpdateFunc
}

// UpdateFunc menyimpan hasil pemrosesan ke database.
type UpdateFunc func(
	ctx context.Context,
	id bson.ObjectID,
	fields map[string]any,
) error

func New(storage *services.R2Storage, update UpdateFunc) *Processor {
	return &Processor{
		storage: storage,
		update:  update,
	}
}

// Process mengambil gambar asli dari R2, mengekstrak EXIF,
// membuat thumbnail, dan mengunggah thumbnail ke R2.
// Penyimpanan hasil ke MongoDB dilakukan oleh runner.
func (p *Processor) Process(
	ctx context.Context,
	key string,
) (Result, error) {
	// 1. Ambil gambar asli dari R2.
	original, contentType, err := p.storage.GetObject(ctx, key)
	if err != nil {
		return Result{}, fmt.Errorf(
			"worker: mengambil foto asli: %w",
			err,
		)
	}

	if contentType != "" &&
		!strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return Result{}, fmt.Errorf(
			"worker: object %q bukan gambar (%s)",
			key,
			contentType,
		)
	}

	if len(original) > maxInputBytes {
		return Result{}, fmt.Errorf(
			"worker: object %q terlalu besar (%d bytes, maks %d)",
			key, len(original), maxInputBytes,
		)
	}

	// 2. Ekstrak EXIF dari gambar asli sebelum transformasi.
	exifMetadata, err := extractEXIF(original)
	if err != nil {
		return Result{}, fmt.Errorf(
			"worker: ekstraksi EXIF: %w",
			err,
		)
	}

	// 3. Buat thumbnail dan BlurHash.
	thumb, hash, width, height, err := transform(original)
	if err != nil {
		return Result{}, err
	}

	// 4. Unggah thumbnail ke R2.
	destKey := thumbKey(key)
	if err := p.storage.PutObject(
		ctx,
		destKey,
		thumb,
		"image/jpeg",
	); err != nil {
		return Result{}, fmt.Errorf(
			"worker: mengunggah thumbnail: %w",
			err,
		)
	}

	// 5. Kembalikan seluruh hasil untuk disimpan oleh runner.
	return Result{
		ThumbKey:  destKey,
		BlurHash:  hash,
		Width:     width,
		Height:    height,
		SizeBytes: len(thumb),
		Exif:      exifMetadata,
	}, nil
}
