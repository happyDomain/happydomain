// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package backup

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"git.happydns.org/happyDomain/internal/forms"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/storage"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
	providerUC "git.happydns.org/happyDomain/internal/usecase/provider"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	happydns "git.happydns.org/happyDomain/model"
)

// ChannelRedactor returns a copy of a notification channel stripped of its
// secrets, and must withhold the configuration it cannot tell secrets in.
type ChannelRedactor interface {
	RedactChannelForExport(ch *happydns.NotificationChannel) (*happydns.NotificationChannel, error)
}

type Usecase struct {
	store    storage.Storage
	secrets  *secret.Manager
	redactor ChannelRedactor
}

type Option func(*Usecase)

// WithChannelRedactor lets BackupUser export the notification channels. Without
// one, it exports none: they cannot leave in clear.
func WithChannelRedactor(r ChannelRedactor) Option {
	return func(u *Usecase) { u.redactor = r }
}

func NewUsecase(store storage.Storage, secrets *secret.Manager, opts ...Option) *Usecase {
	u := &Usecase{store: store, secrets: secrets}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

// jsonStreamer writes a JSON object incrementally to an underlying writer.
// It lets WriteBackup emit a complete backup without ever materialising the
// whole dataset in memory: each record is marshalled and flushed on its own,
// so only one record (plus a small fixed buffer) is resident at a time. This
// keeps the peak memory footprint bounded regardless of database size.
type jsonStreamer struct {
	w          *bufio.Writer
	err        error
	wroteField bool
}

func newJSONStreamer(w io.Writer) *jsonStreamer {
	return &jsonStreamer{w: bufio.NewWriterSize(w, 32*1024)}
}

func (s *jsonStreamer) raw(str string) {
	if s.err != nil {
		return
	}
	_, s.err = s.w.WriteString(str)
}

func (s *jsonStreamer) marshal(v any) {
	if s.err != nil {
		return
	}
	var b []byte
	b, s.err = json.Marshal(v)
	if s.err != nil {
		return
	}
	_, s.err = s.w.Write(b)
}

// beginField writes the leading comma (when needed), the quoted field name and
// the colon separator.
func (s *jsonStreamer) beginField(name string) {
	if s.wroteField {
		s.raw(",")
	}
	s.wroteField = true
	s.marshal(name)
	s.raw(":")
}

// arrayField opens a JSON array field and returns a helper to append elements.
func (s *jsonStreamer) arrayField(name string) *arrayWriter {
	s.beginField(name)
	s.raw("[")
	return &arrayWriter{s: s, first: true}
}

type arrayWriter struct {
	s     *jsonStreamer
	first bool
}

func (a *arrayWriter) item(v any) {
	if !a.first {
		a.s.raw(",")
	}
	a.first = false
	a.s.marshal(v)
}

func (a *arrayWriter) end() { a.s.raw("]") }

// streamIteratorField writes the content of a store iterator as a JSON array
// field, encoding and discarding one item at a time. Used for the global (non
// user-scoped) tables, which are also the ones that grow without bound. When
// keep is non-nil only the items it accepts are emitted, which lets a per-user
// export reuse the same global iterators filtered down to one owner.
func streamIteratorField[T any](s *jsonStreamer, name, label string, it happydns.Iterator[T], err error, addErr func(string), keep func(*T) bool) {
	a := s.arrayField(name)
	defer a.end()

	if err != nil {
		addErr(fmt.Sprintf("unable to retrieve %s: %s", label, err.Error()))
		return
	}
	defer it.Close()

	for it.Next() {
		item := it.Item()
		if keep == nil || keep(item) {
			a.item(item)
		}
	}
	if err := it.Err(); err != nil {
		addErr(fmt.Sprintf("unable to retrieve every %s: %s", label, err.Error()))
	}
}

// streamSliceField writes an already-materialised slice as a JSON array field.
// Used where the store exposes an indexed per-user lookup instead of an
// iterator.
func streamSliceField[T any](s *jsonStreamer, name, label string, items []T, err error, addErr func(string)) {
	a := s.arrayField(name)
	defer a.end()

	if err != nil {
		addErr(fmt.Sprintf("unable to retrieve %s: %s", label, err.Error()))
		return
	}
	for _, item := range items {
		a.item(item)
	}
}

// userSeq invokes fn for each user that should appear in a backup. The
// full-system backup walks every user lazily (one at a time); a single-user
// export calls fn exactly once.
type userSeq func(fn func(*happydns.User))

// forEachUser walks every user lazily. The user-scoped arrays each require
// their own pass (a flat JSON object cannot interleave several top-level
// arrays), which trades a few extra table scans for a bounded memory
// footprint: only one user's data is held at a time.
func (u *Usecase) forEachUser(addErr func(string), fn func(*happydns.User)) {
	it, err := u.store.ListAllUsers()
	if err != nil {
		addErr(fmt.Sprintf("unable to retrieve Users: %s", err.Error()))
		return
	}
	defer it.Close()
	for it.Next() {
		fn(it.Item())
	}
}

// redactProviderMessage returns a copy of pm whose `secret`-tagged settings
// have been replaced by happydns.RedactedSecret. The original is left alone: it
// comes straight from the store and the caller does not own it.
func redactProviderMessage(pm *happydns.ProviderMessage) (*happydns.ProviderMessage, error) {
	p, err := providerUC.ParseProvider(pm)
	if err != nil {
		return nil, err
	}

	forms.RedactSecrets(p.Provider)

	msg, err := p.ToMessage()
	if err != nil {
		return nil, err
	}

	return &msg, nil
}

// streamUserScoped writes the Users, Domains, DomainsLogs, Zones, Providers
// and Sessions fields, all derived from the users yielded by each. WriteBackup feeds it every user; WriteBackupUser feeds it a
// single one. When stripSessionID is set the live session IDs are blanked out
// (per-user export), as they are credentials rather than portable data. When
// redactProviders is set, provider secret settings are replaced by
// happydns.RedactedSecret before being written: this file leaves happyDomain,
// so it must not carry anything that still opens a door. The administrative
// WriteBackup keeps them, as Restore has to put working providers back.
func (u *Usecase) streamUserScoped(s *jsonStreamer, addErr func(string), stripSessionID, redactProviders bool, each userSeq) {
	// Users.
	users := s.arrayField("Users")
	each(func(user *happydns.User) { users.item(user) })
	users.end()

	// Domains.
	domains := s.arrayField("Domains")
	each(func(user *happydns.User) {
		ds, err := u.store.ListDomains(user)
		if err != nil {
			addErr(fmt.Sprintf("unable to retrieve Domain names of %s (%s): %s", user.Id.String(), user.Email, err.Error()))
			return
		}
		for _, dn := range ds {
			domains.item(dn)
		}
	})
	domains.end()

	// Domain logs (keyed by domain id). ListDomains errors were already
	// reported during the Domains pass, so they are skipped silently here.
	s.beginField("DomainsLogs")
	s.raw("{")
	firstLog := true
	each(func(user *happydns.User) {
		ds, err := u.store.ListDomains(user)
		if err != nil {
			return
		}
		for _, dn := range ds {
			ls, logErr := u.store.ListDomainLogs(dn)
			if logErr != nil {
				addErr(fmt.Sprintf("unable to retrieve domain's logs %s/%s (%s): %s", user.Id.String(), dn.Id.String(), dn.DomainName, logErr.Error()))
				continue
			}
			if !firstLog {
				s.raw(",")
			}
			firstLog = false
			s.marshal(dn.Id.String())
			s.raw(":")
			s.marshal(ls)
		}
	})
	s.raw("}")

	// Zones (referenced through each domain's history).
	zones := s.arrayField("Zones")
	each(func(user *happydns.User) {
		ds, err := u.store.ListDomains(user)
		if err != nil {
			return
		}
		for _, dn := range ds {
			for _, zid := range dn.ZoneHistory {
				z, zoneErr := u.store.GetZone(zid)
				if zoneErr != nil {
					addErr(fmt.Sprintf("unable to retrieve domain's zone %s/%s (%s): zoneid=%s: %s", user.Id.String(), dn.Id.String(), dn.DomainName, zid.String(), zoneErr.Error()))
					continue
				}
				zones.item(z)
			}
		}
	})
	zones.end()

	// Providers.
	providers := s.arrayField("Providers")
	each(func(user *happydns.User) {
		ps, err := u.store.ListProviders(user)
		if err != nil {
			addErr(fmt.Sprintf("unable to retrieve Providers: %s", err.Error()))
			return
		}
		for _, p := range ps {
			if !redactProviders {
				providers.item(p)
				continue
			}

			redacted, err := redactProviderMessage(p)
			if err != nil {
				addErr(fmt.Sprintf("unable to redact Provider %s: %s", p.Id.String(), err.Error()))
				// Fail closed: an unredactable body is exported without its
				// settings rather than in clear.
				stripped := *p
				stripped.Provider = json.RawMessage("{}")
				providers.item(&stripped)
				continue
			}
			providers.item(redacted)
		}
	})
	providers.end()

	// Sessions.
	sessions := s.arrayField("Sessions")
	each(func(user *happydns.User) {
		ss, err := u.store.ListUserSessions(user.Id)
		if err != nil {
			addErr(fmt.Sprintf("unable to retrieve Sessions: %s", err.Error()))
			return
		}
		for _, ses := range ss {
			if stripSessionID {
				ses.Id = ""
			}
			sessions.item(ses)
		}
	})
	sessions.end()
}

// WriteBackup streams a complete backup as JSON directly to w. It produces the
// same document shape as Backup (consumable by Restore) but with a memory
// footprint independent of database size, making it safe on memory-constrained
// hosts. Per-record retrieval errors are collected and emitted in the trailing
// "Errors" field, mirroring Backup. The returned error is non-nil only on a
// write/marshal failure (e.g. the client disconnects); since output is streamed
// the HTTP status has already been sent and cannot be changed at that point.
func (u *Usecase) WriteBackup(w io.Writer) error {
	s := newJSONStreamer(w)
	var errs []string
	addErr := func(msg string) { errs = append(errs, msg) }

	s.raw("{")

	s.beginField("Version")
	s.marshal(u.store.SchemaVersion())

	// UserAuth.
	uai, err := u.store.ListAllAuthUsers()
	streamIteratorField(s, "UsersAuth", "AuthUsers", uai, err, addErr, nil)

	// Safes first, as RestoreReader applies fields in document order: the
	// sealed values of the providers only open with them, and with
	// the instance keyset, which is not part of the backup.
	safeIter, err := u.store.ListAllSafes()
	streamIteratorField(s, "Safes", "Safes", safeIter, err, addErr, nil)

	// The keystore each owner chose for new secrets.
	if defaults, err := u.store.ListDefaultKeystores(); err != nil {
		addErr(fmt.Sprintf("unable to retrieve default keystores: %s", err.Error()))
	} else if len(defaults) > 0 {
		s.beginField("DefaultKeystores")
		s.marshal(defaults)
	}

	// User-scoped tables, one user at a time.
	u.streamUserScoped(s, addErr, false, false, func(fn func(*happydns.User)) {
		u.forEachUser(addErr, fn)
	})

	watchIter, err := u.store.ListAllDomainWatches()
	streamIteratorField(s, "DomainWatches", "DomainWatches", watchIter, err, addErr, nil)

	// Zone templates, of the users and global.
	tplIter, err := u.store.ListAllZoneTemplates()
	streamIteratorField(s, "ZoneTemplates", "ZoneTemplates", tplIter, err, addErr, nil)

	// Notification tables: channels as stored, preferences, states
	// (acknowledgements included) and the history of what was sent.
	chIter, err := u.store.ListAllChannels()
	streamIteratorField(s, "NotificationChannels", "NotificationChannels", chIter, err, addErr, nil)

	prefIter, err := u.store.ListAllPreferences()
	streamIteratorField(s, "NotificationPreferences", "NotificationPreferences", prefIter, err, addErr, nil)

	stIter, err := u.store.ListAllStates()
	streamIteratorField(s, "NotificationStates", "NotificationStates", stIter, err, addErr, nil)

	recIter, err := u.store.ListAllRecords()
	streamIteratorField(s, "NotificationRecords", "NotificationRecords", recIter, err, addErr, nil)

	// Global, unbounded tables: streamed one record at a time.
	cfgIter, err := u.store.ListAllCheckerConfigurations()
	streamIteratorField(s, "CheckerConfigurations", "CheckerConfigurations", cfgIter, err, addErr, nil)

	planIter, err := u.store.ListAllCheckPlans()
	streamIteratorField(s, "CheckPlans", "CheckPlans", planIter, err, addErr, nil)

	evalIter, err := u.store.ListAllEvaluations()
	streamIteratorField(s, "CheckEvaluations", "CheckEvaluations", evalIter, err, addErr, nil)

	execIter, err := u.store.ListAllExecutions()
	streamIteratorField(s, "Executions", "Executions", execIter, err, addErr, nil)

	entryIter, err := u.store.ListAllDiscoveryEntries()
	streamIteratorField(s, "DiscoveryEntries", "DiscoveryEntries", entryIter, err, addErr, nil)

	refIter, err := u.store.ListAllDiscoveryObservationRefs()
	streamIteratorField(s, "DiscoveryObservationRefs", "DiscoveryObservationRefs", refIter, err, addErr, nil)

	snapIter, err := u.store.ListAllSnapshots()
	streamIteratorField(s, "ObservationSnapshots", "ObservationSnapshots", snapIter, err, addErr, nil)

	// Errors collected along the way, emitted last so they can be gathered
	// while streaming.
	s.beginField("Errors")
	s.marshal(errs)

	s.raw("}")

	if s.err != nil {
		return s.err
	}
	return s.w.Flush()
}

// WriteBackupUser streams a single user's data export as JSON directly to w,
// with the same memory benefit as WriteBackup. It produces the same document
// shape (consumable by Restore) but scoped to one user: credentials and live
// session IDs are stripped, and the global tables are filtered to records owned
// by the user. Observation snapshots are intentionally omitted, matching the
// resources exposed by the GDPR export. Per-record errors are collected into
// the trailing "Errors" field; a non-nil return means the stream broke after
// the HTTP status was already sent.
func (u *Usecase) WriteBackupUser(w io.Writer, user *happydns.User) error {
	uid := user.Id.String()
	s := newJSONStreamer(w)
	var errs []string
	addErr := func(msg string) { errs = append(errs, msg) }

	s.raw("{")

	s.beginField("Version")
	s.marshal(u.store.SchemaVersion())

	// UserAuth for this user — strip credentials before export.
	auth := s.arrayField("UsersAuth")
	if ua, err := u.store.GetAuthUser(user.Id); err == nil {
		ua.Password = nil
		ua.PasswordRecoveryKey = nil
		auth.item(ua)
	} else if !errors.Is(err, happydns.ErrAuthUserNotFound) {
		addErr(fmt.Sprintf("unable to retrieve UserAuth: %s", err.Error()))
	}
	auth.end()

	// User-scoped tables for this single user; strip live session IDs.
	u.streamUserScoped(s, addErr, true, true, func(fn func(*happydns.User)) {
		fn(user)
	})

	watches, err := u.store.ListDomainWatchesByUser(user.Id)
	streamSliceField(s, "DomainWatches", "DomainWatches", watches, err, addErr)

	// Only their own: the global templates belong to no one.
	templates, err := u.store.ListZoneTemplatesByUser(user.Id)
	streamSliceField(s, "ZoneTemplates", "ZoneTemplates", templates, err, addErr)

	// Notification channels, minus their secrets, and the preferences, states
	// and records attached to them.
	channels := s.arrayField("NotificationChannels")
	if u.redactor != nil {
		if chs, err := u.store.ListChannelsByUser(user.Id); err != nil {
			addErr(fmt.Sprintf("unable to retrieve NotificationChannels: %s", err.Error()))
		} else {
			for _, ch := range chs {
				redacted, err := u.redactor.RedactChannelForExport(ch)
				if err != nil {
					addErr(fmt.Sprintf("unable to redact NotificationChannel %s: %s", ch.Id.String(), err.Error()))
					continue
				}
				channels.item(redacted)
			}
		}
	}
	channels.end()

	prefs, err := u.store.ListPreferencesByUser(user.Id)
	streamSliceField(s, "NotificationPreferences", "NotificationPreferences", prefs, err, addErr)

	states, err := u.store.ListStatesByUser(user.Id)
	streamSliceField(s, "NotificationStates", "NotificationStates", states, err, addErr)

	recs, err := u.store.ListRecordsByUser(user.Id, 0)
	streamSliceField(s, "NotificationRecords", "NotificationRecords", recs, err, addErr)

	// Global tables filtered to records owned by this user. The checker
	// options are withheld as the API does: the export leaves happyDomain.
	cfgIter, err := u.store.ListAllCheckerConfigurations()
	streamIteratorField(s, "CheckerConfigurations", "CheckerConfigurations", cfgIter, err, addErr, func(cfg *happydns.CheckerOptionsPositional) bool {
		if cfg.UserId == nil || !cfg.UserId.Equals(user.Id) {
			return false
		}
		cfg.Options = checkerUC.RedactOptions(cfg.CheckName, cfg.Options)
		return true
	})

	// Check plans and executions use indexed per-user lookups.
	plans, err := u.store.ListCheckPlansByUser(user.Id)
	streamSliceField(s, "CheckPlans", "CheckPlans", plans, err, addErr)

	evalIter, err := u.store.ListAllEvaluations()
	streamIteratorField(s, "CheckEvaluations", "CheckEvaluations", evalIter, err, addErr, func(eval *happydns.CheckEvaluation) bool {
		return eval.Target.UserId == uid
	})

	execs, err := u.store.ListExecutionsByUser(user.Id, 0, nil)
	streamSliceField(s, "Executions", "Executions", execs, err, addErr)

	entryIter, err := u.store.ListAllDiscoveryEntries()
	streamIteratorField(s, "DiscoveryEntries", "DiscoveryEntries", entryIter, err, addErr, func(entry *happydns.StoredDiscoveryEntry) bool {
		return entry.Target.UserId == uid
	})

	refIter, err := u.store.ListAllDiscoveryObservationRefs()
	streamIteratorField(s, "DiscoveryObservationRefs", "DiscoveryObservationRefs", refIter, err, addErr, func(ref *happydns.DiscoveryObservationRef) bool {
		return ref.Target.UserId == uid
	})

	s.beginField("Errors")
	s.marshal(errs)

	s.raw("}")

	if s.err != nil {
		return s.err
	}
	return s.w.Flush()
}

// The restore* helpers each apply a single decoded record to the store. They
// are shared by the in-memory Restore and the streaming RestoreReader so both
// paths agree on how every record type is persisted.

func (u *Usecase) restoreAuthUser(ua *happydns.UserAuth) error {
	return u.store.UpdateAuthUser(ua)
}

func (u *Usecase) restoreUser(user *happydns.User) error {
	return u.store.CreateOrUpdateUser(user)
}

func (u *Usecase) restoreProvider(ctx context.Context, provider *happydns.ProviderMessage) error {
	p, err := providerUC.ParseProvider(provider)
	if err != nil {
		return err
	}

	// Sealed values are restored as they are; clear ones, from a backup
	// taken before sealing existed, are sealed under the current policy.
	if err := u.secrets.SealObjectInto(ctx, providerUC.SecretContext(p), p.Provider, secret.Destination{}.UnderPolicy()); err != nil {
		return fmt.Errorf("provider %s: %w", p.Id.String(), err)
	}

	return u.store.UpdateProvider(p)
}

func (u *Usecase) restoreDomain(domain *happydns.Domain) error {
	return u.store.UpdateDomain(domain)
}

// restoreDomainLogs applies every log of a single domain. Only the domain id is
// needed to key the logs, so the full Domain need not be reconstructed.
func (u *Usecase) restoreDomainLogs(domainID string, logs []*happydns.DomainLog) error {
	id, err := happydns.NewIdentifierFromString(domainID)
	if err != nil {
		return fmt.Errorf("invalid domain id %q in DomainsLogs: %w", domainID, err)
	}
	d := &happydns.Domain{Id: id}

	var errs error
	for _, l := range logs {
		errs = errors.Join(errs, u.store.UpdateDomainLog(d, l))
	}
	return errs
}

func (u *Usecase) restoreDomainWatch(w *happydns.DomainWatch) error {
	if w == nil {
		return nil
	}
	return u.store.RestoreDomainWatch(w)
}

func (u *Usecase) restoreZoneTemplate(t *happydns.ZoneTemplateMessage) error {
	if t == nil {
		return nil
	}
	return u.store.RestoreZoneTemplate(t)
}

func (u *Usecase) restoreZone(zmsg *happydns.ZoneMessage) error {
	zone, err := zoneUC.ParseZone(zmsg)
	if err != nil {
		return err
	}
	return u.store.UpdateZone(zone)
}

func (u *Usecase) restoreSession(session *happydns.Session) error {
	return u.store.UpdateSession(session)
}

func (u *Usecase) restoreCheckerConfiguration(ctx context.Context, cfg *happydns.CheckerOptionsPositional) error {
	if cfg == nil {
		return nil
	}
	// After the safes: secret options in clear are sealed.
	opts, err := checkerUC.RestoreOptions(ctx, u.secrets, cfg.CheckName, cfg.UserId, cfg.DomainId, cfg.ServiceId, cfg.Options)
	if err != nil {
		return fmt.Errorf("options of %s: %w", cfg.CheckName, err)
	}
	return u.store.UpdateCheckerConfiguration(cfg.CheckName, cfg.UserId, cfg.DomainId, cfg.ServiceId, opts)
}

func (u *Usecase) restoreChannel(ch *happydns.NotificationChannel) error {
	if ch == nil {
		return nil
	}
	return u.store.RestoreChannel(ch)
}

func (u *Usecase) restorePreference(pref *happydns.NotificationPreference) error {
	if pref == nil {
		return nil
	}
	return u.store.RestorePreference(pref)
}

func (u *Usecase) restoreState(state *happydns.NotificationState) error {
	if state == nil {
		return nil
	}
	return u.store.PutState(state)
}

func (u *Usecase) restoreNotificationRecord(rec *happydns.NotificationRecord) error {
	if rec == nil {
		return nil
	}
	return u.store.RestoreRecord(rec)
}

// restoreDefaultKeystores applies the keystore each owner chose, once their
// safes are back.
func (u *Usecase) restoreDefaultKeystores(defaults map[string]string) error {
	var errs error
	for owner, ref := range defaults {
		id, err := happydns.NewIdentifierFromString(owner)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("default keystore of %q: %w", owner, err))
			continue
		}
		errs = errors.Join(errs, u.store.PutDefaultKeystore(id, ref))
	}
	return errs
}

func (u *Usecase) restoreCheckPlan(plan *happydns.CheckPlan) error {
	return u.store.RestoreCheckPlan(plan)
}

func (u *Usecase) restoreEvaluation(eval *happydns.CheckEvaluation) error {
	return u.store.RestoreEvaluation(eval)
}

func (u *Usecase) restoreExecution(exec *happydns.Execution) error {
	return u.store.RestoreExecution(exec)
}

func (u *Usecase) restoreDiscoveryEntry(entry *happydns.StoredDiscoveryEntry) error {
	return u.store.RestoreDiscoveryEntry(entry)
}

func (u *Usecase) restoreDiscoveryObservationRef(ref *happydns.DiscoveryObservationRef) error {
	return u.store.RestoreDiscoveryObservationRef(ref)
}

func (u *Usecase) restoreSnapshot(snap *happydns.ObservationSnapshot) error {
	if err := u.store.RestoreSnapshot(snap); err != nil {
		return err
	}

	var errs error
	for key := range snap.Data {
		existing, _ := u.store.GetCachedObservation(snap.Target, key)
		if existing == nil || snap.CollectedAt.After(existing.CollectedAt) {
			if err := u.store.PutCachedObservation(snap.Target, key, &happydns.ObservationCacheEntry{
				SnapshotID:  snap.Id,
				CollectedAt: snap.CollectedAt,
			}); err != nil {
				errs = errors.Join(errs, err)
			}
		}
	}
	return errs
}

// RestoreReader streams a backup from r, decoding and applying one record at a
// time so the whole dataset is never held in memory. This makes restores safe
// on memory-constrained hosts, mirroring WriteBackup on the export side: a
// previous version decoded the entire document into a happydns.Backup first,
// which could exhaust memory long before any record was applied.
//
// Records are applied in the order their fields appear in the document, which
// is the order WriteBackup emits them. This ordering carries the dependencies
// between record types: Version is validated first, check plans land before the
// evaluations that reference them, and observation refs before the snapshots
// restored last. Per-record store errors are collected and returned joined; a
// malformed document instead returns the underlying json error (e.g.
// *json.SyntaxError) so callers can tell client input apart from restore
// failures.
func (u *Usecase) RestoreReader(ctx context.Context, r io.Reader) error {
	dec := json.NewDecoder(r)

	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return fmt.Errorf("invalid backup: expected a JSON object")
	}

	var errs error
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return err
		}
		key, _ := keyTok.(string)

		switch key {
		case "Version":
			var v int
			if err := dec.Decode(&v); err != nil {
				return err
			}
			if current := u.store.SchemaVersion(); v != 0 && v != current {
				return fmt.Errorf("backup schema version %d does not match current database schema version %d", v, current)
			}
		case "UsersAuth":
			errs = errors.Join(errs, decodeArray(dec, u.restoreAuthUser))
		case "Users":
			errs = errors.Join(errs, decodeArray(dec, u.restoreUser))
		case "Safes":
			// Before the providers: sealing a clear value restored below
			// must find the owner's safe rather than create another one.
			errs = errors.Join(errs, decodeArray(dec, func(safe *happydns.Safe) error { return u.restoreSafe(ctx, safe) }))
		case "DefaultKeystores":
			var defaults map[string]string
			if err := dec.Decode(&defaults); err != nil {
				return err
			}
			errs = errors.Join(errs, u.restoreDefaultKeystores(defaults))
		case "Providers":
			errs = errors.Join(errs, decodeArray(dec, func(pm *happydns.ProviderMessage) error { return u.restoreProvider(ctx, pm) }))
		case "Domains":
			errs = errors.Join(errs, decodeArray(dec, u.restoreDomain))
		case "DomainsLogs":
			errs = errors.Join(errs, u.decodeDomainsLogs(dec))
		case "Zones":
			errs = errors.Join(errs, decodeArray(dec, u.restoreZone))
		case "Sessions":
			errs = errors.Join(errs, decodeArray(dec, u.restoreSession))
		case "DomainWatches":
			errs = errors.Join(errs, decodeArray(dec, u.restoreDomainWatch))
		case "ZoneTemplates":
			errs = errors.Join(errs, decodeArray(dec, u.restoreZoneTemplate))
		case "NotificationChannels":
			errs = errors.Join(errs, decodeArray(dec, u.restoreChannel))
		case "NotificationPreferences":
			errs = errors.Join(errs, decodeArray(dec, u.restorePreference))
		case "NotificationStates":
			errs = errors.Join(errs, decodeArray(dec, u.restoreState))
		case "NotificationRecords":
			errs = errors.Join(errs, decodeArray(dec, u.restoreNotificationRecord))
		case "CheckerConfigurations":
			errs = errors.Join(errs, decodeArray(dec, func(cfg *happydns.CheckerOptionsPositional) error { return u.restoreCheckerConfiguration(ctx, cfg) }))
		case "CheckPlans":
			errs = errors.Join(errs, decodeArray(dec, u.restoreCheckPlan))
		case "CheckEvaluations":
			errs = errors.Join(errs, decodeArray(dec, u.restoreEvaluation))
		case "Executions":
			errs = errors.Join(errs, decodeArray(dec, u.restoreExecution))
		case "DiscoveryEntries":
			errs = errors.Join(errs, decodeArray(dec, u.restoreDiscoveryEntry))
		case "DiscoveryObservationRefs":
			errs = errors.Join(errs, decodeArray(dec, u.restoreDiscoveryObservationRef))
		case "ObservationSnapshots":
			errs = errors.Join(errs, decodeArray(dec, u.restoreSnapshot))
		default:
			// Unknown field (e.g. the export-only "Errors"): discard its
			// value without materialising it.
			if err := skipValue(dec); err != nil {
				return err
			}
		}
	}

	// Consume the closing '}'.
	if _, err := dec.Token(); err != nil {
		return err
	}

	return errs
}

func (u *Usecase) restoreSafe(ctx context.Context, safe *happydns.Safe) error {
	if safe == nil {
		return nil
	}

	// Restored with another keyset, or none, it would never open.
	if err := u.secrets.CheckSafe(ctx, safe); err != nil {
		return fmt.Errorf("safe %s not restored: %w", safe.Id.String(), err)
	}

	_, err := u.store.GetSafe(safe.Id)
	if errors.Is(err, happydns.ErrSafeNotFound) {
		return u.store.CreateSafe(safe)
	}
	if err != nil {
		return err
	}
	return u.store.UpdateSafe(safe)
}

// decodeArray reads a JSON array field element by element, decoding each into a
// fresh T and applying it immediately, so only one element is resident at a
// time. A decode/structural error stops the array and is returned unwrapped (so
// callers can inspect its concrete json type); apply errors are joined.
func decodeArray[T any](dec *json.Decoder, apply func(T) error) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if tok == nil {
		// JSON null in place of the array.
		return nil
	}
	if d, ok := tok.(json.Delim); !ok || d != '[' {
		return fmt.Errorf("invalid backup: expected a JSON array")
	}

	var errs error
	for dec.More() {
		var item T
		if err := dec.Decode(&item); err != nil {
			return err
		}
		if err := apply(item); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	// Consume the closing ']'.
	if _, err := dec.Token(); err != nil {
		return err
	}
	return errs
}

// decodeDomainsLogs reads the DomainsLogs object (domain id -> logs), applying
// each domain's logs as it is decoded. A single domain's logs are bounded, so
// decoding them as one slice keeps the footprint small.
func (u *Usecase) decodeDomainsLogs(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if tok == nil {
		return nil
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return fmt.Errorf("invalid backup: DomainsLogs must be a JSON object")
	}

	var errs error
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return err
		}
		domainID, _ := keyTok.(string)

		var logs []*happydns.DomainLog
		if err := dec.Decode(&logs); err != nil {
			return err
		}
		errs = errors.Join(errs, u.restoreDomainLogs(domainID, logs))
	}

	// Consume the closing '}'.
	if _, err := dec.Token(); err != nil {
		return err
	}
	return errs
}

// skipValue consumes and discards the next JSON value (scalar, array or object)
// without buffering nested structures into memory.
func skipValue(dec *json.Decoder) error {
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		if d, ok := tok.(json.Delim); ok {
			switch d {
			case '[', '{':
				depth++
			case ']', '}':
				depth--
			}
		}
		if depth == 0 {
			return nil
		}
	}
}
