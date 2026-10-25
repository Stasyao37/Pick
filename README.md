# Pick

TUI-центр приложений для Linux: поиск и установка программ из удалённых
репозиториев в одном интерфейсе. Сначала Flatpak, затем dnf и apt через
PackageKit.

> Сейчас приложение работает на выдуманных данных (mock-бэкенд): это
> каркас для разработки интерфейса.

## Запуск

Нужен Go 1.24+.

```sh
go run ./cmd/pick
```

Клавиши: `↑/↓` — выбор, `/` — поиск, `enter` — карточка приложения,
`i` — установить или удалить, `esc` — назад, `q` или `ctrl+c` — выход.

## Ярлык в меню

```sh
go install ./cmd/pick
cp data/io.github.nook_apps.Pick.desktop ~/.local/share/applications/
cp -r data/icons/hicolor ~/.local/share/icons/
```

Ярлык запускает `pick` в терминале (`Terminal=true`), поэтому `~/go/bin` должен быть в `PATH`
не только у оболочки, но и у сеанса: лаунчеры Hyprland берут `PATH` из него.

## Тесты

```sh
go test ./...
```

## Структура

```text
cmd/pick/                 точка входа
internal/backend/         интерфейс Backend и общие типы
internal/backend/mock/    выдуманный бэкенд для разработки UI
internal/tui/             интерфейс на Bubble Tea
```

Новый пакетный менеджер — это новый пакет в `internal/backend/<name>`,
реализующий интерфейс `Backend`. Интерфейс при этом не меняется.
