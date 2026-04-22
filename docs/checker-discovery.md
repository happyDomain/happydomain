# Cross-checker discovery

This document describes the contract between the SDK types (`DiscoveryEntry`,
`DiscoveryPublisher`, `RelatedObservation`, `ObservationGetter.GetRelated`,
`ReportContext`) and the happyDomain host that implements the scheduler and
storage behind them.

It exists because the SDK alone does not answer the interesting questions:
*who* stores entries, *when* are related observations resolved, what happens
when a consumer is missing, how stale data is pruned, and so on. Checker
authors need to know this to write code that behaves correctly at the edges.

## Model

```
  ┌──────────────┐  Collect         ┌─────────────┐
  │  producer A  │ ───────────────▶ │   host      │
  │              │ ◀─── entries ─── │  (scheduler │
  └──────────────┘                  │   + store)  │
                                    │             │
  ┌──────────────┐  Collect (fed    │             │
  │  consumer B  │   entries via    │             │
  │              │   AutoFill…)    ◀│             │
  │              │ ─── observation─▶│             │
  └──────────────┘                  └─────────────┘
                                          │
                                          ▼
                            GetRelated / ReportContext.Related
                            on producer A's next evaluate / report
```

Two independent flows composed by the host:

1. **Publication.** `DiscoveryPublisher.DiscoverEntries` returns a set of
   `DiscoveryEntry` at the end of each `Collect`. The host replaces the
   previous set for `(producer, target)` atomically. Entries are opaque
   to the host beyond `(Type, Ref)`.
2. **Observation lineage.** When a consumer checker runs on the same target
   and its option `AutoFillDiscoveryEntries` is populated, the host passes
   it the entries it knows about. The consumer filters by `Type`, reads
   `Payload` under the corresponding contract, produces its observation,
   and includes per-entry references (matching `DiscoveryEntry.Ref`) in its
   output. The host indexes those references so that a subsequent
   `GetRelated` / `ReportContext.Related` call from the original producer
   can return them.

## Host responsibilities

- **Entry index.** Store entries keyed by `(producer checker id, target,
  entry type, ref)`. On each successful collection, compute the diff vs.
  the previous set and apply it atomically. The visible state must never be
  a mix of old and new entries.
- **Observation → entry linkage.** When a consumer stores an observation
  on behalf of a `Ref`, record that linkage. The producer's next
  `GetRelated(key)` query reads this index.
- **Filtering at `AutoFillDiscoveryEntries` fill time.** Give the consumer
  all entries known for the target, not a pre-filtered subset. The SDK does
  not know which types the consumer understands; the consumer is the only
  place that knows its own contract.
- **Garbage collection.** When an entry disappears from the producer's
  latest set, the observations that covered it become stale. Drop them at
  the next consumer cycle or keep them with a TTL — either is acceptable,
  but `GetRelated` must not return observations whose `Ref` no longer
  exists.
- **`CollectedAt` fidelity.** `RelatedObservation.CollectedAt` must be
  populated by the host so reporters can decide whether to trust stale
  data.
