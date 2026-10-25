// Package mock — выдуманный бэкенд для разработки интерфейса без
// настоящего пакетного менеджера.
package mock

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nook-apps/pick/internal/backend"
)

// Backend хранит каталог в памяти и имитирует установку.
type Backend struct {
	mu   sync.Mutex
	apps []backend.Details
	// StepDelay — пауза между шагами имитации установки.
	StepDelay time.Duration
}

// New возвращает бэкенд с набором выдуманных приложений.
func New() *Backend {
	b := &Backend{StepDelay: 150 * time.Millisecond}
	for _, a := range catalog {
		b.apps = append(b.apps, backend.Details{
			Package: backend.Package{
				Ref:          backend.PackageRef{Backend: "mock", ID: a.id, Origin: "flathub", Scope: backend.ScopeUser},
				AppStreamID:  a.id,
				Name:         a.name,
				Summary:      a.summary,
				Version:      a.version,
				Installed:    a.installed,
				DownloadSize: a.sizeMB << 20,
			},
			Description: a.summary + ". Это тестовая запись mock-бэкенда.",
			License:     a.license,
		})
	}
	return b
}

func (b *Backend) ID() string { return "mock" }

func (b *Backend) Caps() backend.Caps {
	return backend.Caps{UserScope: true, SystemScope: true, DryRun: true, AppStream: true}
}

func (b *Backend) Available(context.Context) bool { return true }

func (b *Backend) Refresh(context.Context) error { return nil }

func (b *Backend) Search(_ context.Context, q backend.Query) ([]backend.Package, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	text := strings.ToLower(strings.TrimSpace(q.Text))
	var out []backend.Package
	for _, a := range b.apps {
		if text == "" || strings.Contains(strings.ToLower(a.Name), text) ||
			strings.Contains(strings.ToLower(a.Summary), text) {
			out = append(out, a.Package)
		}
	}
	return out, nil
}

func (b *Backend) Installed(ctx context.Context) ([]backend.Package, error) {
	all, _ := b.Search(ctx, backend.Query{})
	var out []backend.Package
	for _, p := range all {
		if p.Installed {
			out = append(out, p)
		}
	}
	return out, nil
}

func (b *Backend) Updates(context.Context) ([]backend.Package, error) { return nil, nil }

func (b *Backend) Details(_ context.Context, ref backend.PackageRef) (*backend.Details, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, a := range b.apps {
		if a.Ref.ID == ref.ID {
			d := a
			return &d, nil
		}
	}
	return nil, fmt.Errorf("mock: пакет %q не найден", ref.ID)
}

func (b *Backend) Plan(ctx context.Context, op backend.Operation) (*backend.Plan, error) {
	p := &backend.Plan{Op: op}
	for _, ref := range op.Refs {
		d, err := b.Details(ctx, ref)
		if err != nil {
			return nil, err
		}
		p.Packages = append(p.Packages, d.Package)
		p.DownloadSize += d.DownloadSize
		if ref.Scope == backend.ScopeSystem {
			p.NeedsPassword = true
		}
	}
	return p, nil
}

func (b *Backend) Apply(ctx context.Context, p *backend.Plan, events chan<- backend.Event) error {
	defer close(events)
	steps := []string{"Загрузка", "Загрузка", "Загрузка", "Проверка", "Установка", "Установка", "Готово"}
	for i, s := range steps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(b.StepDelay):
		}
		events <- backend.Event{Percent: float64(i+1) / float64(len(steps)), Step: s}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ref := range p.Op.Refs {
		for i := range b.apps {
			if b.apps[i].Ref.ID == ref.ID {
				b.apps[i].Installed = p.Op.Kind != backend.OpRemove
			}
		}
	}
	return nil
}

type app struct {
	id, name, summary, version, license string
	sizeMB                              int64
	installed                           bool
}

var catalog = []app{
	{"org.gimp.GIMP", "GIMP", "Графический редактор", "3.0.4", "GPL-3.0", 320, false},
	{"org.mozilla.firefox", "Firefox", "Веб-браузер", "131.0", "MPL-2.0", 110, true},
	{"org.telegram.desktop", "Telegram", "Мессенджер", "5.6.1", "GPL-3.0", 90, false},
	{"org.videolan.VLC", "VLC", "Медиаплеер", "3.0.21", "GPL-2.0", 75, false},
	{"org.libreoffice.LibreOffice", "LibreOffice", "Офисный пакет", "24.8.2", "MPL-2.0", 650, false},
	{"com.obsproject.Studio", "OBS Studio", "Запись и трансляция экрана", "30.2", "GPL-2.0", 210, false},
	{"org.inkscape.Inkscape", "Inkscape", "Векторный редактор", "1.4", "GPL-3.0", 180, false},
	{"org.blender.Blender", "Blender", "3D-моделирование и анимация", "4.2", "GPL-3.0", 400, false},
	{"com.spotify.Client", "Spotify", "Музыкальный сервис", "1.2.45", "Proprietary", 160, false},
	{"org.kde.krita", "Krita", "Цифровое рисование", "5.2.6", "GPL-3.0", 250, true},
}
