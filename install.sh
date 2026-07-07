#!/bin/bash
# ============================================================
# install.sh — Deployment script for Proxmox
# ============================================================
# Запускать НА ПРОКСМОКС ХОСТЕ (не в контейнере):
#   bash -c "$(curl -fsSL https://raw.githubusercontent.com/Edikskrim/domain-list-manager/main/install.sh)"
# ============================================================

set -euo pipefail

# --------------- Настройки ---------------
CONTAINER_HOSTNAME="domain-list-manager"
DEBIAN_TEMPLATE="debian-12-standard"
VM_DISK_SIZE="8G"
VM_CPU="2"
VM_MEMORY="2048"
VM_BRIDGE="vmbr0"
FRONT_PORT="8080"
GHCR_IMAGE="ghcr.io/edikskrim/domain-list-manager"
RAW_URL="https://raw.githubusercontent.com/Edikskrim/domain-list-manager/main"

# --------------- Цвета ---------------
GREEN='\033[0;32m'
BOLD_GREEN='\033[1;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# --------------- Проверки ---------------
echo -e "${YELLOW}Проверка окружения...${NC}"

# Проверка: Proxmox
if ! command -v pveversion &>/dev/null; then
    echo -e "${RED}Ошибка: pveversion не найден. Скрипт нужно запускать на Proxmox-хосте.${NC}"
    exit 1
fi

# Проверка: root
if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}Ошибка: скрипт должен запускаться от root (sudo).${NC}"
    exit 1
fi

echo -e "${GREEN}OK: Proxmox хост найден, запущен от root.${NC}"

# --------------- Определение VMID ---------------
echo -e "${YELLOW}Определяю свободный VMID...${NC}"
VMID=$(pvesh get /cluster/nextid 2>/dev/null)
if [[ -z "$VMID" ]]; then
    # fallback: найди свободный
    VMID=99
    while pct status "$VMID" &>/dev/null; do
        VMID=$((VMID + 1))
    done
fi
echo -e "${GREEN}Используемый VMID: ${VMID}${NC}"

# --------------- Проверка/скачивание шаблона ---------------
echo -e "${YELLOW}Проверка шаблона Debian 12...${NC}"
if ! pveam list local | grep -q "$DEBIAN_TEMPLATE"; then
    echo -e "${YELLOW}Скачиваю шаблон $DEBIAN_TEMPLATE...${NC}"
    pveam download local "$DEBIAN_TEMPLATE"
    echo -e "${GREEN}Шаблон скачан.${NC}"
else
    echo -e "${GREEN}Шаблон уже присутствует.${NC}"
fi

# --------------- Создание LXC ---------------
echo -e "${YELLOW}Создаю LXC контейнер...${NC}"
ARCH="$(dpkg --print-architecture)"

pct create "$VMID" \
    "local:vztmpl/${DEBIAN_TEMPLATE}.tar.zst" \
    -features keyctl=1,nesting=1 \
    -arch "$ARCH" \
    -name "$CONTAINER_HOSTNAME" \
    -unprivileged 1 \
    -cores "$VM_CPU" \
    -memory "$VM_MEMORY" \
    -swap 512 \
    -ostype debian \
    -rootfs "local-lvm:${VM_DISK_SIZE}" \
    -net0 "name=eth0,bridge=${VM_BRIDGE},ip=dhcp" \
    -onboot 1

echo -e "${GREEN}Контейнер создан.${NC}"

# --------------- Запуск контейнера ---------------
echo -e "${YELLOW}Запускаю контейнер...${NC}"
pct start "$VMID"

# Ждём поднятия
echo -e "${YELLOW}Ожидаю поднятия контейнера...${NC}"
MAX_WAIT=30
for i in $(seq 1 $MAX_WAIT); do
    if pct status "$VMID" 2>/dev/null | grep -q "running"; then
        echo -e "${GREEN}Контейнер запущен!${NC}"
        break
    fi
    if [[ $i -eq $MAX_WAIT ]]; then
        echo -e "${RED}Контейнер не запустился за ${MAX_WAIT} секунд.${NC}"
        exit 1
    fi
    sleep 2
done

# --------------- Установка Docker ---------------
echo -e "${YELLOW}Устанавливаю Docker внутри контейнера...${NC}"
DOCKER_EXIT=0
pct exec "$VMID" -- sh -c '
    set -euo pipefail
    apt-get update -qq
    apt-get install -y -qq ca-certificates curl gnupg >/dev/null 2>&1
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg >/dev/null 2>&1
    chmod a+r /etc/apt/keyrings/docker.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian bookworm stable" > /etc/apt/sources.list.d/docker.list
    apt-get update -qq
    apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-compose-plugin >/dev/null 2>&1
    systemctl enable --now docker
    sleep 2
    docker info >/dev/null 2>&1
' || DOCKER_EXIT=$?

if [[ $DOCKER_EXIT -ne 0 ]]; then
    echo -e "${RED}Ошибка при установке Docker.${NC}"
    exit 1
fi
echo -e "${GREEN}Docker установлен!${NC}"

# --------------- Скачивание файлов для deployment ---------------
echo -e "${YELLOW}Подготовка файлов...${NC}"
TMP_DIR=$(mktemp -d)
curl -fsSL "${RAW_URL}/docker-compose.yml" -o "${TMP_DIR}/docker-compose.yml"
if [[ -f ".env" ]]; then
    cp ".env" "${TMP_DIR}/.env"
else
    curl -fsSL "${RAW_URL}/.env.example" -o "${TMP_DIR}/.env"
fi

# --------------- Развертывание сервиса ---------------
echo -e "${YELLOW}Разворачиваю сервис...${NC}"
pct exec "$VMID" -- mkdir -p /opt/domain-list-manager

# Копируем файлы внутрь контейнера
pct push "$VMID" "${TMP_DIR}/docker-compose.yml" "/opt/domain-list-manager/docker-compose.yml"
pct push "$VMID" "${TMP_DIR}/.env" "/opt/domain-list-manager/.env"

# Очистка
rm -rf "$TMP_DIR"

COMPOSE_EXIT=0
pct exec "$VMID" -- sh -c '
    cd /opt/domain-list-manager
    docker compose pull
    docker compose up -d
' || COMPOSE_EXIT=$?

if [[ $COMPOSE_EXIT -ne 0 ]]; then
    echo -e "${RED}Ошибка при запуске docker compose.${NC}"
    exit 1
fi
echo -e "${GREEN}Сервис запущен!${NC}"

# --------------- Получение IP ---------------
echo -e "${YELLOW}Ожидаю IP-адрес (DHCP)...${NC}"
CONTAINER_IP=""
for i in $(seq 1 10); do
    ip_output=$(pct exec "$VMID" -- ip -4 addr show eth0 2>/dev/null | grep -oP 'inet \K[0-9.]+') || true
    if [[ -n "$ip_output" ]]; then
        CONTAINER_IP="$ip_output"
        break
    fi
    sleep 2
done

if [[ -z "$CONTAINER_IP" ]]; then
    echo -e "${YELLOW}Не удалось получить IP автоматически. Проверьте вручную:${NC}"
    pct exec "$VMID" -- ip -4 addr show eth0
    CONTAINER_IP="не определён"
else
    echo -e "${GREEN}IP контейнера: ${CONTAINER_IP}${NC}"
fi

# --------------- Финальный вывод ---------------
echo ""
echo -e "${BOLD_GREEN}╔══════════════════════════════════════════╗${NC}"
echo -e "${BOLD_GREEN}║   ${CONTAINER_HOSTNAME} успешно развернут!       ║${NC}"
echo -e "${BOLD_GREEN}╚══════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BOLD_GREEN}Сервис доступен по адресу: http://${CONTAINER_IP}:${FRONT_PORT}${NC}"
echo ""
echo -e "${YELLOW}Управление:${NC}"
echo -e "  pct exec ${VMID} -- docker compose -f /opt/domain-list-manager/docker-compose.yml logs"
echo -e "  pct exec ${VMID} -- docker compose -f /opt/domain-list-manager/docker-compose.yml down"
echo -e "  pct exec ${VMID} -- docker compose -f /opt/domain-list-manager/docker-compose.yml restart"
echo ""
