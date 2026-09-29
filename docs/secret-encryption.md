# Encrypting stored credentials

happyDomain stores credentials on behalf of its users: the API keys of their
DNS providers, the signing secret and the header values of their webhooks
(an `Authorization` token, say), the endpoint of their UnifiedPush channels,
and the checker options documented as secret, such as the API keys of the
blacklist sources. By
default they are stored in clear in the database, as they always were.

The `instance` policy encrypts them under a keyset held by the server, outside
the database. A leaked database dump or backup, or a database hosted apart from
happyDomain, no longer gives the credentials away. It does not protect against
a compromised happyDomain server, which holds the keyset.

The design is described in [secret-management.md](secret-management.md).

## Generating the keyset

```
happydomain secret-keyset generate /etc/happydomain/secret-keyset.json
```

The keyset is a cleartext [Tink](https://developers.google.com/tink) JSON
keyset, which `tinkey` can also read. The file is created readable by its owner
only; make sure that owner is the user happyDomain runs as. happyDomain warns
at startup when the file can be read by other users. The command never
overwrites an existing file.

With Docker, mount the file as a secret rather than baking it into an image,
readable by the `happydomain` user of the image (uid 15353) only. A Swarm
secret is mounted with mode `0444` unless told otherwise: set `uid: "15353"`
and `mode: 0400` in the long syntax of `secrets`. A Compose secret read from a
file is bind-mounted and keeps the owner and mode of that file on the host.

## Backing up the keyset

**Losing the keyset loses every credential it protects.** Back it up, and keep
that copy apart from the database backups: a backup holding both is no better
than a database in clear.

Back it up again after each rotation: once the users' keys are re-encrypted
under the new key, a copy of the keyset taken before no longer opens them.

## Enabling encryption

```
secret-keyset-file=/etc/happydomain/secret-keyset.json
secret-policy=instance
```

or `HAPPYDOMAIN_SECRET_KEYSET_FILE` and `HAPPYDOMAIN_SECRET_POLICY`.

From then on, every credential saved is encrypted. Those already stored are
encrypted the next time their provider, channel or options are saved; to do it
at once, see [Migrating existing credentials](#migrating-existing-credentials).

When several happyDomain processes share the database, give all of them the
same keyset and policy, and restart all of them, before migrating: a process
left on the previous configuration keeps storing credentials its own way.

Nothing changes for users.

## Startup checks

happyDomain refuses to start rather than fail provider by provider when:

- the `instance` policy is chosen without a keyset;
- the keyset does not open the check record stored in the database the first
  time a keyset was used with it: this is not the keyset the credentials were
  encrypted with;
- credentials are encrypted in the database but no keyset is configured;
- no keyset is configured, but a user's key record does not decode while the
  check record is there: it may still need the keyset. Keep the keyset
  configured until the record is dealt with, see
  [Damaged keys](#damaged-keys).

## Migrating existing credentials

The operations below are on the admin interface, by default the
`happydomain.sock` socket:

```
curl --unix-socket happydomain.sock http://localhost/api/secrets/status
curl --unix-socket happydomain.sock -X POST http://localhost/api/secrets/reseal
```

`/api/secrets/status` counts, for providers, notification channels and checker
options, the credentials stored in clear, those encrypted, and those that
cannot be decrypted. Records that cannot be looked at, such as a provider of a
type this version no longer knows, are counted as `undecodable` and some of
them named under `problems`. A credential whose user's key cannot be opened
with the current configuration, the keyset missing or lacking the key that
protects it, is not taken for one that cannot be decrypted: its record is
counted as `undecodable` and named under `problems` until the configuration is
repaired, and users are told to contact you rather than to enter it again. It
also lists the keys of the keyset, and how many users' keys each one protects.

Under `damagedSafes`, it lists the users' keys whose record no longer decodes,
along with their owner when known, and the number of credentials encrypted
with each: see [Damaged keys](#damaged-keys).

Which checker options are secret comes from the checkers themselves. For a
checker this version does not load, the options stored encrypted are still
counted and resealed, but those stored in clear cannot be told apart from the
others: they are neither counted nor encrypted. The API and the data export
withhold every option of such a checker.

`/api/secrets/reseal` stores every credential the way the current policy
stores new ones. It reports, for each type of object, what it changed, what it
skipped and what failed, naming some of them under `errors`:

- *skipped*: the object changed while it was being resealed, or belongs to a
  deleted account (the next `tidy` removes it). Run it again.
- *failed*: the object could not be resealed, for instance a credential that no
  longer decrypts. It is left as it was, and the other objects are resealed.

It can be interrupted, and running it again resumes the work. It only writes an
object back if nothing changed it since it was read, so it can run while users
are active, and with several happyDomain processes sharing the database,
provided they all run with the same keyset and policy.

## Rotating the keyset

1. Keep a copy of the current keyset file, then add a new key, which becomes
   the one used from then on. The previous keys stay in the keyset and keep
   decrypting what they encrypted:

   ```
   cp /etc/happydomain/secret-keyset.json /etc/happydomain/secret-keyset.json.before-rotation
   happydomain secret-keyset rotate /etc/happydomain/secret-keyset.json
   ```

   Back up the new file, as described in
   [Backing up the keyset](#backing-up-the-keyset).

2. Restart happyDomain, every process of it when several share the database:
   one still running with the previous file would keep protecting new users'
   keys with the previous key.

3. Re-encrypt the users' keys under the new key. The credentials themselves are
   not rewritten:

   ```
   curl --unix-socket happydomain.sock -X POST http://localhost/api/secrets/rewrap
   ```

   It reports, like a reseal, what it changed, skipped and failed. A user's key
   that fails, for instance because the key protecting it was already removed
   from the keyset, is left as it was, and the others are re-encrypted.

4. Check that the previous key protects nothing any more:

   ```
   curl --unix-socket happydomain.sock http://localhost/api/secrets/status
   happydomain secret-keyset info /etc/happydomain/secret-keyset.json
   ```

   It can then be removed with `tinkey delete-key`. A key still in use shows up
   as `MISSING` once removed, and what it protects no longer opens: put back
   the copy kept at step 1. Users' keys too damaged to tell which key protects
   them are counted under `UNREADABLE`.

## Going back to clear

1. Set `secret-policy=plaintext`, keeping the keyset configured, and restart.
   Encrypted credentials still decrypt, and new ones are stored in clear, as
   are the ones saved again.

2. Store the others in clear too, then check that none is left encrypted:

   ```
   curl --unix-socket happydomain.sock -X POST http://localhost/api/secrets/reseal
   curl --unix-socket happydomain.sock http://localhost/api/secrets/status
   ```

   A credential that can never be decrypted any more, counted as `unreadable`,
   is left as it is, and does not keep the others from being stored in clear.
   Its user has to enter it again.

3. Delete the users' keys and the keyset check record:

   ```
   curl --unix-socket happydomain.sock -X POST http://localhost/api/secrets/drop-safes
   ```

   It refuses while a credential is still encrypted, and while a record is
   counted as `undecodable`, as it may hold encrypted credentials: retry if
   the database was briefly unavailable, otherwise repair or delete the
   records named under `problems` first. Credentials that can never be
   decrypted any more do not stop it: they are lost already. A user's key it
   cannot delete, such as a damaged record, is left as it was, and so is a
   user's key protected by a key missing from the keyset, or disabled: what it
   protects decrypts again once that key is put back. The others are deleted,
   but the request then fails, naming what is left, and the keyset check record
   is kept: do not go on to step 4, the keyset stays required at startup. Put
   the missing key back, or repair the damaged record (see
   [Damaged keys](#damaged-keys)), then run it again.

4. Remove `secret-keyset-file` from the configuration, and restart.

Backups taken before still hold encrypted credentials and the users' keys:
keep a copy of the keyset as long as you may restore one of them. Restored
without it, happyDomain refuses to start.

When several happyDomain processes share the database, restart all of them with
`secret-policy=plaintext` before step 2.

## Damaged keys

A user's key whose record no longer decodes is damaged: a failing disk or a
manual edit corrupted it, since happyDomain refuses to run on a database
written by a later version. It does not repair itself, and the credentials
encrypted with it no longer decrypt.

`/api/secrets/status` lists them under `damagedSafes`: the `key` of the record,
the `id` of the user's key, its `owner` and `kind` when known, and under
`values` the number of credentials that decrypt again only if it is repaired.

While one is there, happyDomain keeps requiring the keyset at startup, and going
back to clear stops before deleting the users' keys. `tidy` does not delete
such a record, as it may be the only key to some credentials, unless it
belongs to a deleted user.

When an administrative backup (`/api/backup.json`) holds it, repair it:

```
curl --unix-socket happydomain.sock -X POST -H 'Content-Type: application/json' \
     --data @backup.json http://localhost/api/secrets/safes/<id>/repair
```

Any backup taken since that key was created will do, as long as the keyset
still holds the key that protected it then: keep the keys a rotation retires
while an older backup may be needed (see [Rotating the keyset](#rotating-the-keyset)).
The key of the backup is only put back if it decrypts with the keyset, and it
is bound to its `id` and `owner`: no other key can take the place of the
damaged one. The credentials encrypted with it then decrypt again.

A damaged key that no backup holds is lost. Forget it, giving up the
credentials encrypted with it:

```
curl --unix-socket happydomain.sock -X POST http://localhost/api/secrets/safes/<id>/forget
```

Its record is deleted, and those credentials are then counted as `unreadable`:
their users have to enter them again, and get a new key when they do. Forgetting
refuses a key that is not damaged. A record listed without `id` can only be
removed from the database by hand, under its `key`.

## Backups and deleted accounts

The administrative backup contains the credentials as they are stored:
encrypted under the `instance` policy, along with the users' keys, and in clear
otherwise. It restores on the same instance, or on another one configured with
the same keyset. The data export a user downloads never contains credentials.

Deleting a user, or a user deleting their account, deletes their keys: their
credentials no longer decrypt from the database. Administrative backups taken
before still hold those keys, and decrypt them with the keyset for as long as
both are kept. The notification channels of deleted users are removed by
`tidy`.

When the key of that user is a damaged record, it cannot be deleted: the user
is still deleted and logged out, but the deletion reports an error rather than
success, and the record stays. The next `tidy` deletes it: what it encrypted was
meant to go with the user anyway. That is the only damaged key `tidy` deletes,
told by the owner index; see [Damaged keys](#damaged-keys) for the others.

## What changes for API clients

The admin API no longer returns credentials: provider settings and checker
options show `••••••••` instead, as the user API does. The administrative
backup (`/api/backup.json`) is the exception: it returns them as stored.
Checker options documented as secret are now also withheld from the user API.
Sending `••••••••` back keeps the stored value.
