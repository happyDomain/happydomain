# The BIND provider

The BIND provider edits zone files on the local file system of the server
running happyDomain. Every other provider talks to a remote API; this one reads
and writes files, with the rights of the happyDomain process.

The directory is picked by the user, in the provider form. Left unchecked, any
account on the instance could therefore name any directory the process can write
to, and the file name is derived from the zone name and the file name format,
which the same form supplies.

happyDomain therefore keeps the provider disabled unless the administrator names
the directories it may use, and refuses anything outside them.

> **Upgrading?** `-with-bind-provider` is no longer a boolean. An instance that
> enabled it with `true`, `1` or an empty value stops at startup with a message
> pointing here. See [From the boolean form](#from-the-boolean-form).

## Enabling it

```
happyDomain -with-bind-provider /etc/named/zones
```

The option both enables the provider and confines it: there is no way to enable
it without declaring at least one directory.

Several directories can be given by repeating the option:

```
happyDomain -with-bind-provider /etc/named/zones -with-bind-provider /srv/zones
```

or, in one value, separated by `:` (`;` on Windows), which is the only usable
form for the environment variable and the config file, as those can only be
given once:

```
HAPPYDOMAIN_WITH_BIND_PROVIDER=/etc/named/zones:/srv/zones
```

```
# happydomain.conf
with-bind-provider=/etc/named/zones
```

Paths must be absolute. A relative one is refused at startup, because it would
be resolved against a working directory that is not the operator's.

## What users can then do

A user configuring the provider gives a **Directory**, which must be one of the
allowed directories or a subdirectory of one, and optionally a **File format**.

Both are checked every time the provider is used, not only when the form is
submitted, so a provider stored before this confinement existed, or stored with
a directory later removed from the list, stops working rather than keeping its
old reach.

Refused, whatever the form says:

- a directory outside the allowed ones, including one reached through `..` or
  through a symlink planted inside an allowed directory: the path is
  canonicalized before the comparison;
- a file name format containing a path separator or a `..`;
- a zone name that is not a plain file name. This rules out RFC 2317 classless
  reverse delegations such as `0/25.2.0.192.in-addr.arpa`: they are valid domain
  names, but cannot be stored in a file named after them.

Failures do not say whether a path exists, so the form cannot be used to probe
the file system.

## Directories mounted later

The allowed paths are resolved when a zone is read or written, not when the
option is parsed, so a zone directory that is a volume mounted after the process
starts is fine: happyDomain logs a warning at startup that it cannot see the
directory yet, and the provider starts working as soon as it appears, with no
restart.

A path that never appears simply allows nothing.

## Choosing the directories

Name the directory holding the zone files and nothing above it. An allowed
directory is allowed with its whole subtree, so `/etc` or `/` hands the process's
write rights to any account on the instance.

The provider is not suitable for a shared or multi-tenant instance at all: users
who can write zone files on the host share one file system, and one user's
directory choice is not isolated from another's.

## From the boolean form

Before this change, `-with-bind-provider` was a flag with no value, and the
provider it enabled could write anywhere the process could. The value is now
mandatory:

| Before | After |
| --- | --- |
| `-with-bind-provider` | `-with-bind-provider /etc/named/zones` |
| `HAPPYDOMAIN_WITH_BIND_PROVIDER=1` | `HAPPYDOMAIN_WITH_BIND_PROVIDER=/etc/named/zones` |
| `with-bind-provider=true` | `with-bind-provider=/etc/named/zones` |

The boolean values are refused rather than treated as "enabled", since there is
no directory to confine the provider to, and refused rather than treated as
"disabled", so that the change surfaces at startup instead of as providers that
mysteriously stopped working.

Use the directory your users already configured on their BIND providers: it is
the `directory` field of the provider form, and any subdirectory of what you
list keeps working.
