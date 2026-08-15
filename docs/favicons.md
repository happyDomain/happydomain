# Favicons

happyDomain shows an icon next to each domain and next to each DNS provider.
Those icons are not shipped with the application: they are the favicon of the
site the domain or the provider belongs to, fetched at runtime and cached.

Where they are fetched from is configurable, because the reasonable answer
depends on the deployment: a public instance wants coverage, a personal one
wants nothing to leave the machine, an air-gapped one wants the feature off.

## The endpoint

```
GET /api/favicon/:domain
```

It is unauthenticated and rate limited per client. It answers with the icon
bytes, served from happyDomain's own origin, with a one-day cache and a
restrictive `Content-Security-Policy`.

The icon is never a redirect. Redirecting the browser to wherever the icon lives
would be cheaper for the server, but it would send the list of domains a user
manages to third parties (the sites themselves, or the icon service), from the
user's own IP address, and it would turn the endpoint into an open redirect
towards whatever URL a remote page declares as its icon. Proxying keeps a single
fetch serving every user of the instance, and keeps the feature working for
browsers that have no route to the outside.

Provider icons are served the same way, by `GET /api/providers/_specs/:psid/icon`.

## Choosing sources

```
-favicon-source direct,duckduckgo
HAPPYDOMAIN_FAVICON_SOURCE=direct,duckduckgo
```

The list is ordered: the next source is tried whenever one fails, and the first
icon obtained is the one served and cached. The option may be repeated or given
as a comma separated list, and accepts `none`.

| Source | What it does | Coverage | Confidentiality |
|---|---|---|---|
| `direct` | Fetches the site's home page, reads its icon links, downloads the best one | Any reachable site, including one nobody has ever crawled | Nothing is told to a third party; each site sees a request from the instance |
| `duckduckgo` | Asks `icons.duckduckgo.com` | Only the sites it crawled; answers 404 for the rest | The service learns which domains this instance looks up |
| `google` | Asks `www.google.com/s2/favicons` | Answers a generic globe rather than an error for unknown domains, so it can never fail: anything listed after it is dead weight | Same as above |
| a URL template | Any other icon service, `{domain}` marking where the host goes | Depends on the service | Depends on the service |
| `none` | Nothing is fetched | No icon at all | Nothing leaves the instance |

The default is `direct,duckduckgo`, the best coverage measured over the
built-in provider list: `direct` alone resolves 47 of the 57 that declare a
website (the rest refuse the page fetch outright), `duckduckgo` alone
resolves 52 but none of the domains it has not crawled, and the two chained
resolve all but two. `direct` is tried first so that a small or private domain
never reaches DuckDuckGo when the site itself would have answered; an instance
that wants nothing told to a third party can set `-favicon-source direct`.

## What `direct` looks for

Only what a browser would put in a tab: `<link rel="icon">`, its `shortcut`
and `alternate` spellings, `apple-touch-icon`, and failing all of those,
`/favicon.ico`.

Deliberately not collected:

- **`og:image` and `twitter:image`**, which generic favicon finders do collect.
  They are share previews, typically a 1200x630 photograph weighing hundreds of
  kilobytes: not an icon, unrecognisable at 16 pixels, and the largest image on
  offer, so any ranking that prefers big images picks it every time.
- **`mask-icon`**, a monochrome silhouette that Safari tints and everything
  else draws as a black blob.

Among the icons a page does declare, the one closest to 64 pixels wins, an SVG
first since it fits every size. If it turns out to be missing, the next ones are
tried, up to four, the last attempt always being `/favicon.ico`.

A self-hosted icon service is written as a template:

```
-favicon-source https://icons.example.org/{domain}.png
```

Only the host name is substituted, never a path or a query from the requested
URL. If the service lives on your own network, its address also has to be
allowed with `-outbound-allowed-target` (see [Outbound
destinations](outbound-targets.md)): the icon fetcher uses the same guard as
every other request happyDomain makes on a user's behalf.

## When no icon is found

The endpoint answers 404, and the interface draws a monogram instead: the first
letter of the name on a coloured tile, the colour derived from the name itself.
It is drawn by the browser, so it costs no request and works on an instance
configured with `none`, or with no route to the outside at all.

The colour comes from the name so that a list of domains without favicons stays
readable: every row looks different, and a given name always gets the same tile.
Nothing is cached and nothing is served in place of a real icon, so a client can
still tell an icon that was found from one that was not.

## What is accepted from a remote host

The bytes come from a host the caller named but are served from happyDomain's
origin, so they are filtered:

- only image media types are served, anything else is refused rather than
  reflected;
- an icon over 256kB is refused rather than truncated;
- an empty body is refused;
- a non-200 answer is a failure, which matters because some icon services
  answer 404 with a placeholder image that a browser would happily display;
- responses are cached, successes for a day (a week for providers), failures
  for fifteen minutes so that a domain without an icon does not turn the
  instance into an amplifier aimed at a third party;
- the cache is bounded in both entries and bytes.
