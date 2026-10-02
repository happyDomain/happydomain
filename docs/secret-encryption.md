# Encrypting stored credentials

happyDomain stores the credentials of the DNS providers of its users: their
API keys, passwords, TSIG keys. By default they are stored in clear in the
database, as they always were.

The `instance` policy encrypts them under a keyset held by the server, outside
the database. A leaked database dump or backup, or a database hosted apart from
happyDomain, no longer gives the credentials away. It does not protect against
a compromised happyDomain server, which holds the keyset.

## Limits of this version

This first version covers the common path. Before enabling it, know that:

- only the credentials of providers are encrypted. The secrets of
  notification channels and the checker options stay in clear;
- the keyset cannot be rotated yet. Do not add or remove keys;
- the migrations below (`reseal`, `drop-safes`) expect a single happyDomain
  process on the database. With several, a reseal may overwrite an update made
  at the same time by another process;
- when a credential no longer decrypts, users get a generic error.

## Generating the keyset

```
happydomain secret-keyset generate /etc/happydomain/secret-keyset.json
```

The keyset is a cleartext [Tink](https://developers.google.com/tink) JSON
keyset, which `tinkey` can also read. The file is created readable by its owner
only; make sure that owner is the user happyDomain runs as. happyDomain warns
at startup when the file can be read by other users. The command never
overwrites an existing file.

`happydomain secret-keyset info <file>` lists the keys of a keyset, never their
material.

With Docker, mount the file as a secret rather than baking it into an image,
readable by the `happydomain` user of the image (uid 15353) only. A Swarm
secret is mounted with mode `0444` unless told otherwise: set `uid: "15353"`
and `mode: 0400` in the long syntax of `secrets`. A Compose secret read from a
file is bind-mounted and keeps the owner and mode of that file on the host.

## Backing up the keyset

**Losing the keyset loses every credential it protects.** Back it up, and keep
that copy apart from the database backups: a backup holding both is no better
than a database in clear.

## Enabling encryption

```
secret-keyset-file=/etc/happydomain/secret-keyset.json
secret-policy=instance
```

or `HAPPYDOMAIN_SECRET_KEYSET_FILE` and `HAPPYDOMAIN_SECRET_POLICY`.

From then on, every credential saved is encrypted. Those already stored are
encrypted the next time their provider is saved; to do it at once, see
[Migrating existing credentials](#migrating-existing-credentials).

Nothing changes for users.

## Startup checks

happyDomain refuses to start rather than fail provider by provider when:

- the `instance` policy is chosen without a keyset;
- the keyset does not open the check record stored in the database the first
  time a keyset was used with it: this is not the keyset the credentials were
  encrypted with;
- credentials are encrypted in the database but no keyset is configured.

## Migrating existing credentials

The operations below are on the admin interface, by default the
`happydomain.sock` socket:

```
curl --unix-socket happydomain.sock http://localhost/api/secrets/status
curl --unix-socket happydomain.sock -X POST http://localhost/api/secrets/reseal
```

`/api/secrets/status` counts the credentials of providers:

- `clear`: stored in clear;
- `sealed`: encrypted, and decrypting;
- `unreadable`: encrypted, but no longer decrypting;
- `undecodable`: providers that could not be looked at, such as a provider of a
  type this version no longer knows. They may hold encrypted credentials.

`/api/secrets/reseal` stores every credential the way the current policy stores
new ones. It reports how many providers it looked at, changed and failed,
naming the failures under `errors`. A provider that fails is left as it was,
and the others are resealed. It can be interrupted, and running it again
resumes the work.

## Going back to clear

1. Set `secret-policy=plaintext`, keeping the keyset configured, and restart.
   Encrypted credentials still decrypt, and new ones are stored in clear.

2. Store the others in clear too, then check that none is left encrypted:

   ```
   curl --unix-socket happydomain.sock -X POST http://localhost/api/secrets/reseal
   curl --unix-socket happydomain.sock http://localhost/api/secrets/status
   ```

3. Delete the users' keys and the keyset check record:

   ```
   curl --unix-socket happydomain.sock -X POST http://localhost/api/secrets/drop-safes
   ```

   It refuses while a credential is still counted as `sealed` or `unreadable`,
   and while a provider is counted as `undecodable`: those would be lost for
   good. Reseal again, have the unreadable credentials entered again, or
   delete the providers concerned, then run it again.

4. Remove `secret-keyset-file` from the configuration, and restart.

Backups taken before still hold encrypted credentials and the users' keys:
keep a copy of the keyset as long as you may restore one of them.

## Backups and deleted accounts

The administrative backup contains the credentials as they are stored:
encrypted under the `instance` policy, along with the users' keys, and in clear
otherwise. It restores on the same instance, or on another one configured with
the same keyset: a user's key that the configured keyset does not open is not
restored, and the restore reports it. The data export a user downloads never
contains credentials.

Deleting a user, or a user deleting their account, deletes their keys: their
credentials no longer decrypt from the database. Administrative backups taken
before still hold those keys, and decrypt them with the keyset for as long as
both are kept. A key left behind, for instance when the database was briefly
unavailable, is deleted by the next `tidy`.

## What changes for API clients

The admin API no longer returns provider credentials: it shows `••••••••`
instead, as the user API does. The administrative backup (`/api/backup.json`)
is the exception: it returns them as stored. Sending `••••••••` back keeps the
stored value.
