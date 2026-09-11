# RAMDNS 🧠⚡

RAMDNS adalah public DNS resolver yang dibangun pakai Go.

Tujuannya simpel:

> DNS harus cepat, aman, simpel, dan gak butuh 47 service cuma buat jawab `example.com`.

RAMDNS fokus ke performa, keamanan, dan reliability. Gak ada dashboard, gak ada database, gak ada control-plane ribet.

Cuma resolver DNS yang kerja, diem, lalu kerja lagi. 🗿

---

## 🚀 Fitur

- ⚡ DNS UDP/TCP pada port `53`
- 🔐 DNSSEC validation
- 🛡️ DNS-over-TLS (DoT) pada port `853`
- 🌐 DNS-over-HTTPS (DoH) pada port `443`
- 🚫 Ad & tracker blocking
- 🔄 Automatic adblock update
- 🧯 Fail-safe saat update adblock gagal
- 🚦 Rate limiting berdasarkan IP
- 🔒 DNS upstream melalui TLS
- 🧱 systemd security hardening
- ⚛️ Atomic adblock snapshot
- 🪶 Tanpa database
- 🪶 Tanpa dashboard
- 🪶 Tanpa komponen yang gak diperlukan

Gambaran sederhananya:

```
Client
  ↓
RAMDNS
  ↓
Filter
  ↓
DNSSEC
  ↓
Secure Upstream
  ↓
Internet
```

---

## 🧠 Kenapa RAMDNS?

Karena kadang kita cuma butuh DNS resolver.

Bukan:

```
DNS
+ Dashboard
+ Database
+ REST API
+ Redis
+ Message Queue
+ Kubernetes
+ 17 container
+ monitoring dashboard buat monitoring dashboard
```

RAMDNS dibuat sesimpel mungkin.

Lebih sedikit komponen berarti lebih sedikit yang bisa rusak, lebih sedikit yang harus dirawat, dan lebih kecil attack surface-nya.

Big brain architecture. 🗿

---

## 🏗️ Arsitektur

```
                        INTERNET
                        │
                        ▼
                ┌───────────────┐
                │    RAMDNS     │
                │               │
                │ DNS Resolver  │
                │ Rate Limiter  │
                │ Adblock Filter│
                │ DNSSEC        │
                └───────┬───────┘
                        │
                        ▼
                 DNS-over-TLS
                   UPSTREAM
                  /         \
                 ▼           ▼
            Cloudflare     Google
               DNS           DNS
```

Client bisa terhubung melalui:

| Port | Protokol |
|------|----------|
| UDP 53 | DNS |
| TCP 53 | DNS |
| TCP 853 | DNS-over-TLS |
| TCP 443 | DNS-over-HTTPS |

---

## 📡 Port

| Port | Protokol | Fungsi |
|------|----------|--------|
| 53   | UDP | DNS |
| 53   | TCP | DNS |
| 853  | TCP | DNS-over-TLS |
| 443  | TCP | DNS-over-HTTPS |

Gak ada port dashboard.

Gak ada port API.

Gak ada port "cuma buat health check yang akhirnya lupa ditutup". 😭

---

## 🛠️ Kebutuhan

RAMDNS membutuhkan:

- Linux
- Go `1.26+`
- systemd
- Koneksi internet
- Hak akses root untuk deployment

Clone repository:

```bash
git clone git@github.com:rmdnl/ramdns.git
cd ramdns
```

---

## 🔨 Build

Sinkronkan dependency:

```bash
go mod tidy
```

Build:

```bash
go build -o ramdns ./cmd/ramdns
```

Test:

```bash
go test ./...
```

Kalau `go test` merah, jangan langsung nyalahin server.

Kemungkinan besar kodenya emang lagi ngambek. 😭

---

## ▶️ Menjalankan RAMDNS

Binary production:

```
/opt/ramdns/ramdns
```

Nama service:

```
ramdns.service
```

Cek status:

```bash
sudo systemctl status ramdns
```

Start:

```bash
sudo systemctl start ramdns
```

Stop:

```bash
sudo systemctl stop ramdns
```

Restart:

```bash
sudo systemctl restart ramdns
```

Aktifkan saat boot:

```bash
sudo systemctl enable ramdns
```

Lihat log:

```bash
sudo journalctl -u ramdns -f
```

---

## 🌍 DNS Upstream

RAMDNS menggunakan DNS-over-TLS untuk komunikasi dengan server upstream.

Deployment saat ini:

| Upstream | Hostname |
|----------|----------|
| 1.1.1.1:853 | cloudflare-dns.com |
| 8.8.8.8:853 | dns.google |

Konfigurasi:

```ini
[Service]
Environment="RAMDNS_UPSTREAMS=1.1.1.1:853|cloudflare-dns.com,8.8.8.8:853|dns.google"
```

Kenapa pakai TLS?

Karena query DNS ke upstream gak perlu jalan-jalan dalam keadaan kosong. 🔐

Hostname upstream juga digunakan untuk verifikasi identitas server berdasarkan sertifikat TLS.

---

## 🚫 Adblock

RAMDNS punya adblock bawaan.

Sumber default:

```
https://cdn.jsdelivr.net/gh/hagezi/dns-blocklists@latest/wildcard/multi-onlydomains.txt
```

Konfigurasi:

```ini
[Service]
Environment="RAMDNS_ADLIST_URL=https://cdn.jsdelivr.net/gh/hagezi/dns-blocklists@latest/wildcard/multi-onlydomains.txt"
```

---

## 🔄 Automatic Adblock Update

RAMDNS gak cuma download blocklist sekali terus pura-pura lupa.

Worker adblock melakukan:

```
Service Start
     │
     ▼
Download Blocklist
     │
     ▼
Parse Rules
     │
     ▼
Compile Rules
     │
     ▼
Atomic Swap
     │
     ▼
Wait 6 Hours
     │
     └──────────► Repeat
```

Update dilakukan:

- Saat startup
- Setiap 6 jam

---

## 🧯 Fail-safe Adblock

Ini bagian penting.

Misalnya source blocklist lagi:

- down
- timeout
- error
- 404
- server ngambek

RAMDNS gak akan menghapus blocklist yang sedang aktif.

Alurnya:

```
Blocklist Lama
      │
      ▼
Download Baru
      │
      ├── Gagal ──────► Tetap pakai blocklist lama
      │
      ▼
Parse + Compile
      │
      ▼
Atomic Swap
```

Jadi kalau internet lagi batuk, adblock gak ikut mati.

---

## 📋 Format Blocklist

RAMDNS mendukung beberapa format umum.

**Format Hosts**

```
0.0.0.0 example.com
127.0.0.1 example.org
:: example.net
::1 example.test
```

**Sintaks Adblock**

```
||example.com^
```

**Domain Biasa**

```
example.com
ads.example.org
tracker.example.net
```

Komentar dan baris kosong akan diabaikan.

---

## 🧬 Domain Matching

Kalau:

```
example.com
```

diblokir, domain turunannya juga ikut kena:

```
example.com
www.example.com
ads.example.com
tracker.ads.example.com
```

Jadi gak perlu masukin satu-satu kayak daftar mantan. 😭

RAMDNS melakukan pencocokan terhadap domain dan parent domain.

---

## ⚛️ Atomic Filter Update

Adblock menggunakan snapshot.

Ketika blocklist baru selesai diproses:

```
Old Snapshot
     │
     │ DNS queries tetap jalan
     │
     ▼
New Snapshot
     │
     ▼
Atomic Swap
```

Resolver gak perlu restart.

DNS query juga gak perlu antre nunggu blocklist selesai di-download.

Jadi worker adblock boleh sibuk, resolver tetap kerja.

---

## 🔐 DNSSEC

RAMDNS melakukan validasi DNSSEC.

Tes:

```bash
dig @127.0.0.1 cloudflare.com A +dnssec
```

Cari flag:

```
ad
```

Flag `ad` menunjukkan response sudah tervalidasi DNSSEC.

---

## 🔒 DNS-over-TLS

RAMDNS menyediakan DoT pada:

```
TCP 853
```

Hostname:

```
dot.ramdns.my.id
```

TLS minimum:

```
TLS 1.3
```

Certificate:

```
/etc/ramdns/tls/fullchain.pem
```

Private key:

```
/etc/ramdns/tls/privkey.pem
```

Tes TLS:

```bash
openssl s_client \
  -connect dot.ramdns.my.id:853 \
  -servername dot.ramdns.my.id
```

Tes DNS-over-TLS:

```bash
kdig @dot.ramdns.my.id example.com +tls
```

Kalau muncul:

```
Verify return code: 0 (ok)
```

TLS-nya lagi gak drama. 👍

---

## 🌐 DNS-over-HTTPS

RAMDNS menyediakan DoH pada:

```
TCP 443
```

Endpoint:

```
/dns-query
```

Hostname:

```
doh.ramdns.my.id
```

DoH menggunakan format DNS wire-format melalui HTTP.

Contoh:

```bash
curl \
  --http2 \
  -H 'Content-Type: application/dns-message' \
  --data-binary @query.bin \
  https://doh.ramdns.my.id/dns-query
```

Response yang diharapkan:

```
HTTP 200
Content-Type: application/dns-message
```

---

## 🚦 Rate Limiting

RAMDNS punya rate limiter berdasarkan IP client.

Konfigurasi default:

| Parameter | Nilai |
|-----------|-------|
| Rate | 50 request/detik/IP |
| Burst | 100 |
| Tracked IP | 20.000 |
| Cleanup | 5 menit |

Tujuannya membantu mengurangi:

- DNS flood
- Abuse
- Resource exhaustion
- Query spam
- Satu client yang tiba-tiba merasa server ini milik neneknya

Rate limiting bukan pengganti firewall atau perlindungan DDoS jaringan.

---

## 🧱 Security Hardening

Service RAMDNS menggunakan systemd hardening.

Konfigurasi utama:

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

RAMDNS hanya diberikan capability:

```
CAP_NET_BIND_SERVICE
```

Capability ini diperlukan untuk bind ke port privileged seperti:

- 53
- 443
- 853

Capability lain tidak diberikan.

Prinsipnya:

- Butuh permission? → kasih seperlunya.
- Gak perlu? → jangan dikasih.

Least privilege, bukan "kasih root aja biar gampang". 🗿

---

## 🔥 Firewall

Contoh UFW:

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing

sudo ufw allow 22/tcp
sudo ufw allow 53/tcp
sudo ufw allow 53/udp
sudo ufw allow 853/tcp
sudo ufw allow 443/tcp

sudo ufw enable
```

Cek:

```bash
sudo ufw status verbose
```

Jangan buka port yang gak diperlukan.

Internet bukan LAN pribadi lu. 😭

---

## 🧪 Pengujian

**DNS UDP**

```bash
dig @127.0.0.1 example.com A
```

**DNS TCP**

```bash
dig @127.0.0.1 example.com A +tcp
```

**DNSSEC**

```bash
dig @127.0.0.1 cloudflare.com A +dnssec
```

**Cek Listener**

```bash
sudo ss -lntup
```

Listener utama:

```
*:53
*:853
*:443
```

---

## 🧪 Tes Adblock

Gunakan domain yang memang ada di blocklist aktif.

Contoh:

```bash
dig @127.0.0.1 ad.001zb.com A +short
```

Jika diblokir, resolver tidak akan memberikan IP normal.

Cek log:

```bash
sudo journalctl -u ramdns -f
```

Contoh:

```
blocked query=ad.001zb.com.
```

Kalau muncul begitu:

```
RAMDNS: 1
Iklan: 0
```

🗿

---

## 📊 Monitoring

Status service:

```bash
systemctl status ramdns
```

Resource:

```bash
top
# atau
htop
```

Port:

```bash
sudo ss -lntup
```

Log realtime:

```bash
sudo journalctl -u ramdns -f
```

Log satu jam terakhir:

```bash
sudo journalctl -u ramdns --since "1 hour ago"
```

Cari aktivitas adblock:

```bash
sudo journalctl -u ramdns --since "1 hour ago" | grep adlist
```

---

## 🩺 Troubleshooting

### RAMDNS gak mau start

Cek:

```bash
sudo systemctl status ramdns
```

Kemudian:

```bash
sudo journalctl -u ramdns -n 100 --no-pager
```

Biasanya error-nya bakal ngomong sendiri.

Kalau masih bingung, baca pelan-pelan.

Server gak bisa baca pikiran. 😭

### Port 53 bentrok

Cek:

```bash
sudo ss -lntup | grep ':53'
```

Kalau `systemd-resolved` masih menggunakan stub listener, konfigurasi:

```ini
[Resolve]
DNSStubListener=no
```

Kemudian:

```bash
sudo systemctl restart systemd-resolved
```

### DoT error

Cek sertifikat:

```bash
openssl x509 \
  -in /etc/ramdns/tls/fullchain.pem \
  -noout \
  -subject \
  -issuer \
  -dates \
  -ext subjectAltName
```

Tes:

```bash
openssl s_client \
  -connect dot.ramdns.my.id:853 \
  -servername dot.ramdns.my.id
```

### DoH error

Cek port:

```bash
sudo ss -lntp | grep ':443'
```

Cek log:

```bash
sudo journalctl -u ramdns -n 100 --no-pager
```

### Adblock gak update

Cek environment:

```bash
sudo systemctl show ramdns --property=Environment
```

Cek log:

```bash
sudo journalctl -u ramdns --since "1 hour ago"
```

Cari:

```
adlist
```

Kalau update gagal, RAMDNS seharusnya tetap memakai snapshot sebelumnya.

---

## 📁 Struktur Proyek

```
ramdns/
├── cmd/
│   └── ramdns/
│       └── main.go
│
├── internal/
│   ├── adlist/
│   │   ├── manager.go
│   │   ├── parser.go
│   │   └── worker.go
│   │
│   ├── filter/
│   │   └── filter.go
│   │
│   └── resolver/
│       └── ...
│
├── go.mod
├── go.sum
├── README.md
└── ...
```

Komponen dashboard, API pengelolaan, dan database sudah tidak menjadi bagian dari arsitektur resolver.

Karena:

> DNS resolver ≠ ERP perusahaan

---

## ⚡ Performa

RAMDNS dirancang untuk:

- Latensi rendah
- Concurrent request
- Penggunaan RAM rendah
- Jalur query sederhana
- Dependency seminimal mungkin
- Overhead rendah

Pengujian lokal menunjukkan:

- DNS query berhasil dengan latency sekitar milidetik
- Concurrent query dapat diproses
- Penggunaan resource relatif kecil
- Update blocklist tidak membutuhkan restart
- Snapshot filter dapat diganti secara atomik

Untuk benchmark serius, gunakan tool khusus DNS seperti:

- `dnsperf`
- `resperf`
- `kdig`
- `dig`

Jangan menjadikan:

```bash
for i in {1..1000}; do dig ...; done
```

sebagai benchmark QPS yang sakral.

Itu juga ngukur overhead bikin proses baru. 😭

---

## 🧠 Prinsip Desain

1. **Simpel** — Komponen lebih sedikit berarti lebih gampang dirawat, lebih gampang di-debug, attack surface lebih kecil.
2. **Cepat** — Jalur query DNS dibuat sesingkat mungkin.
3. **Atomik** — Blocklist dapat diperbarui tanpa restart resolver.
4. **Fail-safe** — Kalau source blocklist mati, snapshot lama tetap dipakai.
5. **Least Privilege** — Service cuma dikasih capability yang diperlukan.
6. **Stateless** — Operasi DNS normal tidak membutuhkan database.
7. **Observable** — Log cukup jelas untuk melihat:
   - Status service
   - Update blocklist
   - Domain yang diblokir
   - Error runtime

---

## 🛡️ Catatan Keamanan

RAMDNS adalah public DNS resolver.

Artinya server tetap bisa menjadi target:

- DNS flood
- Query abuse
- Connection exhaustion
- TLS handshake abuse
- HTTP abuse
- Network scanning
- Resource exhaustion
- DDoS

Dan perlu ditegaskan:

> Gak ada public server yang bisa dijamin 100% kebal DDoS.

RAMDNS sudah punya beberapa lapisan pertahanan:

```
Firewall
   ↓
systemd hardening
   ↓
Rate limiting
   ↓
DNS resolver
   ↓
Secure upstream
```

Tapi kalau serangannya sudah level jaringan besar, aplikasi DNS doang gak bisa tiba-tiba berubah jadi superhero. 🗿

Perlindungan jaringan tambahan tetap diperlukan jika skala deployment memang membutuhkannya.

---

## ✅ Checklist Production

Sebelum dianggap siap tempur:

- [ ] UDP 53 aktif
- [ ] TCP 53 aktif
- [ ] DoT 853 aktif
- [ ] DoH 443 aktif
- [ ] Sertifikat TLS valid
- [ ] TLS 1.3 aktif
- [ ] DNSSEC aktif
- [ ] Upstream DNS-over-TLS aktif
- [ ] Adblock aktif
- [ ] Adblock update berjalan
- [ ] Rate limiting aktif
- [ ] UFW aktif
- [ ] systemd hardening aktif
- [ ] Log normal
- [ ] Resource usage normal

Kalau semuanya centang:

**RAMDNS READY 🚀**

---

## 🔍 Pemeriksaan Cepat

```bash
systemctl is-active ramdns

sudo ss -lntup | grep -E ':(53|443|853)\b'

dig @127.0.0.1 example.com A

dig @127.0.0.1 cloudflare.com A +dnssec

sudo journalctl -u ramdns --since "10 minutes ago" --no-pager
```

---

## 📦 Deployment Saat Ini

| Komponen | Nilai |
|----------|-------|
| Binary | `/opt/ramdns/ramdns` |
| Service | `ramdns.service` |
| DNS | UDP 53, TCP 53 |
| DNS terenkripsi | TCP 853, TCP 443 |
| DoT | `dot.ramdns.my.id` |
| DoH | `doh.ramdns.my.id` |
| Upstream | `1.1.1.1:853`, `8.8.8.8:853` |
| Adblock | HaGeZi Multi OnlyDomains |
| Update adblock | Setiap 6 jam |
| TLS | TLS 1.3 |

---

## 🧹 Kenapa Tanpa Dashboard?

Karena RAMDNS sekarang fokus menjadi resolver.

Bukan:

```
Dashboard → API → Database → Resolver
```

Tapi:

```
Client
  ↓
RAMDNS
  ↓
Done.
```

Konfigurasi dilakukan melalui systemd dan environment variable.

Lebih sedikit moving parts.

Lebih sedikit drama.

---

## 📚 Repository

Source code: [github.com/rmdnl/ramdns](https://github.com/rmdnl/ramdns)

---

## 📄 Lisensi

Lisensi RAMDNS mengikuti file `LICENSE` di repository.

---

## 🏁 Status

RAMDNS saat ini menyediakan:

- Public DNS Resolver
- DNSSEC
- DNS-over-TLS
- DNS-over-HTTPS
- Ad & Tracker Blocking
- Automatic Blocklist Update
- Rate Limiting
- Secure Upstream
- systemd Hardening

**Dashboard?** Tidak ada.

**Database?** Tidak perlu.

**Drama?** Diusahakan seminimal mungkin.

---

## RAMDNS

DNS yang kerjaannya cuma satu: jawab DNS.

Cepat. Aman. Simpel.

Dan kalau bisa, jangan bikin server nangis. 🗿⚡
