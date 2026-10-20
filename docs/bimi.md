# BIMI

[BIMI](https://datatracker.ietf.org/doc/draft-brand-indicators-for-message-identification/)
(Brand Indicators for Message Identification) lets a domain have its brand logo
displayed next to its emails, in the mail clients that support it. It has
three parts:

1. A **DNS record**, `<selector>._bimi.<domain>` TXT, pointing to the files
   below with its `l=` and `a=` tags. The selector is `default` unless the
   domain publishes several logos.
2. A **logo**, an SVG file using the SVG Tiny Portable/Secure profile, served
   over HTTPS.
3. Optionally, a **mark certificate**, a Verified Mark Certificate (VMC) or a
   Common Mark Certificate (CMC) issued for that logo, served over HTTPS in PEM
   format. Most mail providers (Gmail, Yahoo, Apple Mail) only display the logo
   when there is one.

Hosting the two files is what stops most people. happyDomain ships a
**BIMI (hosted logo)** service that takes care of all three parts. It stores
the logo and the certificate, serves them at `https://bimi.<domain>/bimi/`
behind a `bimi.<domain>` CNAME toward the happyDomain instance, and generates
the TXT record pointing to them.

The plain **BIMI** service remains available for domains that host the files
themselves: it only manages the TXT record.

## Configuration

happyDomain needs to know the public FQDN where it serves hosted content: it is
the target of the `bimi.` CNAME. This is the `--service-hosting-host` setting
described in the [email auto-configuration](email-autoconfig.md) page, which
shares it. The deprecated `--mail-autoconfig-host` is an alias of it, so it
applies to BIMI too.

That hostname must be reachable over HTTPS and able to get a valid certificate
for `bimi.<user-domain>`.

## Endpoints exposed by happyDomain

All three are public, rate-limited and read-only, under the same conditions as the
[email auto-configuration](email-autoconfig.md#endpoints-exposed-by-happydomain)
endpoints, whose limit they share.

| Method | Path                | Purpose                                 |
| ------ | ------------------- | --------------------------------------- |
| GET    | `/bimi/<name>.svg`  | The logo of the requested domain        |
| GET    | `/bimi/<name>.pem`  | The mark certificate of the domain      |
| GET    | `/api/caddy/ask`    | Caddy on-demand TLS validation hook     |

The file endpoints derive the domain from the `Host` header, and only answer on
`bimi.<domain>` host names: the files are user content, so happyDomain never
serves them from its own host name. `<name>` is derived from the content of the
file (the first 16 hex digits of its SHA-256), so only the current files are
served: a former name gets a 404.

The Caddy hook is shared by every hosted service. For this one, it authorises
certificates for `bimi.<X>` where `X` is a domain registered in happyDomain
*and* has the BIMI (hosted logo) service with its `bimi.` CNAME. Nothing else
gets a certificate.

## End-user flow

1. User adds their domain to happyDomain (existing flow).
2. From the service catalogue (Email category), the user picks
   "BIMI (hosted logo)".
3. The dedicated form asks for:
   - The selector, `default` unless several logos are published.
   - The logo, an SVG Tiny PS file of at most 32 KiB.
   - Optionally, the mark certificate with its chain, in PEM format.
4. The form shows the TXT record that will be published. happyDomain
   generates it from the files, so it always points at what is served.
5. Saving generates the TXT record and the `bimi.` CNAME. The user applies the
   diff to the zone as usual.
6. Receiving mail servers now fetch the logo, and display it for mail that
   passes DMARC.

## Deploying with Caddy (recommended)

The Caddy setup is the one of the
[email auto-configuration](email-autoconfig.md#deploying-with-caddy-recommended),
whose catch-all covers the `bimi.` host names: a single Caddy instance fronts
happyDomain and handles TLS for the main UI and every hosted service. If you
deployed it before BIMI support, add `bimi` to the `@hosted` host regexp
(`^(?:autoconfig|autodiscover|mta-sts|bimi)\.`), otherwise Caddy ignores the
`bimi.` host names.

When a user (say `example.com`) configures the service, their DNS will hold:

```
default._bimi.example.com. 3600 IN TXT   "v=BIMI1;l=https://bimi.example.com/bimi/3f1c0e6a2b9d4c71.svg;a=https://bimi.example.com/bimi/8a0b5e2f9c3d1e47.pem"
bimi.example.com.          3600 IN CNAME happydomain.example.com.
```

The first time a receiving mail server hits
`https://bimi.example.com/bimi/3f1c0e6a2b9d4c71.svg`, Caddy:

1. Receives the request for an unknown hostname.
2. Calls `https://happydomain.example.com/api/caddy/ask?domain=bimi.example.com`.
3. happyDomain checks: parent `example.com` is registered and has the BIMI
   hosted-logo service with its `bimi.` CNAME → returns 200.
4. Caddy obtains a Let's Encrypt certificate for `bimi.example.com` and
   reverse-proxies the request to happyDomain.
5. happyDomain serves the logo from the user's stored service config.

Subsequent requests reuse the cached certificate.

## Notes

- Receivers only display the logo for mail passing a DMARC policy at
  `quarantine` or `reject`. The service's compliance checks point out a zone
  without such a policy.
- The files are checked when the zone is saved, and again before being
  served. The logo must declare `baseProfile="tiny-ps"` and carry nothing
  active: no script, no `foreignObject`, no event handler, no reference outside
  the document, no entity declaration. The certificate file must only contain
  certificates, so that a private key pasted by mistake is never published.
  Whether the certificate is a valid mark certificate for the domain is left to
  the receivers.
- Every file is served with `X-Content-Type-Options: nosniff` and a sandboxing
  `Content-Security-Policy`, so that a browser opening the logo directly never
  runs anything from it.
- As file names change with their content, a new logo gets a new URL that
  receivers have not cached yet. For the same reason, happyDomain lets
  intermediaries cache the files for a day (`Cache-Control: public,
  max-age=86400`).
- The files cannot be recovered from DNS: after importing a zone, the record
  and the CNAME are recognised as this service, but the logo and the
  certificate must be uploaded again. Until then the record is published as
  is, and the compliance checks point out the missing logo.
- The service publishes a single selector per domain, as the `bimi.` CNAME can
  only back one of them. Other selectors can still use the plain BIMI service.
