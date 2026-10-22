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
certificate on a hostname that exists only for this purpose. HappyDomain ships
an **MTA-STS (hosted policy)** service that publishes both halves. It emits the
TXT record *and* a `mta-sts.<domain>` CNAME toward the HappyDomain instance,
which serves the policy file itself.

The plain **MTA-STS** service remains available for domains that serve the
policy file from their own web server: it only manages the TXT record.

## Configuration

HappyDomain needs to know the public FQDN where it serves hosted content: it is
the target of the `mta-sts.` CNAME. This is the `--service-hosting-host` setting
described in the [email auto-configuration](email-autoconfig.md) page, which
shares it. The deprecated `--mail-autoconfig-host` is an alias of it, so it
applies to MTA-STS too.

That hostname must be reachable over HTTPS and able to get a valid certificate
for `mta-sts.<user-domain>`.

## Endpoints exposed by HappyDomain

Both are public, rate-limited and read-only, under the same conditions as the
[email auto-configuration](email-autoconfig.md#endpoints-exposed-by-happydomain)
endpoints, whose limit they share.

| Method | Path                       | Purpose                                       |
| ------ | -------------------------- | --------------------------------------------- |
| GET    | `/.well-known/mta-sts.txt` | RFC 8461 policy file for the requested domain |
| GET    | `/api/caddy/ask`           | Caddy on-demand TLS validation hook           |

The policy endpoint derives the domain from the `Host` header: RFC 8461 pins
the URL down exactly, leaving no room for a query parameter.

The Caddy hook is shared by every hosted service, [BIMI](bimi.md) included.
For this one, it authorises certificates for `mta-sts.<X>` where `X` is a
domain registered in HappyDomain *and* has the MTA-STS (hosted policy) service
with the policy hosting enabled.
Nothing else gets a certificate.

## End-user flow

1. User adds their domain to HappyDomain (existing flow).
2. From the service catalogue (Email category), the user picks
   "MTA-STS (hosted policy)".
3. The dedicated form asks for:
   - Policy mode: `testing` (report only), `enforce`, or `none`.
   - Max age: how long senders cache the policy. One week minimum recommended.
   - Authorized MX hosts, **pre-filled from the zone's own MX records** and
     editable, with RFC 8461 `*.` wildcards allowed.
   - A toggle to have HappyDomain serve the policy file.
4. The form shows the exact policy file that will be served, and the editor
   bumps the TXT record's `id` whenever the policy changes, which is what
   RFC 8461 requires for senders to notice the update.
5. Saving generates the TXT record and the CNAME. The user applies the diff to
   the zone as usual.
6. Sending mail servers now discover the policy and enforce TLS.

## Deploying with Caddy (recommended)

The Caddy setup is the one of the
[email auto-configuration](email-autoconfig.md#deploying-with-caddy-recommended),
whose catch-all already covers the `mta-sts.` host names: a single Caddy
instance fronts HappyDomain and handles TLS for the main UI and every hosted
service.

When a user (say `example.com`) configures the service, their DNS will hold:

```
_mta-sts.example.com.  3600 IN TXT   "v=STSv1; id=20260814T101500Z"
mta-sts.example.com.   3600 IN CNAME happydomain.example.com.
```

The first time a sending mail server hits
`https://mta-sts.example.com/.well-known/mta-sts.txt`, Caddy:

1. Receives the request for an unknown hostname.
2. Calls `https://happydomain.example.com/api/caddy/ask?domain=mta-sts.example.com`.
3. HappyDomain checks: parent `example.com` is registered and has the MTA-STS
   hosted-policy service → returns 200.
4. Caddy obtains a Let's Encrypt certificate for `mta-sts.example.com` and
   reverse-proxies the request to HappyDomain.
5. HappyDomain renders the policy from the user's stored service config:

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
  the Caddyfile of the email auto-configuration page satisfies them.
- HappyDomain answers with `Cache-Control: public, max-age=3600`, deliberately
  shorter than the policy's own `max_age`: a long HTTP cache would keep a stale
  policy in front of senders that re-fetched after seeing a new `id` in DNS.
- The service's compliance checks fetch the live policy and compare it with
  what is configured, which surfaces a CNAME that has not propagated, a missing
  certificate, or an edit that was never applied to the zone.
