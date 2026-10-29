package worker

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/bbrks/go-blurhash"
	"github.com/disintegration/imaging"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/Ranwiesiel/livesync-api/internal/services"
)

const (
	// ThumbnailWidth adalah lebar thumbnail; tinggi mengikuti aspect ratio.
	ThumbnailWidth = 400

	jpegQuality = 82

	// Komponen BlurHash. 4x3 menghasilkan string sekitar 28 karakter.
	blurHashX = 4
	blurHashY = 3
)

// transform mengubah gambar asli menjadi thumbnail JPEG dan blurhash-nya.
// Fungsi ini murni: tidak menyentuh jaringan maupun disk.
func transform(original []byte) ([]byte, string, int, int, error) {
	img, err := imaging.Decode(bytes.NewReader(original), imaging.AutoOrientation(true))
	if err != nil {
		return nil, "", 0, 0, fmt.Errorf("worker: decode gambar: %w", err)
	}

	resized := imaging.Resize(img, ThumbnailWidth, 0, imaging.Lanczos)

	var buf bytes.Buffer
	if err := imaging.Encode(&buf, resized, imaging.JPEG, imaging.JPEGQuality(jpegQuality)); err != nil {
		return nil, "", 0, 0, fmt.Errorf("worker: encode jpeg: %w", err)
	}

	hash, err := blurhash.Encode(blurHashX, blurHashY, resized)
	if err != nil {
		return nil, "", 0, 0, fmt.Errorf("worker: blurhash: %w", err)
	}

	b := resized.Bounds()
	return buf.Bytes(), hash, b.Dx(), b.Dy(), nil
}

// thumbKey menaruh thumbnail di folder yang sama dengan file aslinya.
// Memakai paket path (bukan filepath) karena ini key object, bukan path disk.
func thumbKey(originalKey string) string {
	dir := path.Dir(originalKey)
	name := path.Base(originalKey)
	name = strings.TrimSuffix(name, path.Ext(name))

	key := "thumb_" + name + ".jpg"
	if dir == "." {
		return key
	}
	return dir + "/" + key
}

// Result adalah hasil pemrosesan satu foto.
type Result struct {
	ThumbKey  string
	BlurHash  string
	Width     int
	Height    int
	SizeBytes int
}

// UpdateFunc menyimpan hasil pemrosesan ke database.
type UpdateFunc func(ctx context.Context, id bson.ObjectID, fields map[string]any) error

// Processor mengubah foto asli di R2 menjadi thumbnail dan blurhash.
type Processor struct {
	storage *services.R2Storage
	update  UpdateFunc
}

func New(storage *services.R2Storage, update UpdateFunc) *Processor {
	return &Processor{storage: storage, update: update}
}

// Process mengambil object asli dari R2, membuat thumbnail, lalu mengunggah
// thumbnail tersebut kembali ke bucket.
func (p *Processor) Process(ctx context.Context, key string) (Result, error) {
	original, contentType, err := p.storage.GetObject(ctx, key)
	if err != nil {
		return Result{}, err
	}

	if contentType != "" && !strings.HasPrefix(contentType, "image/") {
		return Result{}, fmt.Errorf("worker: object %q bukan gambar (%s)", key, contentType)
	}

	thumb, hash, width, height, err := transform(original)
	if err != nil {
		return Result{}, err
	}

	destKey := thumbKey(key)
	if err := p.storage.PutObject(ctx, destKey, thumb, "image/jpeg"); err != nil {
		return Result{}, err
	}

	return Result{
		ThumbKey:  destKey,
		BlurHash:  hash,
		Width:     width,
		Height:    height,
		SizeBytes: len(thumb),
	}, nil
}
