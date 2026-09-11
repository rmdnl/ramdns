# RAMDNS 🚀

> **DNS server sendiri. Cepat, private, bisa ngeblok iklan, dan nggak
> perlu dashboard admin yang nongkrong terbuka di Internet.**

RAMDNS adalah **recursive DNS resolver berbasis Go** yang dibangun buat
VPS kecil tapi tetap pengen performa serius.

Filosofinya simpel:

**query masuk → filter → cache → kalau perlu baru tembak upstream →
jawab.**

Nggak ada magic. Nggak ada 47 service cuma buat resolve `google.com`. 🗿

------------------------------------------------------------------------

## 🔥 Kenapa RAMDNS?

Karena DNS itu harusnya boring.

Client nanya:

``` text
"IP google.com apa?"
```

DNS jawab:

``` text
"Ini."
```

Selesai.

RAMDNS dibuat supaya proses sesimpel itu tetap:

-   ⚡ cepat
-   🔐 terenkripsi ke upstream
-   🧱 bisa blok iklan/tracker
-   💾 hemat query dengan cache
-   🧩 tahan burst dengan SingleFlight
-   🚦 punya rate limiting
-   🛡️ jalan dengan systemd hardening
-   🧹 punya attack surface yang kecil

------------------------------------------------------------------------

# ✨ Fitur

  Fitur                    Status
  ------------------------ ----------------------
  DNS UDP `:53`            🟢
  DNS TCP `:53`            🟢
  DNS over TLS `:853`      🟢
  DNS over HTTPS `:443`    🟢
  DNSSEC validation        🟢
  Persistent DoT           🟢
  Parallel upstream        🟢
  DNS response cache       🟢
  SingleFlight             🟢
  Multi-source adlist      🟢
  Atomic adlist reload     🟢
  ETag / Last-Modified     🟢
  Periodic adlist update   🟢
  Client rate limiting     🟢
  systemd hardening        🟢
  Public control API       🔴 Sengaja nggak ada

------------------------------------------------------------------------

# 🏗️ Arsitektur

``` text
                        CLIENT
                          │
             ┌────────────┼────────────┐
             │            │            │
          UDP :53      TCP :53      DoT :853
             │            │            │
             └────────────┼────────────┘
                          │
                       DoH :443
                          │
                          ▼
                 ┌─────────────────┐
                 │     RAMDNS      │
                 │    RESOLVER     │
                 └────────┬────────┘
                          │
             ┌────────────┼────────────┐
             │            │            │
             ▼            ▼            ▼
          🧱 FILTER     💾 CACHE    🧩 SINGLEFLIGHT
             │            │            │
             └────────────┼────────────┘
                          │
                          ▼
                  🔐 PERSISTENT DoT
                          │
                  ┌───────┴───────┐
                  ▼               ▼
             Cloudflare         Google
              1.1.1.1           8.8.8.8
```

------------------------------------------------------------------------

# 🚀 DNS

DNS standar tersedia di:

``` text
UDP :53
TCP :53
```

Test:

``` bash
dig @127.0.0.1 google.com A +stats
```

Kalau dapat `NOERROR`, resolver hidup.

Kalau dapat `SERVFAIL`, nah itu baru waktunya mulai nyari setan. 👻

------------------------------------------------------------------------

# 🔐 DNS over TLS

RAMDNS menyediakan DoT di:

``` text
TCP :853
```

Upstream recursive DNS juga menggunakan DoT.

Default:

``` text
1.1.1.1:853|cloudflare-dns.com
8.8.8.8:853|dns.google
```

TLS minimum:

``` text
TLS 1.3
```

Certificate verification:

``` text
ON
```

Jadi bukan:

``` text
InsecureSkipVerify: true
```

karena kita masih punya harga diri. 🗿

## Konfigurasi upstream

Buat drop-in:

``` text
/etc/systemd/system/ramdns.service.d/upstream.conf
```

Contoh:

``` ini
[Service]
Environment="RAMDNS_UPSTREAMS=1.1.1.1:853|cloudflare-dns.com,8.8.8.8:853|dns.google"
```

Setelah mengubah:

``` bash
sudo systemctl daemon-reload
sudo systemctl restart ramdns
```

------------------------------------------------------------------------

# 🌐 DNS over HTTPS

DoH tersedia di:

``` text
HTTPS :443
```

Endpoint:

``` text
/dns-query
```

DoH menggunakan DNS wire format melalui HTTPS.

Cek listener:

``` bash
sudo ss -lntup | grep -E ':(53|853|443)\b'
```

Expected:

``` text
*:53
*:853
*:443
```

------------------------------------------------------------------------

# 🛡️ DNSSEC

RAMDNS mendukung DNSSEC validation.

Test:

``` bash
dig @127.0.0.1 cloudflare.com A +dnssec
```

Response tervalidasi akan memiliki flag:

``` text
ad
```

Kalau `ad` muncul, DNSSEC-nya lagi kerja. 🔐

------------------------------------------------------------------------

# 🧱 Ad & Tracker Blocking

Nah, ini bagian yang lumayan enak.

RAMDNS bisa menggabungkan **beberapa adlist sekaligus**.

Deployment saat ini menggunakan:

``` text
3 sources
266.554 rules
```

Jumlah rules tentu bisa berubah ketika list upstream diperbarui.

## Source saat ini

``` text
HaGeZi Multi
StevenBlack Hosts
AdAway Hosts
```

## Format yang didukung

### Hosts

``` text
0.0.0.0 ads.example.com
127.0.0.1 tracker.example.com
:: ads.example.com
```

### Plain domain

``` text
ads.example.com
tracker.example.com
```

### Adblock

``` text
||ads.example.com^
||tracker.example.com^
```

Jadi parser nggak cuma ngerti satu jenis list.

------------------------------------------------------------------------

# ➕ Nambah Adlist

Adlist dikonfigurasi lewat systemd drop-in:

``` text
/etc/systemd/system/ramdns.service.d/adlist.conf
```

Contoh:

``` ini
[Service]
Environment="RAMDNS_ADLIST_URL=https://list-a.example/list.txt,https://list-b.example/list.txt,https://list-c.example/list.txt"
```

URL dipisahkan dengan koma.

Setelah edit:

``` bash
sudo systemctl daemon-reload
sudo systemctl restart ramdns
```

Cek:

``` bash
journalctl -u ramdns -n 20 --no-pager | grep adlist
```

Contoh sukses:

``` text
adlist updated sources=3 rules=266554 duration=1.2s
```

## 🔄 Cara update adlist

Worker melakukan:

``` text
startup
   ↓
download source
   ↓
parse
   ↓
compile rules
   ↓
gabungkan
   ↓
atomic replace
   ↓
tunggu 6 jam
   ↓
ulang lagi
```

Download beberapa source dilakukan secara concurrent.

Kalau satu source gagal:

``` text
source gagal ❌
     ↓
snapshot lama tetap aktif ✅
```

Jadi satu server list ngambek tidak bikin seluruh blocking mati.

RAMDNS juga mendukung:

``` text
ETag
Last-Modified
304 Not Modified
```

supaya tidak download ulang data yang belum berubah.

------------------------------------------------------------------------

# 💾 Cache

RAMDNS punya DNS response cache.

Karakteristik:

-   TTL-aware
-   bounded
-   response di-copy sebelum dikembalikan
-   expired entry dibuang
-   minimum TTL digunakan untuk menentukan lifetime cache
-   negative response bisa dicache jika punya TTL valid

Kapasitas default saat ini:

``` text
50.000 entries
```

Cache key mempertimbangkan:

``` text
QNAME
QTYPE
QCLASS
DO bit
```

Jadi query yang sudah ada di cache nggak perlu jalan-jalan lagi ke
Internet.

------------------------------------------------------------------------

# 🧩 SingleFlight

Misalnya 100 client tiba-tiba nanya:

``` text
example.com
```

bersamaan.

Tanpa coalescing:

``` text
100 client
   │
   ├──► upstream
   ├──► upstream
   ├──► upstream
   └──► ...
```

RAMDNS:

``` text
100 client
   │
   ▼
SingleFlight
   │
   ▼
1 upstream query
   │
   ▼
100 client
```

Lebih hemat upstream traffic dan CPU.

------------------------------------------------------------------------

# 🔗 Persistent DoT

Salah satu optimasi paling signifikan di RAMDNS.

Versi lama bisa terkena biaya:

``` text
TCP connect
    ↓
TLS handshake
    ↓
DNS query
    ↓
close
```

berulang-ulang.

Sekarang koneksi DoT dipertahankan dan digunakan kembali:

``` text
TCP
  ↓
TLS 1.3
  ↓
┌──────────────────────┐
│ query 1              │
│ query 2              │
│ query 3              │
│ query 4              │
│ ...                  │
└──────────────────────┘
```

Hasil benchmark-nya lumayan brutal. 🔥

------------------------------------------------------------------------

# ⚡ Parallel Upstream

RAMDNS menggunakan lebih dari satu upstream untuk resilience.

Contoh:

``` text
Cloudflare ─┐
            ├──► RAMDNS
Google ─────┘
```

Tujuannya menghindari satu upstream lambat membuat seluruh resolver ikut
lemot.

Konfigurasi:

``` ini
[Service]
Environment="RAMDNS_UPSTREAMS=1.1.1.1:853|cloudflare-dns.com,8.8.8.8:853|dns.google"
```

------------------------------------------------------------------------

# 🚦 Rate Limiting

RAMDNS mempunyai rate limiter berbasis client IP.

Tujuannya:

-   mengontrol burst
-   mengurangi abuse
-   mencegah satu client menghabiskan resource
-   menjaga resolver tetap responsif

Default implementation saat ini menggunakan rate limit per IP dengan
burst dan batas jumlah IP yang dilacak.

------------------------------------------------------------------------

# 🔒 Security & systemd Hardening

RAMDNS tidak perlu jalan sebagai root.

Service menggunakan capability minimum untuk bind ke low port:

``` ini
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
```

Proteksi systemd meliputi:

``` ini
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictSUIDSGID=true
RestrictRealtime=true
RestrictNamespaces=true
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6
SystemCallArchitectures=native
LimitNOFILE=65536
TasksMax=4096
```

Tujuannya simpel:

**kalau service nggak butuh akses, jangan kasih akses.**

------------------------------------------------------------------------

# 🚫 Kenapa Nggak Ada Port 8080?

Ada control API lokal di versi sebelumnya.

Sekarang sudah dihapus.

RAMDNS aktif hanya membutuhkan:

``` text
53/udp
53/tcp
853/tcp
443/tcp
```

Tidak ada:

``` text
8080
```

Less surface area, less headache. 😎

------------------------------------------------------------------------

# 🔐 TLS Certificate

Contoh lokasi certificate:

``` text
/etc/ramdns/tls/fullchain.pem
/etc/ramdns/tls/privkey.pem
```

Hostname certificate harus sesuai dengan hostname DoT/DoH.

Contoh:

``` text
ramdns.example.com
dot.ramdns.example.com
doh.ramdns.example.com
```

**Production deployment wajib punya certificate renewal otomatis.**

Certificate expired itu bukan warning kecil.

Itu:

``` text
DoT ❌
DoH ❌
user: "kok DNS mati?"
admin: 💀
```

------------------------------------------------------------------------

# 🧪 Testing

Run semua test:

``` bash
go test ./...
```

Build:

``` bash
go build -o ramdns ./cmd/ramdns
```

DNS:

``` bash
dig @127.0.0.1 cloudflare.com A +stats
```

DNSSEC:

``` bash
dig @127.0.0.1 cloudflare.com A +dnssec
```

Service:

``` bash
systemctl is-active ramdns
```

Listener:

``` bash
sudo ss -lntup | grep -E ':(53|853|443)\b'
```

Adlist:

``` bash
journalctl -u ramdns -n 50 --no-pager | grep adlist
```

------------------------------------------------------------------------

# 📊 Benchmark

Hardware pengujian:

``` text
CPU : 2 vCPU
RAM : ~1.9 GiB
```

## Sebelum optimasi upstream

Benchmark awal:

``` text
500 queries

p50  = 12 ms
p95  = 200 ms
p99  = 704 ms
max  = 1520 ms
```

Ada:

``` text
3 SERVFAIL
```

Jelas tail latency-nya masih bisa dibenerin.

------------------------------------------------------------------------

## Setelah parallel upstream

``` text
500 queries

p50  = 16 ms
p95  = 32 ms
p99  = 52 ms
max  = 76 ms
```

SERVFAIL:

``` text
0
```

------------------------------------------------------------------------

## Setelah persistent DoT

``` text
500 queries

p50  = 0 ms
p90  = 4 ms
p95  = 4 ms
p99  = 8 ms
max  = 8 ms
```

------------------------------------------------------------------------

## 5.000 query sequential

``` text
count = 5000
p50   = 4 ms
p95   = 4 ms
p99   = 4 ms
max   = 16 ms
```

------------------------------------------------------------------------

## 5.000 query / 100 concurrent workers

``` text
count = 5000
p50   = 0 ms
p95   = 4 ms
p99   = 8 ms
max   = 16 ms
```

Observed:

``` text
SERVFAIL = 0
```

Network interface selama pengujian juga tidak menunjukkan error/drop.

> Benchmark localhost terutama menunjukkan performa resolver dan
> connection handling. Latency client Internet akan berbeda.

------------------------------------------------------------------------

# 📉 Resource Usage

Pada benchmark persistent DoT sebelumnya:

``` text
CPU ≈ 14.7%
RSS ≈ 73 MB
```

VPS pengujian:

``` text
2 vCPU
~1.9 GiB RAM
```

RAMDNS sendiri relatif kecil dibanding resource VPS.

------------------------------------------------------------------------

# ⚙️ Konfigurasi systemd

Drop-in yang digunakan:

``` text
/etc/systemd/system/ramdns.service.d/
├── adlist.conf
├── upstream.conf
├── capabilities.conf
└── hardening.conf
```

Setelah perubahan konfigurasi:

``` bash
sudo systemctl daemon-reload
sudo systemctl restart ramdns
```

Cek:

``` bash
sudo systemctl status ramdns --no-pager
```

------------------------------------------------------------------------

# 🔥 Firewall

Port public:

``` text
53/udp
53/tcp
853/tcp
443/tcp
```

SSH:

``` text
22/tcp
```

Port management jangan dibuka ke Internet kalau tidak perlu.

Rule firewall harus mengikuti kebutuhan deployment. Jangan buka port
karena "siapa tahu nanti kepake". Nanti-nanti itu sering berubah jadi
"kok kena scan". 🗿

------------------------------------------------------------------------

# 📁 Struktur Project

``` text
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
├── go.mod
├── go.sum
└── README.md
```

Komponen:

``` text
adlist    → download + parse + compile blocklist
cache     → DNS response cache
dns       → DNS utilities
doh       → DNS over HTTPS
filter    → domain blocking
metrics   → resolver counters
ratelimit → client rate limiting
upstream  → encrypted recursive upstream
```

------------------------------------------------------------------------

# 🧑‍💻 Development

Clone:

``` bash
git clone git@github.com:rmdnl/ramdns.git
cd ramdns
```

Test:

``` bash
go test ./...
```

Build:

``` bash
go build -o ramdns ./cmd/ramdns
```

------------------------------------------------------------------------

# 🛣️ Roadmap

RAMDNS sudah jalan dan benchmark-nya bagus, tapi belum berarti boleh
rebahan selamanya. 😎

Prioritas berikutnya:

-   🩺 upstream health-aware routing
-   🧠 latency-aware upstream selection
-   🔌 connection pool dengan concurrency lebih tinggi
-   📊 per-upstream metrics
-   🛡️ DNS amplification resistance lebih ketat
-   🚨 abuse detection
-   🔐 DoT/DoH connection exhaustion protection
-   🔄 automatic TLS certificate renewal
-   📈 observability yang lebih lengkap
-   💾 cache eviction yang lebih efisien untuk workload ekstrem
-   🌍 IPv6 production hardening
-   🧪 load test dari beberapa lokasi Internet

Prinsip development:

> **Ukur dulu. Baru tuning. Jangan utak-atik sysctl cuma karena
> kelihatan keren.**

------------------------------------------------------------------------

# ✅ Production Checklist

Sebelum buka RAMDNS ke public Internet:

-   [ ] Firewall sudah benar
-   [ ] SSH sudah dibatasi
-   [ ] TLS certificate valid
-   [ ] Certificate renewal otomatis
-   [ ] Hostname DoT sesuai certificate
-   [ ] Hostname DoH sesuai certificate
-   [ ] Rate limiting aktif
-   [ ] Adlist berhasil update
-   [ ] Upstream DoT tervalidasi
-   [ ] DNSSEC sudah dites
-   [ ] UDP 53 sudah dites
-   [ ] TCP 53 sudah dites
-   [ ] DoT 853 sudah dites
-   [ ] DoH 443 sudah dites
-   [ ] Logging/monitoring tersedia
-   [ ] Resource limit sudah diperiksa
-   [ ] Amplification protection sudah direview
-   [ ] IPv6 sudah direview
-   [ ] Certificate expiry sudah dimonitor

------------------------------------------------------------------------

# 🟢 Status Sekarang

``` text
Core DNS             🟢
UDP :53              🟢
TCP :53              🟢
DoT :853             🟢
DoH :443             🟢
DNSSEC               🟢
Multi-source adlist  🟢
266K+ rules          🟢
Persistent DoT       🟢
Cache                🟢
SingleFlight         🟢
Rate limiting        🟢
systemd hardening    🟢
Control API :8080    🔴 Dihapus
5K concurrent test   🟢
```

------------------------------------------------------------------------

# 🤝 Kontribusi

Pull request dan issue dipersilakan.

Kalau mau nambah fitur:

**test dulu.**

Kalau mau tuning:

**benchmark dulu.**

Kalau mau buka port:

**tanya dulu port itu beneran perlu atau cuma pengen punya.** 😂

------------------------------------------------------------------------

# 📜 Lisensi

Lihat file `LICENSE` di repository untuk informasi lisensi.

------------------------------------------------------------------------

## RAMDNS 🚀

**Cepat. Terenkripsi. Ngeblok sampah. Irit resource.**

DNS server yang nggak perlu banyak gaya untuk melakukan satu pekerjaan
dengan benar:

> **client nanya, RAMDNS jawab.**
