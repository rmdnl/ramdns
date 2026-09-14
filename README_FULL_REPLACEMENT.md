# RAMDNS 🚀

**RAMDNS** adalah public DNS resolver berbasis Go yang dibuat untuk VPS kecil sampai menengah, dengan fokus pada:

- ⚡ latency rendah dan concurrency
- 🔒 security-first operation
- 🧱 adlist filtering
- 🧠 cache + SingleFlight
- 🌐 DNS, DoT, dan DoH
- 🩺 upstream health awareness
- 📊 observability lokal
- 🛠️ deployment yang reproducible dan tidak bikin operator menangis jam 03:00

> Query masuk → rate limit → filter → cache → kalau perlu upstream → jawab.

RAMDNS sengaja tidak punya public admin API. Port management tetap lokal. Dashboard boleh cakep, tapi password dan token jangan pernah jalan-jalan ke browser. 😌

---

## ✨ Status Fitur

| Fitur | Status |
|---|---|
| DNS UDP `:53` | 🟢 |
| DNS TCP `:53` | 🟢 |
| DNS-over-TLS `:853` | 🟢 |
| DNS-over-HTTPS `:443` | 🟢 |
| DNSSEC | 🟢 |
| Persistent DoT upstream | 🟢 |
| Parallel upstream | 🟢 |
| Health-aware upstream routing | 🟢 |
| Upstream health probing | 🟢 |
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
| Local metrics | 🟢 |
| Private management API | 🟢 |
| Web dashboard | 🟢 |
| Public control API | 🔴 Sengaja tidak ada |
| Runtime configuration editor | 🟡 Bertahap |
| Prometheus-native metrics | 🟡 Roadmap |

---

# 🏗️ Arsitektur

```text
                                  INTERNET
                                      │
                 ┌────────────────────┼────────────────────┐
                 │                    │                    │
              UDP :53              TCP :53          Encrypted DNS
                                                         │
                                              ┌──────────┴──────────┐
                                              │                     │
                                           DoT :853             HTTPS :443
                                              │                     │
                                              │             ┌───────┴────────┐
                                              │             │                │
                                              │            DoH         Dashboard
                                              │       doh.ramdns...   dashboard.ramdns...
                                              │             │                │
                                              └─────────────┴────────┬───────┘
                                                                     │
                                                              ┌──────▼──────┐
                                                              │   RAMDNS    │
                                                              └──────┬──────┘
                                                                     │
                                                              Rate Limiter
                                                                     │
                                                              Adlist Filter
                                                                     │
                                                                    Cache
                                                                     │
                                                               SingleFlight
                                                                     │
                                                        Health-aware routing
                                                                     │
                                                       Parallel DoT exchange
                                                                     │
                                               ┌─────────────────────┴─────────────────────┐
                                               │                                           │
                                      Cloudflare DoT                              Google DoT
                                       1.1.1.1:853                                8.8.8.8:853
```

Public HTTPS pada `:443` dipisahkan berdasarkan hostname:

```text
https://doh.ramdns.my.id/dns-query
    └──> DoH

https://dashboard.ramdns.my.id/
    └──> Web Dashboard

hostname lain
    └──> HTTP 421 Misdirected Request
```

Management API dan metrics tidak dipublish ke Internet:

```text
127.0.0.1:8502  -> Management API
127.0.0.1:8503  -> Dashboard backend
127.0.0.1:8080  -> Metrics
```

---

# 🧩 Request Flow

Untuk query normal:

```text
Client
  │
  ▼
DNS listener
  │
  ▼
Rate limit
  │
  ▼
Adlist filter
  │
  ├── blocked -> response lokal
  │
  ▼
Cache lookup
  │
  ├── hit -> response
  │
  ▼
SingleFlight
  │
  ▼
Healthy upstreams
  │
  ├── parallel exchange
  │
  ├── first successful response wins
  │
  ▼
Cache response
  │
  ▼
Client
```

Jadi query yang sama tidak perlu rame-rame mengetuk upstream seperti satu grup chat yang semua orang nanya hal sama.

---

# 📦 Project Structure

```text
ramdns/
├── cmd/
│   └── ramdns/
│       └── main.go
│
├── internal/
│   ├── adlist/
│   ├── cache/
│   ├── dashboard/
│   ├── dns/
│   ├── doh/
│   ├── filter/
│   ├── metrics/
│   ├── ratelimit/
│   └── upstream/
│
├── web/
│   └── dashboard/
│       └── index.html
│
├── deploy/
│   ├── certbot/
│   │   └── ramdns-cert.sh
│   ├── install.sh
│   └── systemd/
│       ├── ramdns.service
│       ├── adlist.conf
│       ├── capabilities.conf
│       ├── hardening.conf
│       └── upstream.conf
│
├── go.mod
├── go.sum
├── .gitignore
└── README.md
```

---

# 🖥️ Target Deployment

RAMDNS dirancang untuk footprint yang relatif kecil. Deployment yang digunakan selama pengembangan/operasional proyek ini:

```text
OS       : Ubuntu
CPU      : 2 vCPU
RAM      : ~2 GB
Swap     : 0
Runtime  : systemd
User     : ubuntu
```

RAMDNS tetap bergantung pada:

- network yang sehat
- DNS upstream yang reachable
- certificate TLS yang valid
- file permission yang benar
- kernel/systemd yang mendukung hardening yang digunakan

Untuk public resolver, VPS juga butuh public IPv4 dan/atau IPv6.

---

# 🚀 Fresh VPS Deployment

Untuk fresh Ubuntu, repository sudah menyediakan installer:

```text
deploy/install.sh
```

Installer menyiapkan:

- binary RAMDNS
- systemd unit
- systemd drop-ins
- hardening
- `CAP_NET_BIND_SERVICE`
- konfigurasi upstream
- konfigurasi adlist
- management token
- enable + start service

Installer **tidak** menyimpan:

- Cloudflare API token
- private key TLS
- certificate secret

di Git.

## 1. Siapkan repository

```bash
sudo mkdir -p /opt
sudo git clone https://github.com/rmdnl/ramdns.git /opt/ramdns
sudo chown -R ubuntu:ubuntu /opt/ramdns

cd /opt/ramdns
```

## 2. Jalankan installer

```bash
sudo bash deploy/install.sh
```

Installer mengharapkan user/group `ubuntu` tersedia di mesin.

## 3. Verifikasi service

```bash
systemctl status ramdns --no-pager
```

Atau:

```bash
systemctl is-active ramdns
```

## 4. Cek listener

```bash
sudo ss -lntup | grep -E ':(53|443|853)\b'
```

Expected public listeners:

```text
:53/udp
:53/tcp
:853/tcp
:443/tcp
```

Management dan metrics tetap localhost-only.

---

# 🧱 Manual Build

Installer adalah jalur deployment utama untuk fresh VPS, tetapi build manual tetap berguna untuk debugging.

## Install dependency

```bash
sudo apt update
sudo apt install -y \
  ca-certificates \
  curl \
  git \
  golang \
  openssl \
  ufw \
  certbot \
  python3-certbot-dns-cloudflare
```

Cek Go:

```bash
go version
```

## Download dependency dan test

```bash
cd /opt/ramdns
go mod download
go test ./...
```

## Build

```bash
go build -trimpath -o ramdns ./cmd/ramdns
```

---

# 👤 Runtime User dan Privilege

RAMDNS dijalankan sebagai user non-root:

```text
User  : ubuntu
Group : ubuntu
```

Karena DNS membutuhkan port rendah seperti `53`, service mendapat capability minimum:

```ini
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
```

Tujuannya sederhana:

> bisa bind ke port yang dibutuhkan, tapi tidak berubah jadi root cosplay. 🥸

---

# 🔥 Firewall

Port public yang dibutuhkan RAMDNS:

```text
22/tcp    SSH
53/tcp    DNS
53/udp    DNS
853/tcp   DNS-over-TLS
443/tcp   DNS-over-HTTPS + Dashboard
```

Contoh baseline:

```bash
sudo ufw default deny incoming
sudo ufw allow 22/tcp
sudo ufw allow 53/tcp
sudo ufw allow 53/udp
sudo ufw allow 853/tcp
sudo ufw allow 443/tcp
sudo ufw enable
sudo ufw status verbose
```

`80/tcp` tidak diperlukan untuk certificate issuance karena RAMDNS menggunakan DNS-01.

Port berikut **jangan dibuka ke Internet**:

```text
8080
8502
8503
```

Port `8501` tidak dikelola RAMDNS dan dapat tetap digunakan service lain apabila memang dibutuhkan host.

> Jangan mengaktifkan UFW sebelum SSH sudah diizinkan. Server yang terkunci dari admin panel itu plot twist yang tidak diperlukan.

---

# 🌐 DNS Records

RAMDNS saat ini menggunakan tiga hostname:

```text
dot.ramdns.my.id
doh.ramdns.my.id
dashboard.ramdns.my.id
```

Record IPv4:

```text
dot.ramdns.my.id        A     <SERVER_IPV4>
doh.ramdns.my.id        A     <SERVER_IPV4>
dashboard.ramdns.my.id  A     <SERVER_IPV4>
```

Untuk IPv6:

```text
dot.ramdns.my.id        AAAA  <SERVER_IPV6>
doh.ramdns.my.id        AAAA  <SERVER_IPV6>
dashboard.ramdns.my.id  AAAA  <SERVER_IPV6>
```

Pastikan ketiga hostname mengarah ke server yang sama sebelum issuance certificate.

---

# 🔐 TLS / Let's Encrypt / Cloudflare DNS-01

RAMDNS menggunakan:

```text
Let's Encrypt
Certbot
Cloudflare DNS-01
```

Kenapa DNS-01?

Karena port `80` tidak perlu dibuka. ACME challenge dilakukan melalui DNS.

## 1. Cloudflare API Token

Gunakan API Token khusus RAMDNS dengan least privilege.

Permission yang dibutuhkan:

```text
Zone
└── DNS
    └── Edit
```

Scope-kan hanya ke zone:

```text
ramdns.my.id
```

Jangan taruh token tersebut di repository.

## 2. Credential file

```bash
sudo install -d -m 700 /root/.secrets/certbot
sudo nano /root/.secrets/certbot/cloudflare.ini
```

Isi:

```ini
dns_cloudflare_api_token = YOUR_CLOUDFLARE_API_TOKEN
```

Lock permission:

```bash
sudo chmod 600 /root/.secrets/certbot/cloudflare.ini
```

## 3. Issue certificate final

Certificate sebaiknya mencakup semua hostname public:

```bash
sudo certbot certonly \
  --dns-cloudflare \
  --dns-cloudflare-credentials /root/.secrets/certbot/cloudflare.ini \
  --dns-cloudflare-propagation-seconds 30 \
  -d dot.ramdns.my.id \
  -d doh.ramdns.my.id \
  -d dashboard.ramdns.my.id
```

Cek:

```bash
sudo certbot certificates
```

Certificate lineage biasanya berada di:

```text
/etc/letsencrypt/live/dot.ramdns.my.id/
```

---

# 🔗 TLS Files untuk RAMDNS

RAMDNS membaca certificate dari:

```text
/etc/ramdns/tls/fullchain.pem
/etc/ramdns/tls/privkey.pem
```

Directory:

```bash
sudo install -d -o ubuntu -g ubuntu -m 700 /etc/ramdns/tls
```

Permission yang direkomendasikan:

```text
fullchain.pem  -> 0644 ubuntu:ubuntu
privkey.pem    -> 0600 ubuntu:ubuntu
```

Private key jangan pernah masuk Git.

---

# ♻️ Certificate Deployment Hook

Repository menyediakan:

```text
deploy/certbot/ramdns-cert.sh
```

Konsep deployment hook:

```text
Certbot renewal
      │
      ▼
Deploy hook
      │
      ├── copy fullchain.pem
      ├── copy privkey.pem
      └── restart ramdns
```

Install hook:

```bash
sudo install -o root -g root -m 0700 \
  deploy/certbot/ramdns-cert.sh \
  /etc/letsencrypt/renewal-hooks/deploy/ramdns-cert.sh
```

Pastikan hook menggunakan certificate lineage yang benar pada mesin target.

---

# ⏰ Automatic Certificate Renewal

Cek timer:

```bash
systemctl status certbot.timer --no-pager
```

Enable:

```bash
sudo systemctl enable --now certbot.timer
```

Lihat jadwal:

```bash
systemctl list-timers certbot.timer --no-pager
```

Test renewal:

```bash
sudo certbot renew --dry-run
```

---

# 🔍 TLS Verification

## DoT

```bash
openssl s_client \
  -connect dot.ramdns.my.id:853 \
  -servername dot.ramdns.my.id \
  -tls1_3 </dev/null 2>/dev/null \
  | openssl x509 -noout -subject -issuer -dates -ext subjectAltName
```

## DoH

```bash
openssl s_client \
  -connect doh.ramdns.my.id:443 \
  -servername doh.ramdns.my.id \
  -tls1_3 </dev/null 2>/dev/null \
  | openssl x509 -noout -subject -issuer -dates -ext subjectAltName
```

## Dashboard

```bash
openssl s_client \
  -connect dashboard.ramdns.my.id:443 \
  -servername dashboard.ramdns.my.id \
  -tls1_3 </dev/null 2>/dev/null \
  | openssl x509 -noout -subject -issuer -dates -ext subjectAltName
```

SAN harus mencakup:

```text
DNS:dot.ramdns.my.id
DNS:doh.ramdns.my.id
DNS:dashboard.ramdns.my.id
```

Minimum TLS:

```text
TLS 1.3
```

---

# ⚙️ systemd

File deployment:

```text
deploy/systemd/ramdns.service
deploy/systemd/adlist.conf
deploy/systemd/capabilities.conf
deploy/systemd/hardening.conf
deploy/systemd/upstream.conf
```

Hardening utama:

```ini
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
```

Tujuan hardening:

- mengurangi filesystem write access
- mengurangi visibility terhadap process lain
- membatasi kernel-facing features
- membatasi network families
- membatasi privilege escalation
- menjaga file descriptor budget tetap cukup untuk DNS workload

---

# 🔐 Management API

Management API sekarang private dan listen pada:

```text
127.0.0.1:8502
```

Endpoint yang tersedia:

```text
GET /api/v1/status
GET /api/v1/metrics
GET /api/v1/upstreams
```

Semua request membutuhkan Bearer token.

Contoh:

```bash
TOKEN="$(sudo sed -n 's/^RAMDNS_MANAGEMENT_TOKEN=//p' \
  /etc/ramdns/management.env)"

curl -sS \
  http://127.0.0.1:8502/api/v1/status \
  -H "Authorization: Bearer $TOKEN"
```

Expected:

```json
{
  "status": "ok",
  "service": "ramdns",
  "api": "v1"
}
```

Upstream health:

```bash
curl -sS \
  http://127.0.0.1:8502/api/v1/upstreams \
  -H "Authorization: Bearer $TOKEN"
```

Response berisi health state dan statistik upstream, termasuk:

```text
healthy
queries
successes
failures
latency_ns
```

Management token disimpan server-side di:

```text
/etc/ramdns/management.env
```

Permission file:

```text
0600 root:root
```

---

# 🖥️ Web Dashboard

Dashboard tersedia melalui:

```text
https://dashboard.ramdns.my.id/
```

Dashboard static frontend di:

```text
web/dashboard/
```

Backend/dashboard path:

```text
127.0.0.1:8503
```

Yang penting dari sisi security:

```text
Browser
  │
  ▼
dashboard.ramdns.my.id
  │
  ▼
RAMDNS dashboard proxy
  │
  ▼
127.0.0.1:8502
```

Token management **tidak disimpan di browser**.

Tidak ada:

```text
localStorage token
sessionStorage token
browser Authorization token
token prompt di frontend
```

Dashboard menggunakan backend server-side untuk meneruskan authentication ke management API.

Jadi browser tidak perlu pegang crown jewels. 👑

---

# 🔒 Public HTTPS Routing

Port `443/tcp` dipakai bersama oleh DoH dan dashboard.

Routing berdasarkan hostname:

```text
Host: doh.ramdns.my.id
    └──> DoH handler

Host: dashboard.ramdns.my.id
    └──> Dashboard handler

Host: anything-else.example
    └──> HTTP 421 Misdirected Request
```

Ini penting supaya satu listener HTTPS tidak berubah menjadi “semua request dilempar ke mana aja”.

DoH endpoint:

```text
https://doh.ramdns.my.id/dns-query
```

Dashboard:

```text
https://dashboard.ramdns.my.id/
```

---

# 🌐 DNS-over-HTTPS

Endpoint canonical:

```text
https://doh.ramdns.my.id/dns-query
```

Transport:

```text
HTTPS
HTTP
DNS wire format
```

Untuk smoke test jaringan, cukup pastikan TLS dan HTTP endpoint reachable dari client.

---

# 🔐 DNS-over-TLS

Endpoint:

```text
dot.ramdns.my.id:853
```

TLS minimum:

```text
TLS 1.3
```

Certificate verification untuk upstream juga tetap aktif.

Smoke test:

```bash
openssl s_client \
  -connect dot.ramdns.my.id:853 \
  -servername dot.ramdns.my.id \
  -tls1_3
```

Untuk functional DNS client test, gunakan resolver/DoT client yang benar-benar mengirim DNS-over-TLS, bukan sekadar membuka TCP.

---

# 🧠 Upstream DNS

Current upstream:

```text
1.1.1.1:853 | cloudflare-dns.com
8.8.8.8:853 | dns.google
```

Konfigurasi systemd:

```ini
[Service]
Environment="RAMDNS_UPSTREAMS=1.1.1.1:853|cloudflare-dns.com,8.8.8.8:853|dns.google"
```

Upstream menggunakan:

```text
DoT
TLS verification
TLS 1.3 minimum
```

---

# ⚡ Parallel Upstream

RAMDNS dapat mengirim query ke upstream sehat secara paralel.

```text
                 ┌── Cloudflare ──┐
                 │                │
Query ───────────┤                ├──> first success
                 │                │
                 └── Google ──────┘
```

Tujuan utama:

- mengurangi tail latency
- menghindari bergantung pada satu upstream
- tetap punya fallback ketika salah satu provider sedang tidak bahagia

---

# 🩺 Upstream Health Awareness

RAMDNS melakukan health check berkala pada upstream.

State yang relevan:

```text
healthy
unhealthy
recovering
```

Konsep routing:

```text
healthy upstreams
       │
       ▼
parallel exchange
       │
       ▼
first successful response
```

Jika seluruh upstream untuk sementara ditandai unhealthy, resolver tetap memiliki fallback behavior untuk mencoba configured upstreams daripada langsung menyerah.

Parameter operasional utama saat ini:

```text
default query timeout : 3s
per-server timeout    : 1.5s
dial timeout           : 1.5s
idle timeout           : 30s
health interval        : 10s
probe timeout          : 1.2s
failures before down   : 3
successes before up    : 2
```

`latency_ns` pada management API merepresentasikan latency exchange terakhir yang tercatat untuk upstream tersebut, bukan rolling average.

---

# 🔗 Persistent DoT

Koneksi upstream tidak harus handshake dari nol untuk setiap query.

Konsep:

```text
TCP connect
    ↓
TLS 1.3 handshake
    ↓
DNS exchange
    ↓
reuse connection
    ↓
DNS exchange berikutnya
```

Connection idle akan ditutup sesuai idle timeout.

Ini penting karena TLS handshake mahal kalau dilakukan seperti lagi kenalan ulang setiap lima detik. 😭

---

# 💾 DNS Cache

RAMDNS memakai bounded response cache.

Karakteristik utama:

- cache hit/miss tracking
- TTL-aware behavior
- response isolation/copy
- bounded capacity
- query key mempertimbangkan DNS request attributes yang relevan

Secara konseptual:

```text
QNAME
QTYPE
QCLASS
DO bit
```

Cache bukan database permanen. Restart service berarti isi cache hilang dan itu normal.

---

# 🧩 SingleFlight

SingleFlight menggabungkan request identik yang datang bersamaan.

```text
100 client requests
       │
       ▼
   SingleFlight
       │
       ▼
1 upstream exchange
       │
       ▼
shared result
```

Benefits:

- mengurangi duplicate upstream queries
- menekan burst load
- membantu cache warm-up
- lebih ramah ke upstream

---

# 🧱 Adlist / DNS Filtering

RAMDNS mendukung parser:

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

Filtering dilakukan sebelum resolver mengirim query ke upstream untuk domain yang cocok.

---

# 📚 Current Adlist Sources

Source yang digunakan saat ini:

```text
https://big.oisd.nl/
https://hagezi-mirror.dnsbunker.org/adblock/pro.txt
https://hagezi-mirror.dnsbunker.org/adblock/tif.txt
```

Systemd config:

```ini
[Service]
Environment="RAMDNS_ADLIST_URL=https://big.oisd.nl/,https://hagezi-mirror.dnsbunker.org/adblock/pro.txt,https://hagezi-mirror.dnsbunker.org/adblock/tif.txt"
```

Jumlah rule bersifat dinamis karena upstream list dapat berubah.

---

# 🔄 Adlist Lifecycle

Flow:

```text
download
   ↓
validate
   ↓
parse
   ↓
compile
   ↓
atomic replace
```

Jika update gagal:

```text
new snapshot gagal
       ↓
snapshot lama tetap aktif
```

Jadi update list tidak boleh berubah menjadi “filter hilang total karena satu HTTP request ngadat”.

Downloader mendukung:

```text
HTTPS
ETag
Last-Modified
If-None-Match
If-Modified-Since
304 Not Modified
timeout
response size guard
```

Download source dilakukan secara concurrent.

Reload menggunakan replace yang atomic dari sudut pandang query path.

---

# 🧮 Adlist Memory Design

Adlist bisa berisi jutaan rule. Karena itu memory retention perlu dijaga.

Runtime filtering menggunakan compiled matcher tanpa mempertahankan duplicate rule representations yang tidak diperlukan setelah replacement.

Karakteristik yang diinginkan:

```text
source data
    ↓
parse
    ↓
compile
    ↓
active matcher
```

bukan:

```text
source data
    ↓
parse
    ↓
compile
    ↓
compile
    ↓
compile
    ↓
semua duplikat nongkrong di RAM
```

Benchmark internal yang pernah digunakan untuk validasi filter replacement:

```text
BenchmarkReplace100K
BenchmarkIsBlocked100K
```

`IsBlocked` ditargetkan sangat murah pada hot path dan benchmark pernah menunjukkan zero allocation untuk lookup blocked-domain pada test tersebut.

---

# 🚦 Per-IP Rate Limiting

RAMDNS memiliki per-IP rate limit untuk menahan abuse dasar.

Konfigurasi deployment saat ini menargetkan:

```text
50 requests/second/IP
burst: 100
tracked IPs: 20,000
```

Catatan:

Rate limiting aplikasi **bukan** pengganti:

- provider-level DDoS protection
- upstream firewall
- CDN/scrubbing
- volumetric attack mitigation

Public resolver tetap harus dipasang di VPS/provider yang punya baseline network protection yang masuk akal.

---

# 📊 Local Metrics

Metrics internal tersedia di:

```text
http://127.0.0.1:8080/metrics
```

Contoh statistik:

```json
{
  "cache_hits": 257,
  "cache_misses": 2063,
  "upstream_queries": 2062,
  "upstream_errors": 1,
  "total_queries": 2320,
  "cache_hit_ratio": 0.11077586206896552
}
```

Metrics juga memberi insight tentang upstream health/statistics melalui management API.

Test:

```bash
curl -fsS http://127.0.0.1:8080/metrics
```

Port `8080` tetap localhost-only.

---

# 📈 Observability

Untuk operational debugging:

```bash
systemctl status ramdns --no-pager
```

```bash
journalctl -u ramdns -n 100 --no-pager
```

```bash
journalctl -u ramdns -f
```

Management API:

```bash
TOKEN="$(sudo sed -n 's/^RAMDNS_MANAGEMENT_TOKEN=//p' \
  /etc/ramdns/management.env)"

curl -sS \
  http://127.0.0.1:8502/api/v1/upstreams \
  -H "Authorization: Bearer $TOKEN"
```

Ini membantu melihat:

- upstream healthy/unhealthy
- query count
- success/failure
- last exchange latency

---

# 🧪 Verification Checklist

## 1. Service

```bash
systemctl is-active ramdns
```

Expected:

```text
active
```

## 2. Listener

```bash
sudo ss -lntup | grep -E ':(53|443|853)\b'
```

## 3. Local DNS

```bash
dig @127.0.0.1 example.com A +stats
```

## 4. DNSSEC

```bash
dig @127.0.0.1 cloudflare.com A +dnssec
```

Periksa `ad` bila validation chain tersedia dan valid.

## 5. Metrics

```bash
curl -fsS http://127.0.0.1:8080/metrics
```

## 6. Management API

Tanpa token harus ditolak:

```bash
curl -i http://127.0.0.1:8502/api/v1/status
```

Dengan token harus berhasil:

```bash
TOKEN="$(sudo sed -n 's/^RAMDNS_MANAGEMENT_TOKEN=//p' \
  /etc/ramdns/management.env)"

curl -i \
  http://127.0.0.1:8502/api/v1/status \
  -H "Authorization: Bearer $TOKEN"
```

## 7. Upstream health

```bash
curl -sS \
  http://127.0.0.1:8502/api/v1/upstreams \
  -H "Authorization: Bearer $TOKEN"
```

## 8. TLS

```bash
openssl s_client \
  -connect dot.ramdns.my.id:853 \
  -servername dot.ramdns.my.id \
  -tls1_3 </dev/null
```

```bash
openssl s_client \
  -connect doh.ramdns.my.id:443 \
  -servername doh.ramdns.my.id \
  -tls1_3 </dev/null
```

```bash
openssl s_client \
  -connect dashboard.ramdns.my.id:443 \
  -servername dashboard.ramdns.my.id \
  -tls1_3 </dev/null
```

## 9. Adlist

```bash
journalctl -u ramdns -n 100 --no-pager | grep -i adlist
```

## 10. Certificate renewal

```bash
sudo certbot renew --dry-run
```

---

# 🧪 Benchmarking

Benchmark DNS tidak boleh dibaca sebagai angka sakral.

Latency sangat dipengaruhi:

```text
cache state
CPU
network path
upstream latency
VPS noisy neighbors
query mix
concurrency
```

Benchmark internal yang pernah dilakukan pada lingkungan proyek:

### Parallel workload

```text
500 random parallel queries

p50  = 0 ms
p90  = 4 ms
p95  = 4 ms
p99  = 8 ms
max  = 8 ms
SERVFAIL = 0
```

### Sequential workload

```text
5,000 queries

p50  = 4 ms
p95  = 4 ms
p99  = 4 ms
max  = 16 ms
```

### Concurrent workload

```text
5,000 queries
100 workers

p50  = 0 ms
p95  = 4 ms
p99  = 8 ms
max  = 16 ms
SERVFAIL = 0
```

Gunakan benchmark sebagai regression signal, bukan marketing number.

---

# 🔒 Security Model

RAMDNS menjaga beberapa boundary penting:

```text
PUBLIC
├── DNS UDP :53
├── DNS TCP :53
├── DoT :853
└── HTTPS :443

PRIVATE
├── 127.0.0.1:8080 metrics
├── 127.0.0.1:8502 management API
└── 127.0.0.1:8503 dashboard backend
```

Secrets:

```text
Cloudflare credential
    -> /root/.secrets/certbot/cloudflare.ini

Management token
    -> /etc/ramdns/management.env

TLS private key
    -> /etc/ramdns/tls/privkey.pem
```

Jangan commit:

```text
*.pem
*.key
*.crt
*.ini
.env
cloudflare.ini
management.env
API tokens
private keys
```

---

# 🧹 Secret Hygiene

Kalau secret pernah bocor ke Git:

1. revoke/rotate secret
2. buat credential baru
3. hapus secret dari working tree
4. purge dari history bila perlu
5. update server dengan credential baru

Menghapus file dari HEAD saja tidak otomatis menghapus secret dari Git history.

Security itu bukan:

> "kan sekarang filenya udah dihapus."

Git jawab:

> "I remember." 👁️

---

# 🩺 Troubleshooting

## RAMDNS tidak start

```bash
systemctl status ramdns --no-pager
journalctl -u ramdns -n 100 --no-pager
```

## Port conflict

```bash
sudo ss -lntup | grep -E ':(53|443|853)\b'
```

## Dashboard 421

Periksa Host header dan DNS:

```bash
curl -ki https://dashboard.ramdns.my.id/
```

Kalau hostname salah atau request diarahkan ke hostname yang tidak dikenal, `421 Misdirected Request` memang expected.

## DoH tidak bekerja

Pastikan:

```text
doh.ramdns.my.id
    ↓
SERVER_IP
    ↓
TCP 443
    ↓
valid TLS certificate
    ↓
/dns-query
```

Periksa:

```bash
journalctl -u ramdns -n 100 --no-pager
```

## SERVFAIL

Periksa:

```bash
journalctl -u ramdns -n 100 --no-pager
```

Kemudian cek health upstream:

```bash
TOKEN="$(sudo sed -n 's/^RAMDNS_MANAGEMENT_TOKEN=//p' \
  /etc/ramdns/management.env)"

curl -sS \
  http://127.0.0.1:8502/api/v1/upstreams \
  -H "Authorization: Bearer $TOKEN"
```

## Adlist tidak update

Periksa:

```bash
journalctl -u ramdns -n 200 --no-pager | grep -i adlist
```

Pastikan server bisa keluar HTTPS ke source list.

## Certificate gagal renewal

```bash
sudo certbot certificates
sudo certbot renew --dry-run
sudo journalctl -u certbot.service --no-pager
sudo tail -n 100 /var/log/letsencrypt/letsencrypt.log
```

## Certificate sudah diperbarui tetapi RAMDNS masih pakai cert lama

Periksa:

```bash
sudo ls -l /etc/ramdns/tls/
```

Periksa hook:

```bash
sudo sed -n '1,220p' \
  /etc/letsencrypt/renewal-hooks/deploy/ramdns-cert.sh
```

Lalu:

```bash
systemctl restart ramdns
```

dan verify TLS lagi.

---

# 🔄 Upgrade / Redeploy

Untuk update dari Git:

```bash
cd /opt/ramdns
git fetch origin
git checkout main
git pull --ff-only origin main
```

Build:

```bash
go test ./...
go build -trimpath -o ramdns ./cmd/ramdns
```

Restart setelah memastikan configuration dan certificate tetap tersedia:

```bash
sudo systemctl restart ramdns
```

Cek:

```bash
systemctl status ramdns --no-pager
```

Untuk fresh host, gunakan kembali:

```bash
sudo bash deploy/install.sh
```

Jangan menyalin secret dari Git. Secret harus datang dari host deployment.

---

# 📋 Operational Checklist

### DNS

```text
[ ] UDP :53 reachable
[ ] TCP :53 reachable
[ ] DNSSEC validation checked
```

### Encrypted DNS

```text
[ ] DoT :853 reachable
[ ] DoH :443 reachable
[ ] TLS 1.3 works
[ ] SAN correct
```

### Dashboard

```text
[ ] dashboard.ramdns.my.id resolves
[ ] HTTPS works
[ ] management API stays localhost-only
[ ] browser never receives management token
```

### Upstream

```text
[ ] Cloudflare healthy
[ ] Google healthy
[ ] health probe works
[ ] fallback behavior tested
```

### Filtering

```text
[ ] adlist downloads
[ ] adlist compiles
[ ] adlist reload is atomic
[ ] blocked-domain lookup works
```

### Operations

```text
[ ] systemd enabled
[ ] systemd hardening active
[ ] UFW rules correct
[ ] certificate timer active
[ ] certbot dry-run passed
[ ] logs clean
```

---

# 🧭 Roadmap

Yang sudah selesai:

```text
[x] UDP DNS
[x] TCP DNS
[x] DNSSEC
[x] DoT
[x] DoH
[x] Persistent DoT
[x] Parallel upstream
[x] Health-aware upstream routing
[x] Health probing
[x] DNS cache
[x] SingleFlight
[x] Adlist parser
[x] Multi-source adlist
[x] Concurrent adlist download
[x] Atomic adlist reload
[x] ETag / Last-Modified
[x] Periodic adlist refresh
[x] Per-IP rate limiting
[x] TLS 1.3 minimum
[x] Let's Encrypt
[x] Cloudflare DNS-01
[x] systemd hardening
[x] Local metrics
[x] Private management API
[x] Server-side dashboard authentication proxy
[x] Public HTTPS hostname routing
[x] Fresh VPS installer
```

Yang masih bisa ditingkatkan:

```text
[ ] richer runtime configuration management
[ ] better latency history / rolling averages
[ ] Prometheus-native metrics
[ ] more dashboard analytics
[ ] query analytics without exposing sensitive query data
[ ] zero-downtime certificate reload if architecture permits
[ ] additional upstream strategy experimentation
```

---

# 🧠 Design Principles

RAMDNS tidak mengejar jumlah fitur yang bisa dipajang di README.

Prinsipnya:

```text
simple
secure
fast
observable
predictable
boring to operate
```

Prefer:

```text
small surface area
least privilege
local management
explicit configuration
safe defaults
measurable behavior
```

Avoid:

```text
public admin ports
browser-held secrets
magic auto-configuration
unbounded memory growth
silent failure
"works on my VPS™"
```

Target akhirnya:

> DNS resolver yang cepat, private, predictable, gampang di-debug, dan cukup boring untuk dipakai setiap hari.

Karena infrastruktur terbaik sering justru yang tidak bikin notifikasi jam 03:17. 🌙📟

---

# 📜 License

Tentukan license project sesuai kebutuhan distribusi RAMDNS.

---

## 🐹 Built with Go

RAMDNS dibuat untuk orang yang ingin punya resolver DNS sendiri tanpa mengubah VPS menjadi taman bermain dependency.

**RAMDNS**  
Fast DNS. Private ops. Fewer surprises. 🚀
