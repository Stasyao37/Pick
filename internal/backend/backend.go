// Package backend описывает общий интерфейс, через который ядро Pick
// работает с любым пакетным менеджером (Flatpak, PackageKit и т.д.).
package backend

import "context"

// Scope — область установки.
type Scope int

const (
	ScopeUser Scope = iota
	ScopeSystem
)

func (s Scope) String() string {
	if s == ScopeSystem {
		return "system"
	}
	return "user"
}

// PackageRef однозначно указывает на пакет внутри конкретного бэкенда.
type PackageRef struct {
	Backend string // "flatpak", "packagekit", "mock"
	ID      string // app/org.gimp.GIMP/x86_64/stable или имя пакета
	Origin  string // ремоут или репозиторий: flathub, fedora
	Scope   Scope
}

// Package — запись в каталоге или в списке установленного.
type Package struct {
	Ref           PackageRef
	AppStreamID   string // общий ключ для объединения источников
	Name          string
	Summary       string
	Version       string
	Installed     bool
	DownloadSize  int64
	InstalledSize int64
}

// Details — расширенная информация для карточки приложения.
type Details struct {
	Package
	Description string
	License     string
	Homepage    string
}

// Caps описывает, что умеет бэкенд. UI скрывает то, чего бэкенд не умеет.
type Caps struct {
	UserScope   bool
	SystemScope bool
	DryRun      bool
	AppStream   bool
}

// Query — поисковый запрос.
type Query struct {
	Text string
}

// OpKind — вид операции.
type OpKind int

const (
	OpInstall OpKind = iota
	OpRemove
	OpUpdate
)

// Operation — то, что пользователь хочет сделать.
type Operation struct {
	Kind OpKind
	Refs []PackageRef
}

// Plan — результат подготовки операции: что именно будет сделано.
type Plan struct {
	Op            Operation
	Packages      []Package // включая зависимости
	DownloadSize  int64
	NeedsPassword bool
}

// Event — событие прогресса во время Apply.
type Event struct {
	Percent float64 // 0..1
	Step    string
	Log     string
}

// Backend реализуется каждым пакетным менеджером.
type Backend interface {
	ID() string
	Caps() Caps
	Available(ctx context.Context) bool
	Refresh(ctx context.Context) error
	Search(ctx context.Context, q Query) ([]Package, error)
	Installed(ctx context.Context) ([]Package, error)
	Updates(ctx context.Context) ([]Package, error)
	Details(ctx context.Context, ref PackageRef) (*Details, error)
	Plan(ctx context.Context, op Operation) (*Plan, error)
	// Apply выполняет план и закрывает events по завершении.
	Apply(ctx context.Context, p *Plan, events chan<- Event) error
}
