package mock

import (
	"context"
	"testing"

	"github.com/Stasyao37/pick/internal/backend"
)

// Проверка на этапе компиляции: mock реализует интерфейс Backend.
var _ backend.Backend = (*Backend)(nil)

func TestSearch(t *testing.T) {
	b := New()
	got, err := b.Search(context.Background(), backend.Query{Text: "gimp"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "GIMP" {
		t.Fatalf("ожидали только GIMP, получили %v", got)
	}
}

func TestInstallMarksInstalled(t *testing.T) {
	b := New()
	b.StepDelay = 0
	ctx := context.Background()
	ref := backend.PackageRef{Backend: "mock", ID: "org.gimp.GIMP"}

	plan, err := b.Plan(ctx, backend.Operation{Kind: backend.OpInstall, Refs: []backend.PackageRef{ref}})
	if err != nil {
		t.Fatal(err)
	}
	events := make(chan backend.Event)
	done := make(chan error, 1)
	go func() { done <- b.Apply(ctx, plan, events) }()

	var last backend.Event
	for ev := range events {
		last = ev
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if last.Percent != 1 {
		t.Fatalf("последнее событие должно быть 100%%, получили %v", last.Percent)
	}
	d, _ := b.Details(ctx, ref)
	if !d.Installed {
		t.Fatal("после установки пакет должен быть отмечен установленным")
	}
}
