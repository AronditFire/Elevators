# HTTP API

Все адреса начинаются с `/api`. Авторизация — HttpOnly cookie `elevators_session`, SameSite Strict, срок 24 часа. В БД сохраняется SHA-256 токена, пароль — bcrypt. Роль и блокировка читаются при каждом запросе и каждом WebSocket-снимке. Браузерные изменяющие запросы проверяются по Origin. Авторизация и регистрация ограничены 20 попытками в минуту на адрес; при использовании nginx лимит общий для клиентов за одним proxy.

Ошибки возвращаются как `{"error":"сообщение"}`: 400 — неверный ввод/команда, 401 — авторизация, 403 — права, 404 — нет объекта, 409 — существующий логин, 429 — частые попытки, 500 — внутренняя ошибка.

| Метод | Адрес | Назначение |
|---|---|---|
| GET | `/health` | Готовность PostgreSQL и HTTP-сервера |
| POST | `/auth/register` | `{username,password}`, регистрация user |
| POST | `/auth/login` | `{username,password}`, вход и cookie |
| POST | `/auth/logout` | Выход и удаление токена |
| GET | `/auth/me` | Текущий пользователь |
| GET | `/users` | Все аккаунты, только admin |
| PATCH | `/users/{id}` | `{role,disabled}`, только admin |
| GET | `/sessions` | Доступные сеансы |
| POST | `/sessions` | `{config,scenario}`, создание своего сеанса |
| GET | `/sessions/{id}` | Полный снимок |
| PUT | `/sessions/{id}` | `{config,scenario}`, настройка до старта |
| DELETE | `/sessions/{id}` | Удаление готового/остановленного сеанса |
| POST | `/sessions/{id}/commands` | Команда модели |
| GET / PUT | `/sessions/{id}/config.yaml` | Экспорт / импорт YAML конфигурации |
| GET / PUT | `/sessions/{id}/scenario.yaml` | Экспорт / импорт YAML сценария |
| GET | `/sessions/{id}/report.csv` | Отчёт остановленного сеанса |
| GET | `/sessions/{id}/stream` | WebSocket со снимками каждые 200 мс |

У каждого сеанса проверяются права владельца либо admin на все чтения, команды, файлы и WebSocket. Пользователь не может указать другого владельца при создании.

```json
{"type":"start"}
```
```json
{"type":"pause"}
```
```json
{"type":"stop"}
```
```json
{"type":"speed","speed":2}
```
```json
{"type":"go","elevator_id":1}
```
```json
{"type":"fire","active":true}
```
```json
{"type":"people","people":{"from":1,"to":10,"count":3}}
```

Конфигурация и сценарий имеют одинаковые имена полей в JSON и YAML. Примеры — `examples/`. REST-команды возвращают свежий снимок сеанса. WebSocket передаёт только состояние; изменяющие команды отправляются через HTTP API.
