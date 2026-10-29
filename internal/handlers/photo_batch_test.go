package handlers

import (
	"strings"
	"testing"
)

func TestPlanBatchMenormalkanDanMembuangDuplikat(t *testing.T) {
	plan, err := planBatch(ConfirmPhotosBatchRequest{
		EventID:  "  66f1c0ffee00000000000002  ",
		Uploader: "Peserta",
		Photos: []BatchPhotoItem{
			{StorageKey: " uploads/a.png "},
			{StorageKey: "uploads/b.png", BlurHash: "LKO2"},
			{StorageKey: "uploads/a.png", BlurHash: "duplikat"},
		},
	})
	if err != nil {
		t.Fatalf("planBatch error = %v", err)
	}

	if plan.EventID != "66f1c0ffee00000000000002" {
		t.Errorf("EventID = %q", plan.EventID)
	}
	if len(plan.Items) != 2 {
		t.Fatalf("items = %d, mau 2", len(plan.Items))
	}
	if plan.Items[0].StorageKey != "uploads/a.png" {
		t.Errorf("item[0] = %q", plan.Items[0].StorageKey)
	}
	if plan.Items[0].BlurHash != "" {
		t.Errorf("blur hash duplikat harus diabaikan, dapat %q", plan.Items[0].BlurHash)
	}
	if plan.Items[1].StorageKey != "uploads/b.png" || plan.Items[1].BlurHash != "LKO2" {
		t.Errorf("item[1] = %+v", plan.Items[1])
	}
	if len(plan.Failed) != 0 {
		t.Errorf("failed = %+v, mau kosong", plan.Failed)
	}
}

func TestPlanBatchMemisahkanKeyTidakValid(t *testing.T) {
	plan, err := planBatch(ConfirmPhotosBatchRequest{
		EventID: "66f1c0ffee00000000000002",
		Photos: []BatchPhotoItem{
			{StorageKey: "uploads/ok.png"},
			{StorageKey: "../etc/passwd"},
			{StorageKey: ""},
		},
	})
	if err != nil {
		t.Fatalf("planBatch error = %v", err)
	}

	if len(plan.Items) != 1 || plan.Items[0].StorageKey != "uploads/ok.png" {
		t.Fatalf("items = %+v", plan.Items)
	}
	if len(plan.Failed) != 2 {
		t.Fatalf("failed = %+v, mau 2", plan.Failed)
	}
	for _, failure := range plan.Failed {
		if failure.Message == "" {
			t.Errorf("failure tanpa pesan: %+v", failure)
		}
	}
}

func TestPlanBatchMenolakBatchKosong(t *testing.T) {
	_, err := planBatch(ConfirmPhotosBatchRequest{
		EventID: "66f1c0ffee00000000000002",
	})
	if err == nil {
		t.Fatal("batch kosong seharusnya error")
	}
}

func TestPlanBatchMenolakLebihDariBatas(t *testing.T) {
	photos := make([]BatchPhotoItem, maxBatchPhotos+1)
	for i := range photos {
		photos[i] = BatchPhotoItem{StorageKey: "uploads/foto.png"}
	}

	_, err := planBatch(ConfirmPhotosBatchRequest{
		EventID: "66f1c0ffee00000000000002",
		Photos:  photos,
	})
	if err == nil {
		t.Fatal("batch melebihi batas seharusnya error")
	}
	if !strings.Contains(err.Error(), "50") {
		t.Errorf("pesan error = %q, harus menyebut batas", err)
	}
}

func TestPlanBatchMenolakEventIDKosong(t *testing.T) {
	_, err := planBatch(ConfirmPhotosBatchRequest{
		EventID: "   ",
		Photos:  []BatchPhotoItem{{StorageKey: "uploads/foto.png"}},
	})
	if err == nil {
		t.Fatal("event_id kosong seharusnya error")
	}
}
