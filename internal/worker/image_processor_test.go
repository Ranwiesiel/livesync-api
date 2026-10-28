package worker

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/disintegration/imaging"
)

// samplePNG membuat gambar uji berukuran width x height.
func samplePNG(t *testing.T, width, height int) []byte {
	t.Helper()

	img := imaging.New(width, height, color.NRGBA{R: 30, G: 90, B: 160, A: 255})
	img = imaging.Overlay(img,
		imaging.New(width/2, height/2, color.NRGBA{R: 240, G: 200, B: 60, A: 255}),
		image.Pt(width/4, height/4), 1)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png encode: %v", err)
	}
	return buf.Bytes()
}

func TestTransformResizesToThumbnailWidth(t *testing.T) {
	thumb, hash, width, height, err := transform(samplePNG(t, 1200, 800))
	if err != nil {
		t.Fatalf("transform: %v", err)
	}

	if width != 400 {
		t.Fatalf("width = %d, want 400", width)
	}
	if height < 265 || height > 268 {
		t.Fatalf("height = %d, want ~267 (aspect ratio tetap)", height)
	}
	if hash == "" {
		t.Fatal("blurhash kosong")
	}

	out, err := jpeg.Decode(bytes.NewReader(thumb))
	if err != nil {
		t.Fatalf("hasil bukan JPEG valid: %v", err)
	}
	b := out.Bounds()
	if b.Dx() != 400 {
		t.Fatalf("gambar hasil %d px, want 400", b.Dx())
	}
}

func TestThumbKeyKeepsDirectoryAndPrefixesName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"uploads/2026/10/22/abc-foto.jpg", "uploads/2026/10/22/thumb_abc-foto.jpg"},
		{"abc-foto.png", "thumb_abc-foto.jpg"},
		{"tanpa-ekstensi", "thumb_tanpa-ekstensi.jpg"},
	}

	for _, c := range cases {
		if got := thumbKey(c.in); got != c.want {
			t.Fatalf("thumbKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
