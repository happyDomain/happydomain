# MTA-STS

[RFC 8461](https://www.rfc-editor.org/rfc/rfc8461) (SMTP MTA Strict Transport
Security) lets a domain tell other mail servers to only deliver to it over
authenticated TLS. It has two halves:

1. A **DNS record**, `_mta-sts.<domain>` TXT, announcing that a policy exists
   and carrying an `id` that changes whenever the policy does.
2. A **policy file**, served over HTTPS at
   `https://mta-sts.<domain>/.well-known/mta-sts.txt`, listing the authorized
   MX hosts and the enforcement mode.

The second half is what stops most people: it needs a web server and a valid
certificate on a hostname that exists only for this purpose. happyDomain ships
an **MTA-STS (hosted policy)** service that publishes both halves — it emits the
TXT record *and* a `mta-sts.<domain>` CNAME toward the happyDomain instance,
which serves the policy file itself.

The plain **MTA-STS** service remains available for domains that serve the
policy file from their own web server: it only manages the TXT record.

Like the [email auto-configuration](email-autoconfig.md), the HTTPS side is
delegated to a reverse proxy (Caddy is the recommended choice) using **on-demand
TLS**. happyDomain exposes a validation endpoint Caddy queries before issuing
each certificate, so certificates are only obtained for domains that actually
opted into the service.

## Configuration

happyDomain needs to know the public FQDN where it serves hosted content —
that's the target of the `mta-sts.` CNAME the service emits.

| Setting                     | CLI flag / env                                                  | Default                      |
| --------------------------- | --------------------------------------------------------------- | ---------------------------- |
| Public happyDomain URL      | `--externalurl` / `HAPPYDOMAIN_EXTERNAL_URL`                    | `http://localhost:8081`      |
| Public host for hosted services | `--service-hosting-host` / `HAPPYDOMAIN_SERVICE_HOSTING_HOST` | derived from `--externalurl` |

`--service-hosting-host` is shared with the email auto-configuration endpoints.
The former `--mail-autoconfig-host` is still honoured as a deprecated alias, but
it now applies to MTA-STS too, so prefer the new name.

If neither is set, happyDomain uses the host part of `--externalurl`. That
hostname must be reachable over HTTPS and able to get a valid certificate for
`mta-sts.<user-domain>` (see the Caddy section below).

## Endpoints exposed by happyDomain

Both are public, rate-limited (30 req/min per client IP, shared with the email
auto-configuration endpoints), and read-only. Neither requires authentication.
Since happyDomain is meant to be fronted by Caddy here, declare it with
`-trusted-proxy` so the limiter sees the real client and not the proxy (see
[reverse-proxy.md](reverse-proxy.md)).

| Method | Path                         | Purpose                                     |
| ------ | ---------------------------- | ------------------------------------------- |
| GET    | `/.well-known/mta-sts.txt`   | RFC 8461 policy file for the requested domain |
| GET    | `/api/caddy/ask`             | Caddy on-demand TLS validation hook          |

The policy endpoint derives the domain from the `Host` header: RFC 8461 pins
the URL down exactly, leaving no room for a query parameter.

The Caddy hook is shared by every hosted service. It authorises certificates for
`mta-sts.<X>`, `autoconfig.<X>` and `autodiscover.<X>` where `X` is a domain
registered in happyDomain *and* has the matching service configured. Nothing
else gets a certificate.

## End-user flow

1. User adds their domain to happyDomain (existing flow).
2. From the service catalogue (Email category), the user picks
   "MTA-STS (hosted policy)".
3. The dedicated form asks for:
   - Policy mode: `testing` (report only), `enforce`, or `none`.
   - Max age: how long senders cache the policy. One week minimum recommended.
   - Authorized MX hosts, **pre-filled from the zone's own MX records** and
     editable, with RFC 8461 `*.` wildcards allowed.
   - A toggle to have happyDomain serve the policy file.
4. The form shows the exact policy file that will be served, and the editor
   bumps the TXT record's `id` whenever the policy changes — which is what
   RFC 8461 requires for senders to notice the update.
5. Saving generates the TXT record and the CNAME. The user applies the diff to
   the zone as usual.
6. Sending mail servers now discover the policy and enforce TLS.

## Deploying with Caddy (recommended)

A single Caddy instance can front happyDomain and handle TLS for the main UI,
the MTA-STS policy host, and the auto-configuration endpoints.

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

# Catch-all for the hosted services: mta-sts.<X>, autoconfig.<X>,
# autodiscover.<X>. Caddy obtains a certificate on-demand for each new <X>
# only when the /api/caddy/ask endpoint authorises it.
https:// {
    @hosted header_regexp Host ^(?:mta-sts|autoconfig|autodiscover)\.
    handle @hosted {
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
      HAPPYDOMAIN_SERVICE_HOSTING_HOST: happydomain.example.com
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
_mta-sts.example.com.  3600 IN TXT   "v=STSv1; id=20260814T101500Z"
mta-sts.example.com.   3600 IN CNAME happydomain.example.com.
```

The first time a sending mail server hits
`https://mta-sts.example.com/.well-known/mta-sts.txt`, Caddy:

1. Receives the request for an unknown hostname.
2. Calls `https://happydomain.example.com/api/caddy/ask?domain=mta-sts.example.com`.
3. happyDomain checks: parent `example.com` is registered and has the MTA-STS
   hosted-policy service → returns 200.
4. Caddy obtains a Let's Encrypt certificate for `mta-sts.example.com` and
   reverse-proxies the request to happyDomain.
5. happyDomain renders the policy from the user's stored service config:

```
version: STSv1
mode: enforce
mx: mail.example.com
mx: *.example.net
max_age: 604800
```

Subsequent requests reuse the cached certificate.

## Notes

- RFC 8461 forbids redirects when fetching the policy, and requires a
  certificate valid for `mta-sts.<domain>`. Both are the reverse proxy's job;
  the Caddyfile above satisfies them.
- happyDomain answers with `Cache-Control: max-age=3600`, deliberately shorter
  than the policy's own `max_age`: a long HTTP cache would keep a stale policy
  in front of senders that re-fetched after seeing a new `id` in DNS.
- The service's compliance checks fetch the live policy and compare it with
  what is configured, which surfaces a CNAME that has not propagated, a missing
  certificate, or an edit that was never applied to the zone.
