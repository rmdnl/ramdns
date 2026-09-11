RAMDNS

DNS Resolver publik berperforma tinggi yang dibuat menggunakan Go.

RAMDNS dirancang sebagai DNS data plane yang ringan, cepat, dan memiliki attack surface yang kecil. RAMDNS menyediakan DNS standar, DNS-over-TLS (DoT), DNS-over-HTTPS (DoH), cache dalam memori, dukungan DNSSEC, rate limiting per-IP, serta pemblokiran iklan dan tracker secara otomatis.

RAMDNS sengaja tidak menggunakan dashboard, database, maupun API management agar jalur resolusi DNS tetap sederhana dan ringan.

Fitur

- DNS UDP pada port "53"
- DNS TCP pada port "53"
- DNS-over-TLS pada port "853"
- DNS-over-HTTPS pada port "443"
- TLS 1.3 untuk DNS terenkripsi
- Upstream DNS melalui DNS-over-TLS
- Cache DNS dalam memori
- Dukungan DNSSEC
- Dukungan EDNS
- Ukuran payload EDNS sekitar "1232"
- Filter iklan dan tracker dalam memori
- Update adblock otomatis
- Conditional HTTP request menggunakan ETag / Last-Modified
- Update adblock fail-safe
- Rate limiting berdasarkan IP client
- Graceful shutdown
- Hardening dan sandbox systemd
- Tanpa SQLite
- Tanpa dashboard
- Tanpa control API
- Tidak membutuhkan database untuk operasi DNS

Arsitektur

                         Internet
                            │
            ┌───────────────┼────────────────┐
            │               │                │
          DNS 53          DoT 853          DoH 443
        UDP / TCP           TLS              HTTPS
            │               │                │
            └───────────────┼────────────────┘
                            │
                     ┌──────▼──────┐
                     │   RAMDNS    │
                     │             │
                     │ Rate Limit  │
                     │   Filter    │
                     │    Cache    │
                     │  Resolver   │
                     └──────┬──────┘
                            │
                    Upstream DNS-over-TLS
                            │
                    ┌───────┴────────┐
                    │                │
                 Cloudflare       Google
                    DoT             DoT

Alur adblock:

HaGeZi
   │
   ▼
Download HTTPS
   │
   ▼
Parser
   │
   ▼
Compiler
   │
   ▼
Penggantian filter secara atomic
   │
   ▼
Query DNS yang masuk diblokir

Port

Port| Protokol| Fungsi
"53"| UDP| DNS standar
"53"| TCP| DNS standar / fallback
"853"| TCP| DNS-over-TLS
"443"| TCP| DNS-over-HTTPS

RAMDNS tidak membutuhkan port management untuk operasi normal.

Kebutuhan

- Linux
- Go 1.26+
- systemd
- Akses jaringan ke upstream DNS
- Sertifikat TLS untuk DoT/DoH
- Hak akses root saat instalasi dan konfigurasi service

Build

Clone repository:

git clone git@github.com:rmdnl/ramdns.git
cd ramdns

Build:

go mod tidy
go test ./...
go build -o ramdns ./cmd/ramdns

Karena RAMDNS menggunakan port privileged, berikan capability:

sudo setcap cap_net_bind_service=ep ./ramdns

Cek:

getcap ./ramdns

Hasil yang diharapkan:

./ramdns cap_net_bind_service=ep

Konfigurasi

RAMDNS menggunakan environment variable untuk konfigurasi runtime.

Upstream DNS

Contoh:

Environment="RAMDNS_UPSTREAMS=1.1.1.1:853|cloudflare-dns.com,8.8.8.8:853|dns.google"

RAMDNS berkomunikasi dengan upstream menggunakan DNS-over-TLS.

Adblock

Source adblock dikonfigurasi melalui:

Environment="RAMDNS_ADLIST_URL=https://example.com/list.txt"

RAMDNS menggunakan satu source adblock.

Contoh konfigurasi:

Environment="RAMDNS_ADLIST_URL=https://cdn.jsdelivr.net/gh/hagezi/dns-blocklists@latest/wildcard/multi-onlydomains.txt"

Adblock dimuat saat RAMDNS startup dan diperbarui otomatis setiap 6 jam.

Tidak perlu rebuild binary saat mengganti URL adblock.

Mengganti Source Adblock

Edit:

sudo nano /etc/systemd/system/ramdns.service.d/adlist.conf

Contoh:

[Service]
Environment="RAMDNS_ADLIST_URL=https://example.com/list-baru.txt"

Kemudian:

sudo systemctl daemon-reload
sudo systemctl restart ramdns

Cek hasil update:

sudo journalctl -u ramdns --since "2 minutes ago" --no-pager -o cat | grep adlist

Contoh berhasil:

adlist update successful url=https://example.com/list-baru.txt rules=180000

Update Adblock Otomatis

Worker adblock melakukan proses berikut:

1. Memuat list saat RAMDNS startup.
2. Mengunduh list melalui HTTPS.
3. Mem-parsing format domain.
4. Mengompilasi rules.
5. Mengganti filter aktif secara atomic.
6. Mengulangi proses setiap 6 jam.

Jika server source mendukung ETag atau Last-Modified, RAMDNS menggunakan conditional request agar tidak selalu mengunduh data penuh.

Jika update gagal, rules lama tetap aktif.

Dengan demikian, kegagalan source adblock tidak menyebabkan sistem kehilangan filter yang sedang digunakan.

Format Adblock yang Didukung

Domain biasa

ads.example.com
tracker.example.net

Format hosts

0.0.0.0 ads.example.com
127.0.0.1 tracker.example.net

Format adblock

||ads.example.com^

Komentar dan baris kosong diabaikan.

Cara Kerja Blocking

Rules domain disimpan di memori dan diperiksa sebelum cache maupun upstream DNS.

Contoh rule:

ads.example.com

akan memblokir:

ads.example.com
foo.ads.example.com
bar.foo.ads.example.com

Query yang diblokir akan mendapatkan response DNS "NXDOMAIN".

Contoh log:

blocked query=ads.example.com.

DNS-over-TLS

RAMDNS menyediakan DoT pada:

TCP/853

TLS 1.3 digunakan untuk koneksi DNS terenkripsi.

Contoh konfigurasi client:

Server       : dot.example.com
Port         : 853
TLS hostname : dot.example.com

Hostname harus sesuai dengan sertifikat TLS.

Pengujian:

openssl s_client \
  -connect dot.example.com:853 \
  -servername dot.example.com \
  -tls1_3

Hasil yang diharapkan:

Protocol  : TLSv1.3
Verify return code: 0 (ok)

DNS-over-HTTPS

RAMDNS menyediakan DoH pada:

HTTPS/443

Endpoint:

/dns-query

Format standar yang digunakan:

application/dns-message

Request ke "/" dapat menghasilkan:

HTTP 404

Hal tersebut normal jika endpoint DoH berada di "/dns-query".

Request DNS yang valid ke "/dns-query" harus menghasilkan:

HTTP 200
Content-Type: application/dns-message

Rate Limiting

RAMDNS menggunakan rate limiting berdasarkan IP client.

Konfigurasi saat ini:

50 request/detik per IP
Burst: 100 request
Maksimum IP yang dilacak: 20.000

Tujuannya adalah mengurangi abuse tanpa mengganggu penggunaan DNS normal.

State rate limiter disimpan di memori.

Keamanan

RAMDNS dirancang dengan attack surface seminimal mungkin.

Service systemd menggunakan beberapa pembatasan keamanan:

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

RAMDNS berjalan sebagai user yang tidak memiliki hak root.

Firewall sebaiknya hanya membuka port yang diperlukan.

Port publik RAMDNS:

53/tcp
53/udp
853/tcp
443/tcp

Port SSH harus disesuaikan dengan kebutuhan administrasi server.

Contoh Firewall UFW

sudo ufw default deny incoming
sudo ufw default allow outgoing

sudo ufw allow 22/tcp
sudo ufw allow 53/tcp
sudo ufw allow 53/udp
sudo ufw allow 853/tcp
sudo ufw allow 443/tcp

sudo ufw --force enable

Pastikan akses SSH sudah aman sebelum mengaktifkan firewall pada VPS remote.

Service systemd

Contoh service:

[Unit]
Description=RAMDNS Personal DNS Resolver
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=ubuntu
Group=ubuntu
WorkingDirectory=/opt/ramdns
ExecStart=/opt/ramdns/ramdns
Restart=always
RestartSec=2

[Install]
WantedBy=multi-user.target

Aktifkan:

sudo systemctl daemon-reload
sudo systemctl enable --now ramdns

Cek:

sudo systemctl status ramdns

Pemeriksaan Dasar

Cek service:

sudo systemctl is-active ramdns

Cek port:

sudo ss -lntup | grep -E ':(53|443|853)\b'

Hasil yang diharapkan:

*:53
*:443
*:853

Tes DNS:

dig @127.0.0.1 google.com A +short

Tes DNSSEC:

dig @127.0.0.1 cloudflare.com A +dnssec

Cari:

status: NOERROR
flags: qr rd ra ad

Pengujian Adblock

Ambil domain yang ada di source adblock kemudian:

dig @127.0.0.1 ad.example.com A +short

Domain yang diblokir tidak boleh mendapatkan IP normal.

Cek log:

sudo journalctl -u ramdns --since "5 minutes ago" --no-pager -o cat | grep 'blocked query'

Contoh:

blocked query=ad.example.com.

Performa

RAMDNS menjaga jalur resolusi DNS tetap berada di memori dan tidak menggunakan database pada setiap query.

Jalur utama:

Client
  ↓
Rate Limiter
  ↓
Adblock Filter
  ↓
Cache
  ↓
Single-flight
  ↓
Upstream DNS-over-TLS

Pendekatan ini mengurangi operasi storage dan network yang tidak diperlukan.

Tes latency sederhana:

for i in 1 2 3 4 5; do
    /usr/bin/time -f '%e s' \
        dig @127.0.0.1 google.com A +short >/dev/null
done

Untuk benchmark produksi, gunakan traffic generator dari jaringan eksternal agar hasil tidak tercampur dengan performa loopback VPS.

Monitoring

Lihat log:

sudo journalctl -u ramdns --since "10 minutes ago" --no-pager -o cat

Follow log:

sudo journalctl -u ramdns -f

Cek penggunaan resource:

ps -o pid,user,%cpu,%mem,rss,vsz,cmd -C ramdns

Cek socket:

sudo ss -s

Troubleshooting

RAMDNS tidak mau start

sudo systemctl status ramdns
sudo journalctl -u ramdns -n 100 --no-pager

Port 53 sudah digunakan

sudo ss -lntup | grep ':53'

Cari service lain yang menggunakan port tersebut.

Sertifikat DoT bermasalah

openssl s_client \
  -connect dot.example.com:853 \
  -servername dot.example.com \
  -tls1_3

Pastikan:

- sertifikat masih valid
- hostname terdapat pada sertifikat
- file certificate dapat dibaca RAMDNS
- file private key dapat dibaca RAMDNS
- TCP/853 dibuka firewall

DoH menghasilkan 404

Request ke "/" dapat menghasilkan "404".

Gunakan endpoint:

https://dot.example.com/dns-query

dengan DNS wire-format dan:

Content-Type: application/dns-message

Adblock gagal update

Cek:

sudo journalctl -u ramdns --since "1 hour ago" --no-pager -o cat | grep adlist

Jika update gagal, rules lama tetap digunakan.

Cek URL:

sudo systemctl cat ramdns | grep RAMDNS_ADLIST_URL

Domain tidak terblokir

Pastikan domain tersebut memang ada di source adblock.

Cek update:

sudo journalctl -u ramdns --since "1 hour ago" --no-pager -o cat | grep adlist

Kemudian:

sudo journalctl -u ramdns --since "5 minutes ago" --no-pager -o cat | grep 'blocked query'

Struktur Project

ramdns/
├── cmd/
│   └── ramdns/
│       └── main.go
├── internal/
│   ├── adlist/
│   │   ├── compiler.go
│   │   ├── downloader.go
│   │   ├── manager.go
│   │   ├── parser.go
│   │   └── worker.go
│   ├── cache/
│   ├── dns/
│   ├── doh/
│   ├── filter/
│   ├── metrics/
│   ├── ratelimit/
│   └── upstream/
├── go.mod
└── go.sum

Prinsip Desain

Data plane tetap kecil

Resolusi DNS tidak bergantung pada dashboard, database, atau management API.

Prioritaskan memori

Cache dan filter disimpan di memori agar latency tetap rendah.

Update adblock fail-safe

Rules baru hanya menggantikan rules lama setelah berhasil di-download dan dikompilasi.

Minimalkan privilege

RAMDNS berjalan sebagai user non-root dengan pembatasan systemd.

Minimalkan dependency

Semakin sedikit komponen, semakin kecil kompleksitas operasional dan attack surface.

Ukur sebelum tuning

Tuning dilakukan berdasarkan hasil pengujian, bukan sekadar menaikkan limit secara acak.

Deployment Saat Ini

Konfigurasi deployment:

DNS:
  UDP/TCP 53

DoT:
  TCP 853
  TLS 1.3

DoH:
  TCP 443
  HTTP/2
  application/dns-message

Adblock:
  HaGeZi Multi
  1 source
  Update otomatis setiap 6 jam

Upstream:
  Cloudflare DNS-over-TLS
  Google DNS-over-TLS

Proteksi:
  Rate limiting per-IP
  UFW firewall
  systemd hardening

Catatan Keamanan

RAMDNS dirancang untuk mengurangi attack surface dan menangani abuse umum, tetapi tidak ada layanan publik di Internet yang dapat dijamin 100% kebal terhadap serangan atau DDoS.

Untuk deployment dengan trafik sangat besar, perlindungan DDoS di level jaringan, filtering upstream, kapasitas bandwidth, monitoring, dan capacity planning tambahan mungkin diperlukan.

Lisensi

Lihat file lisensi pada repository untuk informasi lisensi terbaru.

Status

RAMDNS ditujukan sebagai DNS resolver publik yang ringan dengan dukungan DNS terenkripsi dan pemblokiran domain otomatis.

Management functionality sengaja tidak ditempatkan di dalam DNS data plane agar sistem tetap sederhana, cepat, dan mudah diamankan.
