# Records compliance: goals & scenarios

## Architecture

Frontend-only, registry-based. `$lib/services/compliance.ts` defines the
`ComplianceIssue` shape and exposes `registerValidators(svctype, …)`; each
service module registers its checks and `compliance/registry.ts` is the single
side-effect import point. `EditorCompliance.svelte` is mounted under every
editor and stays hidden when no validator is registered or no issue is
returned. Wiring a new service type should be one import plus one `register`
call.

Validators come in two layers:

- **Sync** runs on every keystroke from the parsed value. It covers syntax, RFC
  field constraints, and zero-network cross-record checks.

- **Async** is debounced, allowed to call the resolver backend for checks that
  need authoritative DNS or HTTPS lookups.


## Scenarios covered per validator

### SPF (`svcs.SPF`)

Sync:
- missing `v=` or wrong version (error)
- multiple `all` directive, multiple `redirect=`, unknown mechanism, lookup-mechanism without value (error)
- `all` not last, `redirect` with `all`, no `all`/`redirect`, `ptr` deprecated, unknown modifier, empty term (warning)
- duplicate mechanism, record > 255 chars (info)

Async (through new `spf-flatten` resolver route):
- recursive lookup budget > 10, include loop (error)
- ≥ 8 lookups, > 2 void lookups, include with no SPF record (warning)
- per-child resolver/timeout error (info)

### DKIM (`svcs.DKIMRecord`)

Sync:
- missing/invalid selector, parse error, wrong `v=`, missing `p=`, invalid base64, weak RSA key (< 1024) (error)
- revoked key (`p=` empty), short RSA key (< 2048), deprecated hash `sha1`, unknown key type/hash/`t=` flag (warning)
- testing mode `t=y`, unknown service type, deprecated `g=` (info)

### DMARC (`svcs.DMARC`)

Sync:
- wrong owner name, parse error, missing/invalid `v=`/`p=`/`sp=`, invalid `adkim`/`aspf`/`pct`/`ri`, invalid URI scheme, malformed `mailto:`, no alignment source while enforcing (error)
- invalid `fo`, unknown `rf`, empty URI, no alignment source (`p=none`), `adkim=s` without DKIM in zone (warning)
- `p=none`, `pct<100`, external reporting destination detected (info)

Async (through `dmarc-report-auth`):
- external reporting authorization missing/external domain has no DMARC (error)
- resolver error during the lookup (warning)

### MTA-STS (`svcs.MTA_STS`)

Sync:
- wrong owner name, parse error, missing/invalid `v=`/`id=` (error)

Async (through `mta-sts-policy`):
- policy DNS/TLS/not-found/too-large, invalid version/mode, missing `mx:` while filtering, missing/invalid `max_age`, zone MX not covered (in `enforce`) (error)
- HTTP redirect or non-200, fetch error, `mode=none`, `max_age` < 1 day, zone has no apex MX, zone MX not covered (in `testing`) (warning)
- `mode=testing`, policy `mx:` pattern unused by zone (info)

### TLS-RPT (`svcs.TLS_RPT`)

Sync:
- wrong owner name, parse error, missing/invalid `v=`, missing `rua=`, invalid `rua` scheme, malformed `mailto:` (error)
- empty `rua` entry (warning)

### MX (`svcs.MXs`)

Sync:
- null MX mixed with other records, invalid hostname, invalid preference, in-zone target is a CNAME (error)
- null MX with non-zero preference, duplicate target, in-zone target has no A/AAAA (warning)

### BIMI (`svcs.BIMI`)

Sync:
- wrong owner name, missing/invalid selector, invalid `v=`, missing `l=` (outside declination), `l=` or `a=` not HTTPS (error)
- DMARC policy is `none` everywhere, no DMARC in zone, `l=` not `.svg`, `e=` not HTTPS (warning)
- declination detected, missing VMC `a=`, `a=` not `.pem` (info)
