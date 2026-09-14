#!/usr/bin/env bash
set -Eeuo pipefail

APP_NAME="ramdns"
APP_USER="ubuntu"
APP_GROUP="ubuntu"
APP_DIR="/opt/ramdns"
CONFIG_DIR="/etc/ramdns"
TLS_DIR="${CONFIG_DIR}/tls"
SYSTEMD_UNIT="/etc/systemd/system/${APP_NAME}.service"
DROPIN_DIR="/etc/systemd/system/${APP_NAME}.service.d"
BINARY="${APP_DIR}/${APP_NAME}"

log() {
    printf '[ramdns] %s\n' "$*"
}

die() {
    printf '[ramdns] ERROR: %s\n' "$*" >&2
    exit 1
}

require_root() {
    [[ "${EUID}" -eq 0 ]] || die "run this script as root"
}

require_ubuntu() {
    command -v apt-get >/dev/null 2>&1 || die "apt-get not found; supported target is Ubuntu/Debian-like systems"
}

install_packages() {
    log "installing build/runtime prerequisites"

    export DEBIAN_FRONTEND=noninteractive

    apt-get update
    apt-get install -y \
        ca-certificates \
        curl \
        git \
        golang \
        openssl
}

ensure_user() {
    if ! id "${APP_USER}" >/dev/null 2>&1; then
        die "required user '${APP_USER}' does not exist"
    fi

    if ! getent group "${APP_GROUP}" >/dev/null 2>&1; then
        die "required group '${APP_GROUP}' does not exist"
    fi
}

prepare_directories() {
    log "preparing directories"

    install -d \
        -o "${APP_USER}" \
        -g "${APP_GROUP}" \
        -m 0750 \
        "${APP_DIR}"

    install -d \
        -o root \
        -g "${APP_GROUP}" \
        -m 0750 \
        "${CONFIG_DIR}"

    install -d \
        -o "${APP_USER}" \
        -g "${APP_GROUP}" \
        -m 0700 \
        "${TLS_DIR}"

    install -d \
        -o root \
        -g root \
        -m 0755 \
        "${DROPIN_DIR}"
}

build_binary() {
    log "building RAMDNS"

    cd "${APP_DIR}"

    [[ -f go.mod ]] || die "go.mod not found in ${APP_DIR}"

    go build -trimpath -o "${BINARY}" ./cmd/ramdns

    chown "${APP_USER}:${APP_GROUP}" "${BINARY}"
    chmod 0755 "${BINARY}"
}

ensure_management_secret() {
    local env_file="${CONFIG_DIR}/management.env"

    if [[ -f "${env_file}" ]] && \
       grep -q '^RAMDNS_MANAGEMENT_TOKEN=' "${env_file}"; then
        log "management token already exists"
        chmod 0600 "${env_file}"
        chown root:root "${env_file}"
        return
    fi

    log "generating management token"

    local token
    token="$(openssl rand -hex 32)"

    umask 0077
    printf 'RAMDNS_MANAGEMENT_TOKEN=%s\n' "${token}" > "${env_file}"

    chown root:root "${env_file}"
    chmod 0600 "${env_file}"
}

install_systemd_unit() {
    log "installing systemd unit"

    cat > "${SYSTEMD_UNIT}" <<UNIT
[Unit]
Description=RAMDNS Personal DNS Resolver
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${APP_USER}
Group=${APP_GROUP}
WorkingDirectory=${APP_DIR}
ExecStart=${BINARY}
Restart=always
RestartSec=2

NoNewPrivileges=true
PrivateTmp=true
PrivateDevices=true

ProtectSystem=strict
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectKernelLogs=true
ProtectControlGroups=true
ProtectClock=true

ProtectProc=invisible
ProcSubset=pid

RestrictSUIDSGID=true
LockPersonality=true
RestrictRealtime=true
RestrictNamespaces=true

RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6
SystemCallArchitectures=native

UMask=0027
LimitNOFILE=65536
TasksMax=4096

AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

EnvironmentFile=${CONFIG_DIR}/management.env

[Install]
WantedBy=multi-user.target
UNIT

    chmod 0644 "${SYSTEMD_UNIT}"
}

install_adlist_dropin() {
    log "installing adlist configuration"

    cat > "${DROPIN_DIR}/adlist.conf" <<'UNIT'
[Service]
Environment="RAMDNS_ADLIST_URL=https://big.oisd.nl/,https://hagezi-mirror.dnsbunker.org/adblock/pro.txt,https://hagezi-mirror.dnsbunker.org/adblock/tif.txt"
UNIT

    chmod 0644 "${DROPIN_DIR}/adlist.conf"
}

install_capability_dropin() {
    log "installing capability configuration"

    cat > "${DROPIN_DIR}/capabilities.conf" <<'UNIT'
[Service]
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
NoNewPrivileges=true
UNIT

    chmod 0644 "${DROPIN_DIR}/capabilities.conf"
}

reload_systemd() {
    log "reloading systemd"

    systemctl daemon-reload
    systemctl enable "${APP_NAME}.service"
}

start_service() {
    log "starting RAMDNS"

    systemctl restart "${APP_NAME}.service"

    sleep 2

    systemctl is-active --quiet "${APP_NAME}.service" || {
        systemctl --no-pager --full status "${APP_NAME}.service" || true
        journalctl -u "${APP_NAME}.service" -n 80 --no-pager || true
        die "RAMDNS failed to start"
    }
}

verify() {
    log "verifying service"

    systemctl is-enabled "${APP_NAME}.service"
    systemctl is-active "${APP_NAME}.service"

    ss -lntup | grep -E ':(53|443|853)\b' || true

    log "installation complete"
}

main() {
    require_root
    require_ubuntu
    ensure_user
    install_packages
    prepare_directories
    build_binary
    ensure_management_secret
    install_systemd_unit
    install_adlist_dropin
    install_capability_dropin
    reload_systemd
    start_service
    verify
}

main "$@"
