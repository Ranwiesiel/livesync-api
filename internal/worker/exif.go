package worker

import (
	"bytes"
	"strings"

	"github.com/Ranwiesiel/livesync-api/internal/models"
	"github.com/rwcarlsen/goexif/exif"
)

// extractEXIF membaca metadata dari byte gambar asli.
// Tidak adanya EXIF bukan kesalahan fatal bagi pipeline pemrosesan gambar.
func extractEXIF(original []byte) (*models.EXIFMetadata, error) {
	x, err := exif.Decode(bytes.NewReader(original))
	if err != nil {
		// Gambar tanpa EXIF atau format yang tidak didukung tetap diproses.
		return nil, nil
	}

	result := &models.EXIFMetadata{}

	// Make: produsen kamera.
	if tag, err := x.Get(exif.Make); err == nil {
		result.CameraMake = cleanEXIFString(tag.String())
	}

	// Model: model perangkat atau kamera.
	if tag, err := x.Get(exif.Model); err == nil {
		result.CameraModel = cleanEXIFString(tag.String())
	}

	// LensModel: lensa yang dipakai saat memotret.
	if tag, err := x.Get(exif.LensModel); err == nil {
		result.LensModel = cleanEXIFString(tag.String())
	}

	// DateTimeOriginal: waktu pengambilan foto.
	if tag, err := x.Get(exif.DateTimeOriginal); err == nil {
		result.DateTimeOriginal = cleanEXIFString(tag.String())
	}

	// ISO: sensitivitas ISO.
	if tag, err := x.Get(exif.ISOSpeedRatings); err == nil {
		if value, err := tag.Int(0); err == nil && value > 0 {
			result.ISO = &value
		}
	}

	// FNumber: aperture, misalnya f/2.8.
	if tag, err := x.Get(exif.FNumber); err == nil {
		if numerator, denominator, err := tag.Rat2(0); err == nil &&
			denominator != 0 {
			value := float64(numerator) / float64(denominator)
			if value > 0 {
				result.Aperture = &value
			}
		}
	}

	// ExposureTime: durasi eksposur.
	if tag, err := x.Get(exif.ExposureTime); err == nil {
		result.ShutterSpeed = cleanEXIFString(tag.String())
	}

	// Jangan menyimpan sub-document kosong.
	if result.CameraMake == "" &&
		result.CameraModel == "" &&
		result.LensModel == "" &&
		result.DateTimeOriginal == "" &&
		result.ISO == nil &&
		result.Aperture == nil &&
		result.ShutterSpeed == "" {
		return nil, nil
	}

	return result, nil
}

// cleanEXIFString menghapus spasi dan karakter null di akhir string.
func cleanEXIFString(value string) string {
	return strings.TrimSpace(strings.TrimRight(value, "\x00"))
}
