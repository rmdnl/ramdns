RAMDNS

High-performance, security-focused public DNS resolver written in Go.

RAMDNS is designed as a lightweight DNS data plane with a small attack surface. It provides standard DNS, DNS-over-TLS, DNS-over-HTTPS, in-memory caching, DNSSEC-aware resolution, per-client rate limiting, and automatic ad/tracker blocking.

The project intentionally avoids unnecessary control-plane components such as dashboards, databases, and management APIs.

Features

- DNS over UDP on port "53"
- DNS over TCP on port "53"
- DNS-over-TLS (DoT) on port "853"
- DNS-over-HTTPS (DoH) on port "443"
- TLS 1.3 for encrypted DNS
- Upstream DNS-over-TLS
- In-memory DNS cache
- DNSSEC support
- EDNS support
- EDNS UDP payload size optimized around "1232"
- In-memory ad/tracker filtering
- Automatic adblock list updates
- Conditional HTTP requests using ETag / Last-Modified
- Fail-safe adblock updates
- Per-client IP rate limiting
- Graceful shutdown
- systemd sandboxing and privilege restrictions
- No SQLite dependency
- No dashboard
- No control API
- No persistent database required for DNS operation

Architecture

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

The adblock list is downloaded independently and compiled into an in-memory filter.

HaGeZi list
     │
     ▼
 HTTPS download
     │
     ▼
 Parser
     │
     ▼
 Compiler
     │
     ▼
 Atomic filter replacement
     │
     ▼
 DNS queries blocked in memory

Ports

Port| Protocol| Purpose
"53"| UDP| Standard DNS
"53"| TCP| Standard DNS fallback
"853"| TCP| DNS-over-TLS
"443"| TCP| DNS-over-HTTPS

RAMDNS does not require a management port for normal operation.

Requirements

- Linux
- Go 1.26+
- systemd
- Network access to upstream DNS servers
- TLS certificate for DoT/DoH
- Root privileges during installation and service configuration

Build

Clone the repository:

git clone git@github.com:rmdnl/ramdns.git
cd ramdns

Build:

go mod tidy
go test ./...
go build -o ramdns ./cmd/ramdns

Because RAMDNS binds to privileged ports, the binary can be given the required capability:

sudo setcap cap_net_bind_service=ep ./ramdns

Verify:

getcap ./ramdns

Expected:

./ramdns cap_net_bind_service=ep

Configuration

RAMDNS uses environment variables for runtime configuration.

Upstream DNS

Example:

Environment="RAMDNS_UPSTREAMS=1.1.1.1:853|cloudflare-dns.com,8.8.8.8:853|dns.google"

The upstream resolvers are accessed through DNS-over-TLS.

Adblock

Configure the adblock source with:

Environment="RAMDNS_ADLIST_URL=https://example.com/list.txt"

RAMDNS currently uses one adblock source.

Example:

Environment="RAMDNS_ADLIST_URL=https://cdn.jsdelivr.net/gh/hagezi/dns-blocklists@latest/wildcard/multi-onlydomains.txt"

The list is loaded when RAMDNS starts and automatically refreshed every 6 hours.

No binary rebuild is required when changing the list URL.

Changing the Adblock Source

Edit:

sudo nano /etc/systemd/system/ramdns.service.d/adlist.conf

Example:

[Service]
Environment="RAMDNS_ADLIST_URL=https://example.com/new-list.txt"

Reload systemd and restart RAMDNS:

sudo systemctl daemon-reload
sudo systemctl restart ramdns

Verify:

sudo journalctl -u ramdns --since "2 minutes ago" --no-pager -o cat | grep adlist

A successful update looks like:

adlist update successful url=https://example.com/new-list.txt rules=180000

Automatic Adblock Updates

The adblock worker:

1. Loads the list during startup.
2. Downloads the list over HTTPS.
3. Parses supported domain formats.
4. Compiles the rules.
5. Atomically replaces the active in-memory filter.
6. Repeats the process every 6 hours.

RAMDNS uses conditional HTTP requests when supported by the source.

If the update fails, the existing working rules remain active.

This prevents a failed download from accidentally disabling ad blocking.

Supported Adblock Formats

The parser supports common formats including:

Plain domains

ads.example.com
tracker.example.net

Hosts format

0.0.0.0 ads.example.com
127.0.0.1 tracker.example.net

Adblock-style domains

||ads.example.com^

Comments and empty lines are ignored.

Blocking Behavior

Domain rules are stored in memory and checked before cache lookup or upstream resolution.

For example:

ad.example.com

can block:

ad.example.com
foo.ad.example.com
bar.foo.ad.example.com

while unrelated domains remain unaffected.

A blocked query returns DNS "NXDOMAIN".

Example log:

blocked query=ad.example.com.

DNS-over-TLS

RAMDNS exposes DoT on:

TCP/853

TLS 1.3 is enabled for encrypted DNS.

Example client configuration:

Server: dot.example.com
Port: 853
TLS hostname: dot.example.com

The certificate must contain the hostname used by clients.

Test the TLS endpoint:

openssl s_client \
  -connect your-domain.example:853 \
  -servername your-domain.example \
  -tls1_3

A successful connection should show:

Protocol  : TLSv1.3
Verify return code: 0 (ok)

DNS-over-HTTPS

RAMDNS exposes DoH on:

HTTPS/443

The DNS endpoint is:

/dns-query

DoH uses the standard:

application/dns-message

content type.

A normal browser request to "/" may return:

HTTP 404

That does not indicate a broken DoH implementation. The DoH endpoint expects a DNS message at "/dns-query".

A valid DNS wire-format request should return:

HTTP 200
Content-Type: application/dns-message

Rate Limiting

Public DNS traffic is protected with per-client IP rate limiting.

Current configuration:

50 requests/second per IP
100 request burst
20,000 tracked IPs maximum

This is intended to reduce abuse while keeping normal DNS clients responsive.

Rate limiter state is maintained in memory.

Security

RAMDNS is designed with a minimal attack surface.

The systemd service uses security restrictions such as:

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

The service runs as an unprivileged user rather than root.

The firewall should expose only the ports required by the deployment.

Recommended public DNS ports:

53/tcp
53/udp
853/tcp
443/tcp

SSH should remain restricted according to the server administration requirements.

Firewall Example

Using UFW:

sudo ufw default deny incoming
sudo ufw default allow outgoing

sudo ufw allow 22/tcp
sudo ufw allow 53/tcp
sudo ufw allow 53/udp
sudo ufw allow 853/tcp
sudo ufw allow 443/tcp

sudo ufw --force enable

Adjust SSH access before applying firewall rules on a remote server.

systemd Service

Example:

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

Enable:

sudo systemctl daemon-reload
sudo systemctl enable --now ramdns

Check:

sudo systemctl status ramdns

Health Checks

Check service:

sudo systemctl is-active ramdns

Check listeners:

sudo ss -lntup | grep -E ':(53|443|853)\b'

Expected:

*:53
*:443
*:853

Test normal DNS:

dig @127.0.0.1 google.com A +short

Test DNSSEC:

dig @127.0.0.1 cloudflare.com A +dnssec

Look for:

status: NOERROR
flags: qr rd ra ad

Testing Adblock

Find a domain from the configured adblock source and query it:

dig @127.0.0.1 ad.example.com A +short

A blocked domain should not return its normal address.

Check the log:

sudo journalctl -u ramdns --since "5 minutes ago" --no-pager -o cat | grep 'blocked query'

Example:

blocked query=ad.example.com.

Performance

RAMDNS keeps the DNS data path in memory and avoids database access during normal DNS resolution.

The main data path is:

Client
  ↓
Rate limiter
  ↓
Adblock filter
  ↓
Cache
  ↓
Single-flight deduplication
  ↓
Encrypted upstream

This minimizes unnecessary network and storage operations.

Basic local latency can be tested with:

for i in 1 2 3 4 5; do
    /usr/bin/time -f '%e s' \
        dig @127.0.0.1 google.com A +short >/dev/null
done

For meaningful production benchmarking, use an external traffic generator rather than spawning large numbers of "dig" processes on the DNS server itself.

Monitoring

View recent RAMDNS logs:

sudo journalctl -u ramdns --since "10 minutes ago" --no-pager -o cat

Follow logs:

sudo journalctl -u ramdns -f

Check resource usage:

ps -o pid,user,%cpu,%mem,rss,vsz,cmd -C ramdns

Check sockets:

sudo ss -s

Troubleshooting

RAMDNS does not start

sudo systemctl status ramdns
sudo journalctl -u ramdns -n 100 --no-pager

Port 53 is already in use

Check:

sudo ss -lntup | grep ':53'

If "systemd-resolved" owns the DNS stub, disable its stub listener or resolve the port conflict before starting RAMDNS.

DoT certificate error

Check:

openssl s_client \
  -connect your-domain.example:853 \
  -servername your-domain.example \
  -tls1_3

Verify that:

- the certificate is valid
- the hostname is included in the certificate
- the certificate and key are readable by the RAMDNS service
- port "853/tcp" is reachable

DoH returns 404

A request to "/" can legitimately return "404".

Test the actual endpoint:

https://your-domain.example/dns-query

with a DNS wire-format request and:

Content-Type: application/dns-message

Adblock update fails

Check:

sudo journalctl -u ramdns --since "1 hour ago" --no-pager -o cat | grep adlist

The existing rules remain active when an update fails.

Verify the configured URL:

sudo systemctl cat ramdns | grep RAMDNS_ADLIST_URL

Adblock domain is not blocked

First verify that the domain exists in the configured source.

Then check the RAMDNS startup/update log:

sudo journalctl -u ramdns --since "1 hour ago" --no-pager -o cat | grep adlist

Finally check:

sudo journalctl -u ramdns --since "5 minutes ago" --no-pager -o cat | grep 'blocked query'

Project Structure

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

Design Principles

RAMDNS follows a few simple principles:

Keep the DNS data plane small

DNS resolution should not depend on a dashboard, database, or management API.

Prefer memory over disk

The resolver cache and filtering rules are kept in memory for low-latency operation.

Fail closed where appropriate

Invalid or failed adblock updates do not replace a known-good filter snapshot.

Minimize privileges

RAMDNS runs as an unprivileged service with systemd restrictions.

Avoid unnecessary dependencies

Every dependency and exposed service increases operational and security complexity.

Measure before tuning

Performance changes should be based on actual measurements rather than arbitrary configuration changes.

Current Deployment

The reference deployment uses:

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
  Automatic refresh: every 6 hours

Upstream:
  Cloudflare DNS-over-TLS
  Google DNS-over-TLS

Protection:
  Per-IP rate limiting
  UFW firewall
  systemd hardening

Security Disclaimer

RAMDNS is designed to reduce attack surface and mitigate common abuse, but no public Internet service can be guaranteed to be completely immune to attacks or DDoS.

For high-volume production deployments, additional network-level DDoS protection, upstream filtering, capacity planning, and monitoring may be required.

License

See the repository license for the current licensing terms.

Status

RAMDNS is intended to operate as a lightweight public DNS resolver with encrypted DNS support and automatic domain blocking.

The project deliberately keeps management functionality outside the DNS data plane.
