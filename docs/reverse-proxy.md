# Running happyDomain behind a reverse proxy

happyDomain identifies callers by their IP address to enforce the per-source
rate limits (`/api/domaininfo/:domain`, the mail auto-configuration endpoints,
notification channel tests) and the login lockout that requires a captcha after
repeated failures.

When happyDomain is fronted by a reverse proxy, every request reaches it from
the proxy address, and the real caller is only known through the
`X-Forwarded-For` (or `X-Real-IP`) header the proxy adds. That header is trivial
to forge, so happyDomain only honours it when the request comes from a peer the
operator declared as a proxy.

## `-trusted-proxy`

```
-trusted-proxy 10.0.0.0/8
```

The option takes an IP address or a CIDR block, and can be repeated:

```
happyDomain -trusted-proxy 192.0.2.10 -trusted-proxy 2001:db8::10
```

It also accepts a comma separated list, which is the only usable form for the
environment variable and the config file, as those can only be given once:

```
HAPPYDOMAIN_TRUSTED_PROXY=10.0.0.0/8,2001:db8::/32
```

```
# happydomain.conf
trusted-proxy=10.0.0.0/8,2001:db8::/32
```

Give the exact address of your proxy. A range is only for the case where the
proxy address genuinely is not stable, and every address in it is allowed to
claim any client IP, so keep it as small as you can.

The keyword `none` empties the list. The config file, the environment and the
command line all feed the same accumulating option, so without it a value
inherited from a lower-precedence source could only ever be widened:

```
# trust only this proxy, whatever the compose file or happydomain.conf set
happyDomain -trusted-proxy none,192.0.2.10
```

An invalid value stops happyDomain at startup rather than silently disabling
the check. Two forms are refused rather than reinterpreted, because both would
trust something other than what they look like:

- a block with host bits set (`192.0.2.5/24`), which reads like one host but
  covers the whole subnet: write `192.0.2.0/24` or `192.0.2.5`;
- an IPv4-mapped IPv6 literal (`::ffff:192.0.2.1`), which would leave the
  intended proxy untrusted and trust `::1` instead: write `192.0.2.1`.

## Choosing a value

- **No proxy** (happyDomain exposed directly): leave the option unset. This is
  the default: forwarded headers are ignored and callers are identified by their
  socket address.
- **Proxy on the same host**: the address it connects from, usually
  `127.0.0.1`.
- **Proxy in the same container network**: the address of the proxy container,
  not the network it sits on. Pin it (a static address on the compose network,
  or read it back with `docker inspect`), because trusting the whole bridge
  means any sibling container that reaches the port directly, bypassing the
  proxy, can forge `X-Forwarded-For`.
- **Proxy on another machine, or a CDN**: list its addresses or ranges. Only
  list hops you control: every address in the list is allowed to claim any
  client IP.

Private ranges are not special: `10.0.0.0/8` or `192.168.0.0/16` trust every
host on the network the proxy happens to sit on, which on a LAN or a container
bridge is a much larger set than the one machine you meant.

Two failure modes to keep in mind:

- Leaving the option unset while running behind a proxy makes every client share
  a single bucket (the proxy address), so one busy user can rate limit everyone
  and per-source lockouts become effectively global.
- Trusting too much (for instance `0.0.0.0/0`) reinstates the bypass: any caller
  can then pick the address it is throttled under.

## Making sure the proxy sets the header

happyDomain reads `X-Forwarded-For`, then `X-Real-IP`. Your proxy must set them
from the connection it received, and must not pass through a client supplied
value.

Caddy sets `X-Forwarded-For` on its own:

```caddyfile
happydomain.example.com {
    reverse_proxy happydomain:8081
}
```

nginx:

```nginx
location / {
    proxy_pass http://127.0.0.1:8081;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Real-IP $remote_addr;
}
```

Traefik adds `X-Forwarded-For` by default; make sure the entrypoint declares
`forwardedHeaders.trustedIPs` for its own upstreams if there is one.

When a request carrying a forwarded header arrives from a peer that is not in
the trusted list, happyDomain logs a warning (at most once every 15 minutes)
with the peer address. Seeing your proxy address there means `-trusted-proxy`
is missing; seeing a client address there means someone is trying to spoof its
identity. A declared proxy never triggers the warning, including when it
forwards a header happyDomain cannot parse.

Because a single warning covers each 15 minute window, a noisy spoofing source
can delay the report of a second one: check the whole window before concluding
your proxy is correctly declared.
