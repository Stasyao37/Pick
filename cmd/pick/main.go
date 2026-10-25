// Команда pick запускает TUI-центр приложений.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nook-apps/pick/internal/backend/mock"
	"github.com/nook-apps/pick/internal/tui"
)

func main() {
	if os.Geteuid() == 0 {
		fmt.Fprintln(os.Stderr, "pick: запуск от root не поддерживается, права будут запрошены через polkit")
		os.Exit(1)
	}

	// Пока работаем на выдуманных данных; позже здесь появится выбор
	// настоящих бэкендов (Flatpak, PackageKit).
	be := mock.New()

	if _, err := tea.NewProgram(tui.New(be), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "pick:", err)
		os.Exit(1)
	}
}
