# Лифты

Симулятор системы контроля работы лифтов: Go 1.25, React + TypeScript, PostgreSQL и Docker Compose. Пользователь управляет своим сеансом, администратор также может управлять чужими сеансами и назначать роли.

## Запуск

Нужен работающий Docker Desktop с Linux containers.

```powershell
Copy-Item .env.example .env
docker compose up -d --build
```

Открой `http://localhost:8080`. Пароль PostgreSQL в `.env` предназначен для локальной разработки; перед использованием на сервере замени его. Если пароль содержит специальные символы URL (`@`, `:`, `/`, `%`), задай корректно закодированный `DATABASE_URL` в Compose. Для HTTPS установи `COOKIE_SECURE=true` и настрой TLS на внешнем прокси.

Контейнер `migrate` применяет SQL-миграции goose и завершается. Затем запускаются backend и frontend. PostgreSQL хранит аккаунты и сессии авторизации в Docker volume. История симуляций не сохраняется; перезапуск backend теряет состояние моделей.

```powershell
docker compose logs -f backend
docker compose down
```

`down` сохраняет volume базы. Не используй `down -v`, если аккаунты нужны.

## Первый администратор

1. Зарегистрируй аккаунт через интерфейс, например `raman`.
2. Назначь ему роль прямым изменением БД:

```powershell
docker compose exec postgres psql -U elevators -d elevators -c "UPDATE users SET role = 'admin' WHERE username = 'raman';"
```

При изменении `POSTGRES_USER` / `POSTGRES_DB` подставь свои значения. Роль перепроверяется на сервере при запросах, интерфейс обновит её в течение пяти секунд. Последнего активного администратора нельзя заблокировать или понизить через API.

## Работа с симуляцией

1. Настрой здание и лифты, выбери стратегию. При необходимости добавь события на вкладке «Сценарий».
2. Нажми «Создать сеанс», затем «Старт».
3. На пульте выбери этаж отправления, этаж назначения и количество людей. «Создать людей и вызвать лифт» создаёт группу и сразу нажимает её пульт назначения: требование сделать вызов в течение трёх секунд выполняется.
4. Для людей, добавленных вручную, нажми «Ход» под нужной шахтой. Для людей из сценария система автоматически имитирует «Ход»: лифт сам приезжает на вызов и продолжает поездку после посадки, в том числе после отключения тревоги. Движущийся лифт автоматически учитывает новые вызовы.
5. Останови сеанс, когда все кабины пустые и остановлены. Появятся статистика и экспорт CSV. «Продолжить» возобновляет тот же сеанс, время, сценарий и накопленные показатели.

Кнопки «Вызвать» у этажей выбирают соответствующий этаж отправления на пульте. Число рядом с этажом показывает ожидающих. Позиции, состояния людей и статусная строка обновляются через WebSocket.

Пожарная тревога направляет все лифты на первый этаж без промежуточных остановок. Она автоматически запускает движение кабин, без «Ход». Люди на этажах сразу начинают эвакуацию; люди в кабинах сначала достигают первого этажа. После начала эвакуации люди исчезают через три модельные секунды. На паузе включить тревогу нельзя. Отключение тревоги возвращает обычную работу; уже начатая эвакуация пассажиров кабин завершается на первом этаже.

## YAML без пересборки

Примеры находятся в `examples/config.yaml` и `examples/scenario.yaml`.

Во вкладках «Конфигурация» и «Сценарий» доступны загрузка и скачивание YAML. Для существующего сеанса сервер проверяет файл и сразу применяет его **до первого старта**. После первого старта конфигурация, стратегия и сценарий фиксированы. Для другого YAML останови и удали свой сеанс, затем создай новый. Добавлять людей вручную можно в любом состоянии сеанса.

Файлы не встраиваются в Docker images. При загрузке данные проходят через HTTP API в работающий Go-процесс. Ни перезапуск контейнеров, ни пересборка образа не требуются.

## Архитектура

```text
backend/
  cmd/server/main.go              запуск HTTP-приложения
  cmd/migrate/main.go             запуск goose: up, down, status
  internal/app/app.go             связывание зависимостей и жизненный цикл
  internal/config/env.go          настройки окружения сервера
  internal/domain/               модель, физика, стратегии, пожар, статистика
  internal/services/             авторизация и управление сеансами
  internal/repository/users.go   PostgreSQL-репозиторий учётных записей
  internal/transport/http/       REST, cookie, WebSocket, импорт и экспорт
  pkg/database/                  подключение к БД
  pkg/security/                  bcrypt, случайные токены, SHA-256
  pkg/runner/                    запуск модельного таймера
  pkg/files/                     строгий YAML и CSV
  migrations/                    SQL-миграции goose с Up/Down
frontend/
  src/api/                       сетевой клиент
  src/hooks/                     подписка на сеанс
  src/components/                экраны и элементы интерфейса
  src/lib/                       YAML и черновики
  src/types.ts                   контракт данных API
  src/App.tsx                    навигация и связывание экранов
  src/main.tsx                   подключение React к HTML
  src/styles.css                 внешний вид и адаптивный layout
```

`domain` не зависит от HTTP, PostgreSQL и React. Интерфейсы определены в местах использования: `services`, `transport/http`, `pkg/runner`. Конкретные реализации связываются в `internal/app`. Учётные записи сохраняются в БД, симуляции защищены mutex и хранятся в памяти.

Во frontend комментарии поясняют обращения к Go API и важные решения React. В backend прокомментированы правила маршрутизации, ручной «Ход», модельное время, эвакуация, статистика и разграничение доступа. Пояснения к JSON-настройкам находятся в `frontend/README.md`.

## Проверки

```powershell
cd backend
go test ./...
go test -race ./internal/domain ./internal/services ./internal/transport/http
go vet ./...
go build ./cmd/server ./cmd/migrate
cd ../frontend
npm ci
npm test
npm run build
```

Интеграционный тест PostgreSQL запускается только с `TEST_DATABASE_URL`. Он требует отдельную БД **elevators_test**, применяет и откатывает миграции и очищает её данные. Не указывай рабочую БД.

В основном Compose порт БД не опубликован. Для отдельной тестовой базы удобно использовать:

```powershell
docker run -d --name elevators-test-postgres -e POSTGRES_USER=elevators -e POSTGRES_PASSWORD=test-password -e POSTGRES_DB=elevators_test -p 127.0.0.1:15432:5432 postgres:17-alpine
$env:TEST_DATABASE_URL = 'postgres://elevators:test-password@localhost:15432/elevators_test?sslmode=disable'
cd backend
go test ./internal/repository -run TestPostgresAccountsAndGoose -v
```

После теста останови тестовый контейнер через `docker stop elevators-test-postgres`.

## Локальная разработка

Для backend нужны PostgreSQL и применённые миграции:

```powershell
cd backend
$env:DATABASE_URL = 'postgres://elevators:YOUR_PASSWORD@localhost:5432/elevators?sslmode=disable'
go run ./cmd/migrate up
go run ./cmd/server
```

В другом терминале:

```powershell
cd frontend
npm ci
npm run dev
```

Открой `http://localhost:5173`. Vite проксирует `/api` и WebSocket в `localhost:8080`. Собранный frontend использует те же относительные URL через nginx.

Детали модели: [docs/model.md](docs/model.md). API: [docs/api.md](docs/api.md). Диаграммы находятся в `docs/diagrams`.
