# Adding a service to happyDomain

A **service** is how happyDomain presents a group of DNS records as one
meaningful thing: an SPF policy, a set of mail servers, a CAA policy, a
"domain for sale" statement. This page describes what a service is expected to
look like, what you have to write, and what happens on its own.

Everything a service needs lives in **two files trees, one folder each**:

- `services/<name>.go` (or `services/abstract/<name>.go`) for the backend;
- `web/src/lib/services/<svctype>/` for everything the user sees.

---

## The golden rule: the backend stores raw records, the frontend parses them

**A service body holds the DNS records as they appear in the zone.** Not the
values extracted from them, not a convenient abstraction over them: the records
themselves.

`services/forsale.go` (RFC 10023) is the reference:

```go
// ForSale advertises that the domain name is for sale (RFC 10023).
type ForSale struct {
	Records []*happydns.TXT `json:"txt"`
}
```

Everything the user manipulates in the editor, asking prices, messages, contact
URIs, broker codes, is parsed out of those TXT records **in the frontend**, by
`web/src/lib/services/svcs.ForSale/model.svelte.ts`, and serialized back into
them on save.

Why it matters:

- The zone can be listed and edited **record by record**, across every service,
  because every service is able to hand over its records verbatim. A record does
  not have to be reconstructed from abstract fields before it can be shown.
- Editing one record does not force a complex update path through a service that
  would have to recompute its abstract state, decide which record the change
  belongs to, and rebuild the rest of the RRset.
- Nothing is lost. Tags, parameters and oddities happyDomain does not know about
  survive a load/save round trip, because they were never dropped in the first
  place.

So: **avoid abstract fields whenever the record layout can be kept verbatim.**
A service that stores `Price float64` instead of the TXT record it came from is
the shape to avoid.

What Go may still parse: only what it needs to do its own job, namely claim the
right records in the analyzer and produce `GenComment()`. `ForSaleFields.Analyze`
exists for the comment line, not to become the storage format.

Abstract services (`services/abstract/`) are the deliberate exception: they exist
precisely to offer a higher-level view (a CalDAV server, an email autoconfig).
They remain the exception, not the model to follow for a new record-level
service.

---

## The backend part

One Go file, registering itself in its `init()`:

```go
func init() {
	svc.RegisterService(
		func() happydns.ServiceBody { return &ForSale{} },
		forsale_analyze,
		happydns.ServiceInfos{
			Name:       "Domain For Sale",
			Categories: []string{"service"},
			RecordTypes: []uint16{
				dns.TypeTXT,
			},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
				Single:    true,
				NeedTypes: []uint16{dns.TypeTXT},
			},
		},
		1,
	)
}
```

- The body implements `happydns.ServiceBody`: `GetNbResources()`, `GenComment()`
  and `GetRecords()`. `Initialize()` seeds a brand new service, and
  `MetadataEnricher` carries over what records cannot express.
- The analyzer claims the records that belong to the service, through
  `a.SearchRR(...)` then `a.UseRR(...)`. Records it does not claim fall through
  to the other services, down to the raw TXT or orphan service.
- `Type` is **not** yours to set: `RegisterService` overwrites it with the Go
  type name (`svcs.ForSale`, `abstract.CalDAV`), and that string is the service
  type used everywhere else, including as the name of the frontend folder.
- `Name` is the untranslated name, kept for API consumers and used as a last
  resort by the interface. **There is no `Description` field**: how a service is
  introduced to the user is wording, and wording is translated, so it lives with
  the frontend part (see below).

Nothing else. No list to append to, no central registry to edit.

---

## The frontend part

One folder, named after the service type:

```
web/src/lib/services/svcs.ForSale/
    editor.svelte         # the editor, shown when the service is edited
    editor.svelte.test.ts # mounts the editor, optional
    model.svelte.ts       # parsing and serializing the records
    model.test.ts
    compliance.ts         # RFC checks reported under the editor
    compliance.test.ts
    locales/en.json       # every string the service shows, English is required
    locales/fr.json       # any other language, optional
```

Every file is optional but the folder name: a service with a plain editor and no
parsing has only `editor.svelte` and `locales/en.json`; a service type with no
editor at all (it then falls back to the orphan editor) still owns a folder for
its translations.

Name the model `model.ts`, or `model.svelte.ts` when it uses runes. Tests sit
next to the file they test.

### Testing the editor

`*.test.ts` runs in Node and tests plain functions. A test whose name ends with
`.svelte.test.ts` runs in a simulated browser (jsdom) instead, and can mount a
component. `npm run test` runs both.

Such a test mounts the editor with
[Testing Library](https://testing-library.com/docs/svelte-testing-library/intro),
drives it the way a user would, and reads back the records the page would save.
`svcs.CAAPolicy/editor.svelte.test.ts` is the example to copy. Two things to
know before writing one:

- Load the translations first, with `await loadTranslations("en", "/")`, or
  every label reads back as its translation key.
- Declare the edited value with `$state`, which those files may use. An editor
  mutates its records in place, and a plain object handed to a component is
  proxied on its way in: the proxy does not write back to the object the test
  kept, so nothing would ever seem to change.

### What happens on its own

| Thing | How |
| --- | --- |
| The service is registered | `init()` in the Go file |
| `web/src/lib/services_specs.ts` | `go generate ./...` |
| The editor is found | `import.meta.glob("$lib/services/*/editor.svelte")` in `ServiceEditor.svelte`, keyed by folder name |
| The validators are registered | `import.meta.glob("$lib/services/*/compliance.ts", { eager: true })` in `EditorCompliance.svelte` |
| The translations are loaded | `import.meta.glob("./services/*/locales/*.json")` in `translations.ts`, merged into the application translations |

There is no shared file to edit when adding a service. If you find yourself
editing one, something is off.

### The i18n contract

| Key | Holds |
| --- | --- |
| `svcinfo.<svctype>.name` | The name shown in the service picker, cards, page titles |
| `svcinfo.<svctype>.description` | The one-line description shown under the name |
| `resources.<TAG>.*` | The strings of the editor |
| `compliance.<prefix>.<issue-id>.{title,detail}` | The compliance messages |

`<prefix>` must match the prefix of the issue ids the validator returns: a
validator emitting `forsale.value-too-long` is read as
`compliance.forsale.value-too-long.title`.

English is required, every other language is optional and falls back to English.
Interpolation parameters need no declaration: pass whatever the message uses.

### Compliance

`web/src/lib/services/compliance.ts` holds the framework:

```ts
registerValidators("svcs.ForSale", {
    sync: (raw, ctx) => forsaleSync(raw, ctx),          // on every keystroke
    async: (raw, ctx, signal) => forsaleAsync(raw, ctx, signal), // debounced, may query the resolver
});
```

A validator returns `ComplianceIssue[]`, each with a stable `id`, a `severity`,
optional `params`, `field` and `docUrl`. `ctx.findServices()` and
`ctx.findAllServices()` give access to the sibling services of the zone, which
is how cross-record checks (DMARC looking for DKIM, MTA-STS looking for MX) are
written. See [`compliance-tests-goals.md`](compliance-tests-goals.md).

---

## Checklist

1. Write `services/<name>.go`: the body holding the raw records, the analyzer,
   and the `init()` registration.
2. Write the Go tests, including a round trip through `services/roundtrip_test.go`.
3. Run `go generate ./...` and commit the regenerated
   `web/src/lib/services_specs.ts`.
4. Create `web/src/lib/services/<svctype>/`.
5. Add `locales/en.json` with at least `svcinfo.<svctype>.name` and
   `svcinfo.<svctype>.description`.
6. Add `model.ts` if the records need parsing, with its tests.
7. Add `editor.svelte`, using `resources.<TAG>.*` keys for its strings.
8. Add `compliance.ts` if the RFC has constraints worth reporting, with its
   tests and its `compliance.<prefix>.*` messages.
9. Run `npm run test` and `npm run check` in `web/`. The service tests in
   `web/src/lib/services/services.test.ts` check that the folder is named after
   a known service type, that the validators are registered under that name, and
   that the service is named and described in English.
