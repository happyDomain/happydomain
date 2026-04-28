# Domain checkers

Three built-in checkers operate on domain-level WHOIS/RDAP data: expiry date
monitoring, EPP lock status verification, and contact consistency checks. All
three share a common `whois` observation collected once per run and reused
across rules.

All checkers apply to domains (`ApplyToDomain: true`) and run on a configurable
schedule (minimum 1 h, maximum 7 days, default 24 h, except `domain_expiry`
whose minimum is 12 h).

---

## domain_expiry

Checks whether a domain name is nearing its expiration date. Also exports a
`domain_expiry_days_remaining` metric (gauge, unit: days, label: `registrar`).

### Options

| Id              | Type | Default | Description                                               |
|-----------------|------|---------|-----------------------------------------------------------|
| `warning_days`  | uint | `30`    | Days before expiry to trigger a warning.                  |
| `critical_days` | uint | `7`     | Days before expiry to trigger a critical alert.           |

`critical_days` must be strictly less than `warning_days`.

### Rules

| Code              | Severity | Condition                                                  |
|-------------------|----------|------------------------------------------------------------|
| `whois_error`     | error    | WHOIS observation could not be retrieved.                  |
| `expiry_critical` | critical | Days remaining ≤ `critical_days`.                          |
| `expiry_warning`  | warning  | Days remaining ≤ `warning_days`.                           |
| `expiry_ok`       | ok       | Domain is not near expiration.                             |

---

## domain_lock

Verifies that a domain carries the expected EPP lock statuses as reported by
RDAP/WHOIS.

### Options

| Id                 | Type   | Default                      | Description                                                                           |
|--------------------|--------|------------------------------|---------------------------------------------------------------------------------------|
| `requiredStatuses` | string | `clientTransferProhibited`   | Comma-separated EPP status codes that must all be present (e.g. `clientTransferProhibited,clientUpdateProhibited`). |

### Rules

| Code           | Severity | Condition                                                       |
|----------------|----------|-----------------------------------------------------------------|
| `lock_error`   | error    | WHOIS observation could not be retrieved.                       |
| `lock_skipped` | unknown  | No required statuses configured.                                |
| `lock_missing` | critical | One or more required EPP statuses are absent from the domain.   |
| `lock_ok`      | ok       | All required EPP statuses are present.                          |

---

## domain_contact

Compares the registered domain contacts (registrant, admin, tech) against
user-supplied expected values. Detects privacy-redacted contacts and reports
them as informational rather than mismatches.

### Options

| Id                    | Type   | Default        | Description                                                                              |
|-----------------------|--------|----------------|------------------------------------------------------------------------------------------|
| `expectedName`        | string | *(unset)*      | Expected registrant name (case-insensitive). Skipped if empty.                           |
| `expectedOrganization`| string | *(unset)*      | Expected organization (case-insensitive). Skipped if empty.                              |
| `expectedEmail`       | string | *(unset)*      | Expected email address (case-insensitive). Skipped if empty.                             |
| `checkRoles`          | string | `registrant`   | Comma-separated roles to check. Allowed values: `registrant`, `admin`, `tech`.           |

At least one of `expectedName`, `expectedOrganization`, or `expectedEmail` must
be set, otherwise the check is skipped. All comparisons are case-insensitive.

### Rules

One finding is emitted per role in `checkRoles`.

| Code                | Severity | Condition                                                                           |
|---------------------|----------|-------------------------------------------------------------------------------------|
| `contact_error`     | error    | WHOIS observation could not be retrieved.                                           |
| `contact_skipped`   | unknown  | No expected values or no roles configured.                                          |
| `contact_missing`   | warning  | The role has no contact record in WHOIS.                                            |
| `contact_redacted`  | info     | Contact fields match a known privacy-protection pattern (redacted, WhoisGuard, …).  |
| `contact_mismatch`  | warning  | One or more expected fields differ from the WHOIS data.                             |
| `contact_ok`        | ok       | All configured expected fields match.                                               |
