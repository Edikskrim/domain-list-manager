# Roadmap

## Phase 1 — Project Skeleton
- Создать структуру каталогов.
- Настроить Go-модуль.
- Настроить конфигурацию.
- Добавить запуск HTTP-сервера.
- Проверить, что проект компилируется.

---

## Phase 2 — Database
- Подключить SQLite.
- Создать миграции.
- Реализовать инициализацию БД.
- Реализовать Repository.

---

## Phase 3 — Authentication
- [x] Простая авторизация
- [x] Login / Logout
- [x] Middleware
- [x] Защита админки

---

## Phase 4 — Settings
- Таблица настроек.
- CRUD настроек.
- Загрузка настроек при старте.

---

## Phase 5 — Sources
- CRUD источников.
- Включение / выключение.
- Проверка URL.
- Просмотр информации об источнике.

---

## Phase 6 — Custom Domains
- CRUD пользовательских доменов.
- Массовое добавление.
- Импорт TXT.
- Экспорт TXT.

---

## Phase 7 — Fetcher
- HTTP Client.
- Скачать источник.
- Timeout.
- Retry.
- ETag.
- Last-Modified.

---

## Phase 8 — Parsers
- Raw parser.
- Hosts parser.
- Dnsmasq parser.
- Regex parser.
- Auto parser.

---

## Phase 9 — Builder
- Объединение источников.
- Нормализация.
- Удаление дубликатов.
- Сортировка.
- Генерация domains.lst.

---

## Phase 10 — History
- Snapshot.
- Хранение последних N версий.
- Очистка старых.

---

## Phase 11 — Diff
- Added domains.
- Removed domains.
- JSON diff.

---

## Phase 12 — Intersections
- Поиск пересечений.
- JSON отчёт.

---

## Phase 13 — Dashboard
- Статистика.
- Последняя сборка.
- Ошибки.

---

## Phase 14 — Diagnostics
- Пересечения.
- Ошибки парсинга.
- Невалидные домены.

---

## Phase 15 — REST API
- API для всех операций админки.

---

## Phase 16 — Docker
- Dockerfile.
- docker-compose.
- Volume.
- Healthcheck.

---

## Phase 17 — Testing
- Unit tests.
- Builder.
- Parser.
- Fetcher.

---

## Phase 18 — Polish
- Оптимизация.
- Документация.
- Финальная проверка.