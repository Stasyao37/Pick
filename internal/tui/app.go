// Package tui — интерфейс Pick на Bubble Tea.
//
// Bubble Tea устроен по Elm-архитектуре: есть модель (состояние),
// Update (получает сообщение и возвращает новое состояние) и View
// (рисует состояние строкой). Вся долгая работа идёт через tea.Cmd,
// чтобы интерфейс не зависал.
package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nook-apps/pick/internal/backend"
)

type screen int

const (
	screenList screen = iota
	screenDetails
	screenInstall
)

// Model — корневое состояние приложения.
type Model struct {
	be     backend.Backend
	screen screen
	list   list.Model
	width  int

	details *backend.Details
	bar     progress.Model
	running installStarted
	percent float64
	step    string
	notice  string
	err     error
}

// item оборачивает backend.Package для компонента list.
type item struct{ pkg backend.Package }

func (i item) Title() string {
	if i.pkg.Installed {
		return i.pkg.Name + " ✓"
	}
	return i.pkg.Name
}
func (i item) Description() string { return i.pkg.Summary }
func (i item) FilterValue() string { return i.pkg.Name + " " + i.pkg.Summary }

// New создаёт модель поверх любого бэкенда.
func New(be backend.Backend) Model {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Pick"
	l.SetStatusBarItemName("приложение", "приложений")
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "открыть"))}
	}
	return Model{be: be, list: l, bar: progress.New(progress.WithDefaultGradient())}
}

// --- сообщения ---

type packagesMsg []backend.Package
type detailsMsg *backend.Details
type errMsg struct{ err error }

// installStarted несёт каналы запущенной установки.
type installStarted struct {
	events <-chan backend.Event
	result <-chan error
}
type progressMsg backend.Event
type installDone struct{ err error }

// --- команды ---

func loadPackages(be backend.Backend) tea.Cmd {
	return func() tea.Msg {
		pkgs, err := be.Search(context.Background(), backend.Query{})
		if err != nil {
			return errMsg{err}
		}
		return packagesMsg(pkgs)
	}
}

func loadDetails(be backend.Backend, ref backend.PackageRef) tea.Cmd {
	return func() tea.Msg {
		d, err := be.Details(context.Background(), ref)
		if err != nil {
			return errMsg{err}
		}
		return detailsMsg(d)
	}
}

// startApply готовит план и запускает Apply в отдельной горутине.
func startApply(be backend.Backend, op backend.Operation) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		plan, err := be.Plan(ctx, op)
		if err != nil {
			return installDone{err}
		}
		events := make(chan backend.Event)
		result := make(chan error, 1)
		go func() { result <- be.Apply(ctx, plan, events) }()
		return installStarted{events: events, result: result}
	}
}

// waitEvent ждёт следующее событие прогресса; когда канал закрыт —
// возвращает результат установки.
func waitEvent(s installStarted) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-s.events
		if !ok {
			return installDone{<-s.result}
		}
		return progressMsg(ev)
	}
}

// --- Elm-цикл ---

func (m Model) Init() tea.Cmd { return loadPackages(m.be) }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.list.SetSize(msg.Width, msg.Height-1)
		m.bar.Width = min(msg.Width-4, 60)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case packagesMsg:
		items := make([]list.Item, len(msg))
		for i, p := range msg {
			items[i] = item{p}
		}
		return m, m.list.SetItems(items)

	case detailsMsg:
		m.details = msg
		m.screen = screenDetails
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case installStarted:
		m.running = msg
		return m, waitEvent(msg)
	}

	switch m.screen {
	case screenDetails:
		return m.updateDetails(msg)
	case screenInstall:
		return m.updateInstall(msg)
	}
	return m.updateList(msg)
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && m.list.FilterState() != list.Filtering {
		switch k.String() {
		case "enter":
			if it, ok := m.list.SelectedItem().(item); ok {
				return m, loadDetails(m.be, it.pkg.Ref)
			}
		case "q":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) updateDetails(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch k.String() {
	case "esc", "backspace", "q":
		m.screen = screenList
		m.err, m.notice = nil, ""
	case "i", "enter":
		kind := backend.OpInstall
		if m.details.Installed {
			kind = backend.OpRemove
		}
		m.screen = screenInstall
		m.percent, m.step, m.err, m.notice = 0, "Подготовка", nil, ""
		return m, startApply(m.be, backend.Operation{Kind: kind, Refs: []backend.PackageRef{m.details.Ref}})
	}
	return m, nil
}

func (m Model) updateInstall(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case progressMsg:
		m.percent, m.step = msg.Percent, msg.Step
		// Следующее событие ждём новой командой: так UI остаётся отзывчивым.
		return m, waitEvent(m.running)
	case installDone:
		m.err = msg.err
		if msg.err == nil {
			m.notice = "Установлено"
			if m.details.Installed {
				m.notice = "Удалено"
			}
		}
		// Перечитываем список и карточку: detailsMsg вернёт нас на экран карточки.
		return m, tea.Batch(loadPackages(m.be), loadDetails(m.be, m.details.Ref))
	}
	return m, nil
}

// --- отрисовка ---

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	boxStyle   = lipgloss.NewStyle().Padding(1, 2)
)

func (m Model) View() string {
	switch m.screen {
	case screenDetails, screenInstall:
		return boxStyle.Render(m.viewDetails())
	}
	v := m.list.View()
	if m.err != nil {
		v += "\n" + errStyle.Render(m.err.Error())
	}
	return v
}

func (m Model) viewDetails() string {
	d := m.details
	s := titleStyle.Render(d.Name) + "  " + dimStyle.Render(d.Version) + "\n\n"
	s += d.Description + "\n\n"
	s += dimStyle.Render(fmt.Sprintf("ID: %s\nИсточник: %s (%s)\nЛицензия: %s\nРазмер загрузки: %d МБ",
		d.Ref.ID, d.Ref.Origin, d.Ref.Scope, d.License, d.DownloadSize>>20)) + "\n\n"

	if m.screen == screenInstall {
		s += m.bar.ViewAs(m.percent) + "  " + m.step + "\n\n"
	}
	if m.notice != "" {
		s += okStyle.Render(m.notice) + "\n\n"
	}
	if m.err != nil {
		s += errStyle.Render("Ошибка: "+m.err.Error()) + "\n\n"
	}

	action := "i — установить"
	if d.Installed {
		action = "i — удалить"
	}
	return s + dimStyle.Render(action+" · esc — назад · ctrl+c — выход")
}
