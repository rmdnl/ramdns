# RAMDNS 🚀

DNS server milik sendiri, no government involved 🤪. Cepat, private, bisa ngeblok iklan, dan nggak perlu dashboard admin yang nongkrong terbuka di Internet.

RAMDNS adalah recursive DNS resolver berbasis Go yang dibuat untuk VPS kecil dengan fokus pada performa, keamanan, filtering, dan operasional yang sederhana.

Filosofinya:

> query masuk → rate limit → filter → cache → kalau perlu upstream → jawab

Tidak ada public admin API. Tidak ada port management yang perlu dibuka ke Internet.

---

## ✨ Fitur

| Fitur | Status |
|---|---|
| DNS UDP `:53` | 🟢 |
| DNS TCP `:53` | 🟢 |
| DNS-over-TLS `:853` | 🟢 |
| DNS-over-HTTPS `:443` | 🟢 |
| DNSSEC | 🟢 |
| Persistent DoT upstream | 🟢 |
| Parallel upstream | 🟢 |
| DNS response cache | 🟢 |
| SingleFlight | 🟢 |
| Multi-source adlist | 🟢 |
| Hosts / Adblock / plain domain parser | 🟢 |
| Concurrent adlist download | 🟢 |
| Atomic adlist reload | 🟢 |
| ETag / Last-Modified | 🟢 |
| Periodic adlist update | 🟢 |
| Per-IP rate limiting | 🟢 |
| TLS 1.3 minimum | 🟢 |
| Automatic TLS renewal | 🟢 |
| systemd hardening | 🟢 |
| Public control API | 🔴 Sengaja tidak ada |
| Health-aware upstream routing | 🚧 |
| Prometheus / observability | 🚧 |
| Private management API | 🚧 |
| Web dashboard | 🚧 |

---

# 🏗️ Arsitektur

```text
                         Internet
                            │
              ┌─────────────┼─────────────┐
              │             │             │
           UDP :53       TCP :53       Encrypted DNS
                                          │
                              ┌───────────┴───────────┐
                              │                       │
                           DoT :853                DoH :443
                              │                       │
                              └───────────┬───────────┘
                                          │
                                   ┌──────▼──────┐
                                   │   RAMDNS    │
                                   └──────┬──────┘
                                          │
                              ┌───────────▼───────────┐
                              │     Rate Limiter      │
                              └───────────┬───────────┘
                                          │
                              ┌───────────▼───────────┐
                              │      Adlist Filter     │
                              └───────────┬───────────┘
                                          │
                              ┌───────────▼───────────┐
                              │        Cache           │
                              └───────────┬───────────┘
                                          │
                              ┌───────────▼───────────┐
                              │      SingleFlight      │
                              └───────────┬───────────┘
                                          │
                              ┌───────────▼───────────┐
                              │   Parallel DoT Upstream│
                              └───────────┬───────────┘
                                          │
                         ┌────────────────┴────────────────┐
                         │                                 │
                  Cloudflare DoT                    Google DoT
                  1.1.1.1:853                      8.8.8.8:853
```

---

# 🖥️ Requirements

Contoh deployment RAMDNS saat ini menggunakan:

```text
OS       : Ubuntu
CPU      : 2 vCPU
RAM      : ~2 GB
Swap     : 0
```

RAMDNS membutuhkan:

- root/sudo saat instalasi
- Go untuk build
- domain/subdomain untuk DoT dan DoH
- Cloudflare DNS untuk automatic TLS DNS-01
- public IPv4/IPv6 jika ingin dijadikan public resolver

---

# 📁 Project Structure

```text
ramdns/
├── cmd/
│   └── ramdns/
│       └── main.go
├── internal/
│   ├── adlist/
│   ├── cache/
│   ├── dns/
│   ├── doh/
│   ├── filter/
│   ├── metrics/
│   ├── ratelimit/
│   └── upstream/
├── deploy/
│   ├── certbot/
│   │   └── ramdns-cert.sh
│   └── systemd/
│       ├── ramdns.service
│       ├── adlist.conf
│       ├── capabilities.conf
│       ├── hardening.conf
│       └── upstream.conf
├── go.mod
├── go.sum
├── .gitignore
└── README.md
```

---

# 🚀 Rebuild dari VPS Ubuntu Fresh

## 1. Update system

```bash
sudo apt update
sudo apt upgrade -y
```

Install dependency:

```bash
sudo apt install -y \
  git \
  curl \
  ca-certificates \
  build-essential \
  dnsutils \
  openssl \
  ufw \
  certbot
```

---

## 2. Install Go

Cek:

```bash
go version
```

Gunakan versi Go yang memenuhi directive pada `go.mod`.

---

## 3. Clone repository

```bash
sudo mkdir -p /opt
sudo git clone https://github.com/rmdnl/ramdns.git /opt/ramdns
sudo chown -R ubuntu:ubuntu /opt/ramdns

cd /opt/ramdns
go mod download
```

Test:

```bash
go test ./...
```

Build:

```bash
go build -o ramdns ./cmd/ramdns
```

---

# 👤 Runtime User

RAMDNS dijalankan sebagai user non-root.

Deployment saat ini:

```text
User  : ubuntu
Group : ubuntu
```

Binding ke port rendah dilakukan menggunakan:

```ini
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
```

---

# 🌐 DNS Domain Setup

Deployment saat ini menggunakan:

```text
dot.ramdns.my.id
doh.ramdns.my.id
```

Buat record DNS di Cloudflare:

```text
dot.ramdns.my.id  A     <SERVER_IPV4>
doh.ramdns.my.id  A     <SERVER_IPV4>
```

Jika menggunakan IPv6:

```text
dot.ramdns.my.id  AAAA  <SERVER_IPV6>
doh.ramdns.my.id  AAAA  <SERVER_IPV6>
```

Pastikan hostname dapat mencapai VPS.

---

# 🔥 Firewall

Port yang diperlukan:

```text
22/tcp
53/tcp
53/udp
853/tcp
443/tcp
```

Contoh:

```bash
sudo ufw default deny incoming
sudo ufw default deny outgoing

sudo ufw allow 22/tcp
sudo ufw allow 53/tcp
sudo ufw allow 53/udp
sudo ufw allow 853/tcp
sudo ufw allow 443/tcp

sudo ufw enable
sudo ufw status verbose
```

Port `80/tcp` tidak diperlukan untuk TLS karena certificate menggunakan DNS-01.

Port `8080` juga tidak diperlukan.

> Pastikan SSH sudah diizinkan sebelum mengaktifkan UFW.

---

# 🔐 Automatic TLS Certificate Renewal

RAMDNS menggunakan:

```text
Let's Encrypt
Certbot
Cloudflare DNS-01
```

Hostname certificate:

```text
dot.ramdns.my.id
doh.ramdns.my.id
```

## 1. Install Cloudflare plugin

```bash
sudo apt install -y python3-certbot-dns-cloudflare
```

Cek:

```bash
certbot plugins
```

Pastikan `dns-cloudflare` tersedia.

---

## 2. Buat Cloudflare API Token

Buat API Token khusus RAMDNS.

Gunakan permission minimum:

```text
Zone
└── DNS
    └── Edit
```

Batasi token hanya untuk zone:

```text
ramdns.my.id
```

Jangan menggunakan Global API Key jika API Token sudah cukup.

---

## 3. Simpan credentials

```bash
sudo install -d -m 700 /root/.secrets/certbot
sudo nano /root/.secrets/certbot/cloudflare.ini
```

Isi:

```ini
dns_cloudflare_api_token = YOUR_CLOUDFLARE_API_TOKEN
```

Permission:

```bash
sudo chmod 600 /root/.secrets/certbot/cloudflare.ini
```

Verifikasi:

```bash
sudo ls -l /root/.secrets/certbot/cloudflare.ini
```

Credential ini tidak boleh masuk Git.

---

## 4. Issue certificate pertama kali

```bash
sudo certbot certonly \
  --dns-cloudflare \
  --dns-cloudflare-credentials /root/.secrets/certbot/cloudflare.ini \
  --dns-cloudflare-propagation-seconds 30 \
  -d dot.ramdns.my.id \
  -d doh.ramdns.my.id
```

Cek:

```bash
sudo certbot certificates
```

Certificate:

```text
/etc/letsencrypt/live/dot.ramdns.my.id/
```

---

# 🔗 Deploy Certificate ke RAMDNS

RAMDNS menggunakan:

```text
/etc/ramdns/tls/fullchain.pem
/etc/ramdns/tls/privkey.pem
```

Buat directory:

```bash
sudo install -d -o ubuntu -g ubuntu -m 700 /etc/ramdns/tls
```

Repository menyediakan:

```text
deploy/certbot/ramdns-cert.sh
```

Isi:

```bash
#!/bin/bash
set -euo pipefail

CERT_DIR="/etc/letsencrypt/live/dot.ramdns.my.id"
TLS_DIR="/etc/ramdns/tls"

install -o ubuntu -g ubuntu -m 0644 \
  "$CERT_DIR/fullchain.pem" \
  "$TLS_DIR/fullchain.pem"

install -o ubuntu -g ubuntu -m 0600 \
  "$CERT_DIR/privkey.pem" \
  "$TLS_DIR/privkey.pem"

systemctl restart ramdns
```

Install hook:

```bash
sudo install -o root -g root -m 0700 \
  deploy/certbot/ramdns-cert.sh \
  /etc/letsencrypt/renewal-hooks/deploy/ramdns-cert.sh
```

---

# ⏰ Automatic Renewal

Certbot menggunakan systemd timer.

Cek:

```bash
systemctl status certbot.timer --no-pager
```

Enable:

```bash
sudo systemctl enable --now certbot.timer
```

Cek jadwal:

```bash
systemctl list-timers certbot.timer --no-pager
```

Alurnya:

```text
Certbot timer
     ↓
certbot renew
     ↓
certificate diperbarui
     ↓
deploy hook
     ↓
copy certificate
     ↓
restart RAMDNS
```

Certificate tidak diperbarui setiap timer berjalan. Certbot hanya melakukan renewal ketika certificate sudah masuk renewal window.

---

# 🧪 Test Renewal

Gunakan:

```bash
sudo certbot renew --dry-run
```

Jika gagal:

```bash
sudo journalctl -u certbot.service --no-pager
```

dan:

```bash
sudo tail -n 100 /var/log/letsencrypt/letsencrypt.log
```

---

# 🔍 Verify TLS

DoT:

```bash
openssl s_client \
  -connect dot.ramdns.my.id:853 \
  -servername dot.ramdns.my.id \
  -tls1_3 </dev/null 2>/dev/null \
  | openssl x509 -noout -subject -issuer -dates -ext subjectAltName
```

DoH:

```bash
openssl s_client \
  -connect doh.ramdns.my.id:443 \
  -servername doh.ramdns.my.id \
  -tls1_3 </dev/null 2>/dev/null \
  | openssl x509 -noout -subject -issuer -dates -ext subjectAltName
```

SAN harus mencakup:

```text
DNS:dot.ramdns.my.id
DNS:doh.ramdns.my.id
```

---

# ⚙️ Install systemd

Repository menyediakan:

```text
deploy/systemd/ramdns.service
deploy/systemd/adlist.conf
deploy/systemd/capabilities.conf
deploy/systemd/hardening.conf
deploy/systemd/upstream.conf
```

Install:

```bash
sudo install -D -o root -g root -m 0644 \
  deploy/systemd/ramdns.service \
  /etc/systemd/system/ramdns.service

sudo install -D -o root -g root -m 0644 \
  deploy/systemd/adlist.conf \
  /etc/systemd/system/ramdns.service.d/adlist.conf

sudo install -D -o root -g root -m 0644 \
  deploy/systemd/capabilities.conf \
  /etc/systemd/system/ramdns.service.d/capabilities.conf

sudo install -D -o root -g root -m 0644 \
  deploy/systemd/hardening.conf \
  /etc/systemd/system/ramdns.service.d/hardening.conf

sudo install -D -o root -g root -m 0644 \
  deploy/systemd/upstream.conf \
  /etc/systemd/system/ramdns.service.d/upstream.conf
```

Reload:

```bash
sudo systemctl daemon-reload
sudo systemctl enable ramdns
sudo systemctl restart ramdns
```

Cek:

```bash
systemctl status ramdns --no-pager
```

---

# 🔐 systemd Hardening

RAMDNS menggunakan:

```ini
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictSUIDSGID=true
LockPersonality=true
RestrictRealtime=true
RestrictNamespaces=true
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6
SystemCallArchitectures=native
LimitNOFILE=65536
TasksMax=4096
```

Service berjalan sebagai non-root dan hanya mendapat capability minimum untuk bind low port.

---

# 🔐 Upstream DNS

Default:

```text
1.1.1.1:853 | cloudflare-dns.com
8.8.8.8:853 | dns.google
```

Konfigurasi:

```ini
[Service]
Environment="RAMDNS_UPSTREAMS=1.1.1.1:853|cloudflare-dns.com,8.8.8.8:853|dns.google"
```

Upstream menggunakan DoT dengan certificate verification aktif dan minimum TLS 1.3.

---

# 🔗 Persistent DoT

Koneksi TCP/TLS upstream dipertahankan dan digunakan kembali.

```text
TCP connect
    ↓
TLS 1.3 handshake
    ↓
DNS query
    ↓
connection tetap hidup
    ↓
query berikutnya
```

Idle connection akan ditutup setelah tidak digunakan dalam periode tertentu.

---

# ⚡ Parallel Upstream

RAMDNS menjalankan upstream secara parallel:

```text
                 ┌── Cloudflare
Query ───────────┤
                 └── Google
                       │
                       ▼
                first successful response
```

Tujuannya mengurangi tail latency ketika salah satu upstream lambat.

---

# 💾 Cache

Default:

```text
50,000 entries
```

Cache:

- TTL-aware
- bounded
- response di-copy sebelum dikembalikan
- expired entry dibuang
- negative response dapat dicache jika TTL valid

Cache key mempertimbangkan:

```text
QNAME
QTYPE
QCLASS
DO bit
```

---

# 🧩 SingleFlight

Query identik yang datang bersamaan dapat digabung:

```text
100 clients
     │
     ▼
SingleFlight
     │
     ▼
1 upstream query
     │
     ▼
shared response
```

---

# 🧱 Adlist

Parser mendukung:

### Hosts

```text
0.0.0.0 ads.example.com
127.0.0.1 tracker.example.com
:: ads.example.com
```

### Adblock

```text
||ads.example.com^
||tracker.example.com^
```

### Plain domain

```text
ads.example.com
tracker.example.com
```

---

# 📚 Adlist Sources

Deployment saat ini:

```text
https://big.oisd.nl/
https://hagezi-mirror.dnsbunker.org/adblock/pro.txt
https://hagezi-mirror.dnsbunker.org/adblock/tif.txt
```

Konfigurasi:

```ini
[Service]
Environment="RAMDNS_ADLIST_URL=https://big.oisd.nl/,https://hagezi-mirror.dnsbunker.org/adblock/pro.txt,https://hagezi-mirror.dnsbunker.org/adblock/tif.txt"
```

Jumlah rules bersifat dinamis.

---

# 🔄 Adlist Update

Saat startup:

```text
download
  ↓
parse
  ↓
compile
  ↓
combine
  ↓
atomic replace
```

Update dilakukan secara periodik.

Jika satu source gagal:

```text
source gagal
    ↓
snapshot lama tetap aktif
```

Downloader mendukung:

```text
HTTPS only
ETag
Last-Modified
If-None-Match
If-Modified-Since
304 Not Modified
response size limit
timeout
```

---

# 🚦 Rate Limiting

Default:

```text
50 requests/second/IP
burst: 100
tracked IPs: 20,000
```

Rate limiting bukan pengganti DDoS protection provider.

---

# 🌐 DNS-over-HTTPS

Endpoint:

```text
https://doh.ramdns.my.id/dns-query
```

DoH menggunakan DNS wire format melalui HTTP POST.

Contoh:

```bash
curl \
  -H 'content-type: application/dns-message' \
  --data-binary @query.bin \
  https://doh.ramdns.my.id/dns-query
```

---

# 🔐 DNS-over-TLS

Endpoint:

```text
dot.ramdns.my.id:853
```

Test:

```bash
openssl s_client \
  -connect dot.ramdns.my.id:853 \
  -servername dot.ramdns.my.id \
  -tls1_3
```

---

# 🛡️ DNSSEC

Test:

```bash
dig @127.0.0.1 cloudflare.com A +dnssec
```

Response tervalidasi akan memiliki:

```text
ad
```

---

# 🧪 Verification Checklist

Service:

```bash
systemctl is-active ramdns
```

Listener:

```bash
sudo ss -lntup | grep -E ':(53|853|443)\b'
```

DNS:

```bash
dig @127.0.0.1 example.com A +stats
```

DNSSEC:

```bash
dig @127.0.0.1 cloudflare.com A +dnssec
```

TLS:

```bash
openssl s_client \
  -connect dot.ramdns.my.id:853 \
  -servername dot.ramdns.my.id \
  -tls1_3
```

Adlist:

```bash
journalctl -u ramdns -n 50 --no-pager | grep adlist
```

Renewal:

```bash
sudo certbot renew --dry-run
```

---

# 📊 Benchmark

Benchmark internal setelah persistent DoT dan parallel upstream:

```text
500 random parallel queries

p50  = 0 ms
p90  = 4 ms
p95  = 4 ms
p99  = 8 ms
max  = 8 ms

SERVFAIL = 0
```

5,000 sequential:

```text
count = 5000
p50   = 4 ms
p95   = 4 ms
p99   = 4 ms
max   = 16 ms
```

5,000 query dengan 100 concurrent workers:

```text
count = 5000
p50   = 0 ms
p95   = 4 ms
p99   = 8 ms
max   = 16 ms

SERVFAIL = 0
```

Benchmark bergantung pada cache state, network, upstream, CPU, dan kondisi server.

---

# 📈 Metrics

Internal metrics mencakup:

```text
cache hits
cache misses
upstream queries
upstream errors
total queries
cache hit ratio
```

Prometheus-compatible observability masih berada di roadmap.

Jangan menganggap `/metrics` sebagai public endpoint saat ini.

---

# 🔒 Security

Jangan commit:

```text
*.pem
*.key
cloudflare.ini
API tokens
private keys
.env
```

Credential Cloudflare:

```text
/root/.secrets/certbot/cloudflare.ini
```

Private key:

```text
/etc/ramdns/tls/privkey.pem
```

tetap hanya berada di server.

---

# 🩺 Troubleshooting

## RAMDNS tidak start

```bash
systemctl status ramdns --no-pager
journalctl -u ramdns -n 100 --no-pager
```

## Port conflict

```bash
sudo ss -lntup | grep -E ':(53|853|443)\b'
```

## SERVFAIL

```bash
journalctl -u ramdns -n 100 --no-pager
```

Periksa konektivitas upstream.

## Certificate renewal gagal

```bash
sudo certbot certificates
sudo certbot renew --dry-run
sudo journalctl -u certbot.service --no-pager
sudo tail -n 100 /var/log/letsencrypt/letsencrypt.log
```

## Certificate baru tidak dipakai

```bash
sudo sed -n '1,200p' \
  /etc/letsencrypt/renewal-hooks/deploy/ramdns-cert.sh

sudo ls -l /etc/ramdns/tls/
systemctl status ramdns --no-pager
```

---

# 🔄 Rebuild Checklist

```text
[ ] Install Ubuntu
[ ] Update packages
[ ] Install Git / Go / Certbot / dependencies
[ ] Clone RAMDNS
[ ] go mod download
[ ] go test ./...
[ ] go build
[ ] Configure DNS records
[ ] Configure Cloudflare API Token
[ ] Install Cloudflare Certbot plugin
[ ] Issue Let's Encrypt certificate
[ ] Install RAMDNS TLS deploy hook
[ ] Install systemd service
[ ] Install systemd drop-ins
[ ] daemon-reload
[ ] enable/start RAMDNS
[ ] Configure UFW
[ ] Test DNS
[ ] Test DNSSEC
[ ] Test DoT
[ ] Test DoH
[ ] Test adlist
[ ] Test Certbot dry-run
[ ] Verify certbot.timer
[ ] Verify logs
```

---

# 🧭 Roadmap

## Core Resolver

- [x] UDP DNS
- [x] TCP DNS
- [x] Recursive resolution
- [x] DNS cache
- [x] SingleFlight
- [x] Parallel upstream
- [x] Persistent DoT

## Secure DNS

- [x] DNSSEC
- [x] DoT
- [x] DoH
- [x] TLS 1.3 minimum
- [x] Let's Encrypt
- [x] Cloudflare DNS-01
- [x] Automatic certificate renewal
- [x] Automatic deploy hook

## Filtering

- [x] Hosts parser
- [x] Adblock parser
- [x] Plain domain parser
- [x] Multi-source adlist
- [x] Concurrent downloads
- [x] Atomic reload
- [x] ETag / Last-Modified
- [x] Periodic update

## Security

- [x] Per-IP rate limiting
- [x] Non-root runtime
- [x] CAP_NET_BIND_SERVICE
- [x] systemd hardening
- [x] No public management API
- [x] No public port 8080

## Performance / Reliability

- [x] Persistent DoT
- [x] Parallel upstream
- [ ] Health-aware upstream routing
- [ ] Upstream health probing
- [ ] Better upstream latency tracking
- [ ] Connection pooling
- [ ] More detailed observability

## Management

- [ ] Private management API
- [ ] Authentication / authorization
- [ ] Runtime configuration management
- [ ] Adlist management
- [ ] Query statistics dashboard
- [ ] Upstream health dashboard
- [ ] Web dashboard

---

# 🧠 Design Principles

RAMDNS tidak mengejar fitur sebanyak mungkin.

Prinsipnya:

```text
simple
secure
fast
observable
boring to operate
```

Targetnya adalah membuat DNS server yang cepat, kecil, predictable, dan tidak bikin operator bangun jam 3 pagi gara-gara certificate expired. 🌙

---

# 📜 License

Tentukan license project sesuai kebutuhan distribusi RAMDNS.

---

Built with Go. 🐹

RAMDNS: DNS server kecil yang kerjanya serius.
