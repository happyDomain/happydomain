# Domain watchlist

The domain watchlist lets users follow domain names they do not manage, often
names they would like to own, and be told when such a domain is about to be
released and when they can register it. A watched domain needs no provider and
no zone: it may belong to anyone, or to no one yet.

The user documentation is on the help center. This document is for
administrators, who need to know what the watchlist queries and how often, and
for contributors.

## Goals

- **Follow the life of a domain, not a single date.** An expired domain is not
  released at once: for gTLDs it goes through a grace period, a redemption
  period and a pending deletion period first. Each step is told to the user,
  with an estimate of the release date once it can be made.
- **Check as often as the domain deserves, no more.** An expiration date does
  not come closer by surprise: a registered domain is looked up once a week.
  Past its expiration date, every step lasts days, so twice a day is enough not
  to miss one.
- **Never lose what is known.** A failed lookup keeps what is known of the
  domain.
- **Stay small.** The watchlist is a standalone entity with its own background
  worker. It does not go through the checker engine: a watched domain is not a
  happyDomain domain, and has no checker plan, quota, nor execution history.

## Life cycle

`domaininfo.PhaseOf` (`pkg/domaininfo/lifecycle.go`) tells the phase of a domain
from its RDAP or WHOIS lookup. The `domain_expiry` checker uses the same
function for managed domains (see [Domain checkers](checker-domains.md)). The
registry statuses are compared however they are spelled, whether RDAP's
`pending delete` or EPP's `pendingDelete`.

| Phase | Detected by | Next check |
|---|---|---|
| `registered` | Registered, expiring in more than 30 days, or no expiration date published | 7 days, but never later than 30 days before the expiration date, nor than 6 hours after it; 12 hours while the `autoRenewPeriod` status is present |
| `expiring` | Registered, expiring within 30 days | Same as `registered` |
| `grace` | Expiration date passed | 12 hours |
| `redemption` | `redemptionPeriod` or `pendingRestore` status | 12 hours |
| `pending_delete` | `pendingDelete` status | 12 hours |
| `available` | The registry does not know the domain (RDAP 404, WHOIS "no match") | 12 hours, to see it registered again |
| `unknown` | Never looked up successfully yet | 1 hour after a failed lookup |

Every next check is delayed by up to 30 minutes at random, so that the lookups
spread over time. A failed lookup keeps the phase and the registration facts as
they were, records the error, and tries again 12 hours later.

The estimated release date is the latest date the registry should release the
domain at: 5 days after it was first seen pending deletion, 35 days after it
was first seen in redemption (the ICANN lengths for gTLDs). A domain already in
its phase when first seen is released earlier.

## For administrators

### Load on the registries

Lookups go through the same RDAP then WHOIS chain as the registration data
page, each bounded to 20 seconds. Adding a domain and "check now" look it up
while the user waits; "check now" returns the last result when the domain was
checked less than a minute ago.

The worker (`internal/usecase/domainwatch/worker.go`) wakes up every 15 minutes
and looks up the watches that are due, 4 at a time. A domain watched by several
users is looked up once. A lookup cut short by the shutdown is not recorded:
the watch stays due. With the cadence above, a watched domain costs about one
lookup a week while registered, and two a day once expired or available.

The watches are not paused for inactive users, unlike checks: the point of a
watch is to wait, sometimes for months, without logging in.

### Options

| Option | Default | Description |
|---|---|---|
| `-domain-watch-max-per-user` | `100` | Maximum number of domains a user can watch; `0` for unlimited. |

## Known limits

- **ccTLDs follow their own rules.** Many have no grace or redemption period,
  or call them otherwise; some publish no expiration date (`.de`). The phases
  then come down to `registered` and `available`, which still tells the user
  when the domain is released.
- **Not every unknown domain can be registered.** A reserved, blocked or
  premium name is reported `available` as soon as the registry does not know
  it.
- **Drop-catching services are faster.** A released domain may be registered
  within seconds by such services. The watchlist tells the user when to be
  ready, it does not register the domain for them.
- **`autoRenewPeriod` is a hint.** Some registries auto-renew every expired
  domain while waiting for the registrar: the domain shows as `registered`,
  with its new expiration date, until the registrar confirms the renewal or
  deletes the domain. The watchlist cannot tell which will happen, so it looks
  the domain up twice a day meanwhile, to report a deletion within hours.
