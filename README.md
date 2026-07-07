# Domain List Manager

Управление источниками доменных списков с веб-интерфейсом, планировщиком обновлений и историей изменений.

## Быстрый запуск на Proxmox

Одной командой на Proxmox-хосте (от root):

```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/Edikskrim/domain-list-manager/main/install.sh)"
```

Скрипт автоматически:
- Создаёт LXC-контейнер с Debian 12
- Устанавливает Docker
- Разворачивает сервис через `docker compose`
- Показывает IP-адрес и URL сервиса

## Ручная установка (docker compose)

```bash
# 1. Клонируй репозиторий
git clone https://github.com/Edikskrim/domain-list-manager.git
cd domain-list-manager

# 2. Создай .env из примера
cp .env.example .env
# Отредактируй .env — укажи свой логин/пароль

# 3. Запусти
docker compose pull
docker compose up -d
```

Сервис доступен на `http://localhost:8080`

## Переменные окружения

| Переменная | По умолчанию | Описание |
|---|---|---|
| `SERVER_HOST` | `0.0.0.0` | Адрес привязки веб-сервера |
| `SERVER_PORT` | `8080` | Порт веб-сервера |
| `DB_PATH` | `data/domains.db` | Путь к БД SQLite внутри контейнера |
| `AUTH_USERNAME` | `admin` | Имя пользователя для входа |
| `AUTH_PASSWORD` | `admin` | Пароль для входа (смените!) |
| `FETCHER_TIMEOUT` | `30` | Таймаут HTTP-запросов (сек) |
| `FETCHER_MAX_RETRIES` | `3` | Макс. число повторных попыток |
| `FETCHER_MAX_BODY_SIZE` | `52428800` | Макс. размер тела ответа (байт, 50MB) |
| `FETCHER_MAX_REDIRECTS` | `10` | Макс. число редиректов |
| `BUILDER_SNAPSHOT_COUNT` | `10` | Сколько снимков истории хранить |
| `BUILDER_OUTPUT_PATH` | `output/domains.lst` | Путь к выходному файлу внутри контейнера |
