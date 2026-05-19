package repository

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gridea-pro/backend/internal/domain"
)

func TestMemoListSortedByCreatedAtDesc(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "config"), 0755)
	repo := NewMemoRepository(dir)
	ctx := context.Background()

	now := time.Now()
	stored := []domain.Memo{
		{ID: "mid", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now, Tags: []string{}, Images: []string{}},
		{ID: "new", CreatedAt: now, UpdatedAt: now, Tags: []string{}, Images: []string{}},
		{ID: "old", CreatedAt: now.Add(-5 * time.Hour), UpdatedAt: now, Tags: []string{}, Images: []string{}},
	}
	if err := repo.SaveAll(ctx, stored); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	got, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	want := []string{"new", "mid", "old"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want=%d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID != id {
			gotIDs := make([]string, len(got))
			for j, m := range got {
				gotIDs[j] = m.ID
			}
			t.Fatalf("排序错误：got=%v want=%v", gotIDs, want)
		}
	}
}

func TestMemoSaveUsesLocalWallClockFormat(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "config"), 0755)

	utcMoment := time.Date(2026, 5, 15, 8, 32, 17, 0, time.UTC)
	localMoment := time.Date(2026, 5, 15, 16, 42, 38, 0, time.Local)

	repo := NewMemoRepository(dir)
	ctx := context.Background()
	if err := repo.SaveAll(ctx, []domain.Memo{
		{ID: "from-utc", CreatedAt: utcMoment, UpdatedAt: localMoment, Tags: []string{}, Images: []string{}},
	}); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "config", "memos.json"))
	if err != nil {
		t.Fatalf("read memos.json: %v", err)
	}
	body := string(data)

	if strings.Contains(body, "Z\"") || strings.Contains(body, "+08:00") || strings.Contains(body, "+00:00") {
		t.Fatalf("memos.json 不应再包含 RFC3339 TZ 后缀:\n%s", body)
	}

	wantCreated := utcMoment.Local().Format(domain.TimeLayout)
	wantUpdated := localMoment.Format(domain.TimeLayout)
	if !strings.Contains(body, wantCreated) {
		t.Fatalf("createdAt 期望本地 wall clock %q:\n%s", wantCreated, body)
	}
	if !strings.Contains(body, wantUpdated) {
		t.Fatalf("updatedAt 期望本地 wall clock %q:\n%s", wantUpdated, body)
	}
}

func TestMemoLoadAcceptsLegacyRFC3339(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "config"), 0755)

	legacy := `{
  "memos": [
    {
      "id": "z-form",
      "content": "存的是 Z",
      "tags": [],
      "images": [],
      "createdAt": "2026-05-15T08:32:17Z",
      "updatedAt": "2026-05-15T16:42:38+08:00"
    },
    {
      "id": "offset-form",
      "content": "存的是 +08:00",
      "tags": [],
      "images": [],
      "createdAt": "2026-02-05T09:02:51+08:00",
      "updatedAt": "2026-05-15T17:03:01+08:00"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, "config", "memos.json"), []byte(legacy), 0644); err != nil {
		t.Fatalf("write legacy: %v", err)
	}

	repo := NewMemoRepository(dir)
	memos, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(memos) != 2 {
		t.Fatalf("len=%d want=2", len(memos))
	}
	for _, m := range memos {
		if m.CreatedAt.IsZero() || m.UpdatedAt.IsZero() {
			t.Fatalf("时间没解析出来: %+v", m)
		}
	}
}

func TestMemoListReturnsCopy(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "config"), 0755)
	repo := NewMemoRepository(dir)
	ctx := context.Background()

	now := time.Now()
	stored := []domain.Memo{
		{ID: "a", CreatedAt: now.Add(-3 * time.Hour), Tags: []string{}, Images: []string{}},
		{ID: "b", CreatedAt: now.Add(-2 * time.Hour), Tags: []string{}, Images: []string{}},
		{ID: "c", CreatedAt: now.Add(-1 * time.Hour), Tags: []string{}, Images: []string{}},
	}
	if err := repo.SaveAll(ctx, stored); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	out1, _ := repo.List(ctx)
	if len(out1) > 0 {
		out1[0].ID = "X"
	}
	out2, _ := repo.List(ctx)
	for _, m := range out2 {
		if m.ID == "X" {
			t.Fatalf("List 返回的不是副本：外部修改污染了 cache")
		}
	}
}
