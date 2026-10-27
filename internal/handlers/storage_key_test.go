package handlers

import (
	"regexp"
	"testing"
)

func TestBuildObjectKey(t *testing.T) {
	key := buildObjectKey("../../Foto Pantai!!.PNG")

	pattern := regexp.MustCompile(
		`^uploads/\d{4}/\d{2}/\d{2}/[0-9a-f]{32}-Foto-Pantai--\.PNG$`,
	)
	if !pattern.MatchString(key) {
		t.Fatalf("buildObjectKey() = %q, tidak cocok dengan pola", key)
	}
}

func TestBuildObjectKeyUnik(t *testing.T) {
	const jumlah = 200

	keys := make(map[string]struct{}, jumlah)
	for i := 0; i < jumlah; i++ {
		key := buildObjectKey("foto.png")
		if _, ada := keys[key]; ada {
			t.Fatalf("buildObjectKey() menghasilkan key duplikat: %q", key)
		}
		keys[key] = struct{}{}
	}
}

func TestBuildObjectKeyNamaKosong(t *testing.T) {
	key := buildObjectKey("...")

	pattern := regexp.MustCompile(
		`^uploads/\d{4}/\d{2}/\d{2}/[0-9a-f]{32}-file$`,
	)
	if !pattern.MatchString(key) {
		t.Fatalf("buildObjectKey() = %q, tidak memakai nama cadangan", key)
	}
}
