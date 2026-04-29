# Email auto-configuration

happyDomain ships an integrated **Email Auto-configuration** service that lets
users publish IMAP/POP3/SMTP settings for their mail clients via the three
de-facto standards mail clients try in order:

1. **RFC 6186** — DNS SRV records (`_imap._tcp`, `_imaps._tcp`,
   `_submission._tcp`, …).
2. **Mozilla Autoconfig** — `https://autoconfig.<domain>/mail/config-v1.1.xml`
   (Thunderbird).
3. **Microsoft Autodiscover** — `https://autodiscover.<domain>/Autodiscover/Autodiscover.xml`
   (Outlook).

A single happyDomain service emits the SRV records *and* the CNAMEs for the
two HTTP-based standards. happyDomain itself serves the XML responses for
those CNAMEs, so users get a fully working mail-client auto-configuration
out of the box, without operating an extra Web server.

The HTTP-based standards require valid HTTPS certificates on
`autoconfig.<domain>` and `autodiscover.<domain>`. happyDomain delegates that
to a reverse proxy (Caddy is the recommended choice) that uses **on-demand
TLS** to obtain certificates automatically. happyDomain exposes a small
validation endpoint that Caddy queries before issuing each certificate, so
certificates are only obtained for domains that actually opted into the
service.

## Configuration

happyDomain needs to know the public FQDN where it serves the
auto-configuration XML — that's the target of the `autoconfig.` and
`autodiscover.` CNAMEs the service emits.

| Setting                              | CLI flag / env                                | Default               |
| ------------------------------------ | --------------------------------------------- | --------------------- |
| Public happyDomain URL               | `--externalurl` / `HAPPYDOMAIN_EXTERNAL_URL`  | `http://localhost:8081` |
| Public host for autoconfig endpoints | `--mail-autoconfig-host` / `HAPPYDOMAIN_MAIL_AUTOCONFIG_HOST` | derived from `--externalurl` |

If `--mail-autoconfig-host` is left unset, happyDomain uses the host part of
`--externalurl`. The same hostname must be reachable over HTTPS and able to
get a valid certificate for `autoconfig.<user-domain>` and
`autodiscover.<user-domain>` (see the Caddy section below).

## Endpoints exposed by happyDomain

All three are public, rate-limited (30 req/min per client IP), and read-only.
None require authentication.

| Method  | Path                                | Purpose                                       |
| ------- | ----------------------------------- | --------------------------------------------- |
| GET     | `/mail/config-v1.1.xml`             | Mozilla Autoconfig XML for Thunderbird        |
| GET/POST| `/Autodiscover/Autodiscover.xml`    | Microsoft Autodiscover XML for Outlook        |
| GET/POST| `/autodiscover/autodiscover.xml`    | Same, lowercase variant                       |
| GET     | `/api/caddy/ask`                    | Caddy on-demand TLS validation hook           |

The Caddy hook only authorises certificates for `autoconfig.<X>` /
`autodiscover.<X>` where `X` is a domain registered in happyDomain *and* has
the Email Auto-configuration service configured.

## End-user flow

1. User adds their domain to happyDomain (existing flow).
2. From the service catalogue (Email category), the user picks
   "Email Auto-configuration".
3. The dedicated form asks for:
   - Incoming server: protocol (IMAP/IMAPS/POP3/POP3S), hostname, port,
     authentication method.
   - Outgoing server: protocol (submission/submissions), hostname, port,
     authentication method.
   - Discovery toggle (publishes the autoconfig./autodiscover. CNAMEs).
   - Optional Microsoft Exchange server.
   - Optional branding (display name, username format).
4. Saving the service generates the SRV records, the CNAMEs, and an
   `_autodiscover._tcp` SRV. The user applies the diff to the zone as usual.
5. Mail clients now self-configure when fed `user@<domain>`.

## Deploying with Caddy (recommended)

A single Caddy instance can front happyDomain and handle TLS for both the
main UI and the auto-configuration endpoints.

### Caddyfile

```caddyfile
{
    # Tell Caddy to ask happyDomain before issuing certificates for
    # arbitrary subdomains.
    on_demand_tls {
        ask https://happydomain.example.com/api/caddy/ask
    }
}

# Main happyDomain UI on its own (regular) hostname.
happydomain.example.com {
    reverse_proxy happydomain:8081
}

# Catch-all for autoconfig.<X> and autodiscover.<X>.
# Caddy obtains a certificate on-demand for each new <X> only when the
# /api/caddy/ask endpoint authorises it.
https:// {
    @autoconfig header_regexp Host ^(?:autoconfig|autodiscover)\.
    handle @autoconfig {
        reverse_proxy happydomain:8081
    }

    handle {
        respond 404
    }

    tls {
        on_demand
    }
}
```

### docker-compose example

```yaml
services:
  happydomain:
    image: happydomain/happydomain:latest
    environment:
      HAPPYDOMAIN_EXTERNAL_URL: https://happydomain.example.com
      HAPPYDOMAIN_MAIL_AUTOCONFIG_HOST: happydomain.example.com
    expose:
      - 8081
    volumes:
      - happydomain-data:/data

  caddy:
    image: caddy:2
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy-data:/data
      - caddy-config:/config

volumes:
  happydomain-data:
  caddy-data:
  caddy-config:
```

When a user (say `example.com`) configures the service, their DNS will hold:

```
_imaps._tcp.example.com.       3600 IN SRV 0 1 993 imap.example.com.
_submission._tcp.example.com.  3600 IN SRV 0 1 587 smtp.example.com.
autoconfig.example.com.        3600 IN CNAME happydomain.example.com.
autodiscover.example.com.      3600 IN CNAME happydomain.example.com.
_autodiscover._tcp.example.com. 3600 IN SRV 0 0 443 happydomain.example.com.
```

The first time Thunderbird/Outlook hits
`https://autoconfig.example.com/mail/config-v1.1.xml`, Caddy:

1. Receives the request for an unknown hostname.
2. Calls `https://happydomain.example.com/api/caddy/ask?domain=autoconfig.example.com`.
3. happyDomain checks: parent `example.com` is registered, has the Email
   Auto-configuration service → returns 200.
4. Caddy obtains a Let's Encrypt certificate for `autoconfig.example.com` and
   reverse-proxies the request to happyDomain.
5. happyDomain renders the Mozilla XML from the user's stored service
   config and returns it.

Subsequent requests reuse the cached certificate.
