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

package app

import (
	"time"

	"git.happydns.org/happyDomain/internal/metrics"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

// instrumentedStorage wraps a storage.Storage to record Prometheus metrics for
// every operation.
type instrumentedStorage struct {
	inner storage.Storage
}

// newInstrumentedStorage wraps the given Storage with metrics instrumentation.
func newInstrumentedStorage(s storage.Storage) storage.Storage {
	return &instrumentedStorage{inner: s}
}

// observe records the duration and outcome of a storage operation.
func observeStorage(operation, entity string, start time.Time, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	metrics.StorageOperationsTotal.WithLabelValues(operation, entity, status).Inc()
	metrics.StorageOperationDuration.WithLabelValues(operation, entity).Observe(time.Since(start).Seconds())
}

// Schema / lifecycle

func (s *instrumentedStorage) SchemaVersion() int {
	return s.inner.SchemaVersion()
}

func (s *instrumentedStorage) MigrateSchema() error {
	return s.inner.MigrateSchema()
}

func (s *instrumentedStorage) Close() error {
	return s.inner.Close()
}

// AuthUser

func (s *instrumentedStorage) ListAllAuthUsers() (ret happydns.Iterator[happydns.UserAuth], err error) {
	start := time.Now()
	ret, err = s.inner.ListAllAuthUsers()
	observeStorage("list", "authuser", start, err)
	return
}

func (s *instrumentedStorage) GetAuthUser(id happydns.Identifier) (ret *happydns.UserAuth, err error) {
	start := time.Now()
	ret, err = s.inner.GetAuthUser(id)
	observeStorage("get", "authuser", start, err)
	return
}

func (s *instrumentedStorage) GetAuthUserByEmail(email string) (ret *happydns.UserAuth, err error) {
	start := time.Now()
	ret, err = s.inner.GetAuthUserByEmail(email)
	observeStorage("get", "authuser", start, err)
	return
}

func (s *instrumentedStorage) AuthUserExists(email string) (ret bool, err error) {
	start := time.Now()
	ret, err = s.inner.AuthUserExists(email)
	observeStorage("get", "authuser", start, err)
	return
}

func (s *instrumentedStorage) CreateAuthUser(user *happydns.UserAuth) (err error) {
	start := time.Now()
	err = s.inner.CreateAuthUser(user)
	observeStorage("create", "authuser", start, err)
	return
}

func (s *instrumentedStorage) UpdateAuthUser(user *happydns.UserAuth) (err error) {
	start := time.Now()
	err = s.inner.UpdateAuthUser(user)
	observeStorage("update", "authuser", start, err)
	return
}

func (s *instrumentedStorage) DeleteAuthUser(user *happydns.UserAuth) (err error) {
	start := time.Now()
	err = s.inner.DeleteAuthUser(user)
	observeStorage("delete", "authuser", start, err)
	return
}

func (s *instrumentedStorage) ClearAuthUsers() (err error) {
	start := time.Now()
	err = s.inner.ClearAuthUsers()
	observeStorage("delete", "authuser", start, err)
	return
}

// Domain

func (s *instrumentedStorage) ListAllDomains() (ret happydns.Iterator[happydns.Domain], err error) {
	start := time.Now()
	ret, err = s.inner.ListAllDomains()
	observeStorage("list", "domain", start, err)
	return
}

func (s *instrumentedStorage) ListDomains(user *happydns.User) (ret []*happydns.Domain, err error) {
	start := time.Now()
	ret, err = s.inner.ListDomains(user)
	observeStorage("list", "domain", start, err)
	return
}

func (s *instrumentedStorage) GetDomain(domainid happydns.Identifier) (ret *happydns.Domain, err error) {
	start := time.Now()
	ret, err = s.inner.GetDomain(domainid)
	observeStorage("get", "domain", start, err)
	return
}

func (s *instrumentedStorage) GetDomainByDN(user *happydns.User, fqdn string) (ret []*happydns.Domain, err error) {
	start := time.Now()
	ret, err = s.inner.GetDomainByDN(user, fqdn)
	observeStorage("get", "domain", start, err)
	return
}

func (s *instrumentedStorage) CreateDomain(domain *happydns.Domain) (err error) {
	start := time.Now()
	err = s.inner.CreateDomain(domain)
	observeStorage("create", "domain", start, err)
	return
}

func (s *instrumentedStorage) UpdateDomain(domain *happydns.Domain) (err error) {
	start := time.Now()
	err = s.inner.UpdateDomain(domain)
	observeStorage("update", "domain", start, err)
	return
}

func (s *instrumentedStorage) DeleteDomain(domainid happydns.Identifier) (err error) {
	start := time.Now()
	err = s.inner.DeleteDomain(domainid)
	observeStorage("delete", "domain", start, err)
	return
}

func (s *instrumentedStorage) ClearDomains() (err error) {
	start := time.Now()
	err = s.inner.ClearDomains()
	observeStorage("delete", "domain", start, err)
	return
}

// DomainLog

func (s *instrumentedStorage) ListAllDomainLogs() (ret happydns.Iterator[happydns.DomainLogWithDomainId], err error) {
	start := time.Now()
	ret, err = s.inner.ListAllDomainLogs()
	observeStorage("list", "domain_log", start, err)
	return
}

func (s *instrumentedStorage) ListDomainLogs(domain *happydns.Domain) (ret []*happydns.DomainLog, err error) {
	start := time.Now()
	ret, err = s.inner.ListDomainLogs(domain)
	observeStorage("list", "domain_log", start, err)
	return
}

func (s *instrumentedStorage) CreateDomainLog(domain *happydns.Domain, log *happydns.DomainLog) (err error) {
	start := time.Now()
	err = s.inner.CreateDomainLog(domain, log)
	observeStorage("create", "domain_log", start, err)
	return
}

func (s *instrumentedStorage) UpdateDomainLog(domain *happydns.Domain, log *happydns.DomainLog) (err error) {
	start := time.Now()
	err = s.inner.UpdateDomainLog(domain, log)
	observeStorage("update", "domain_log", start, err)
	return
}

func (s *instrumentedStorage) DeleteDomainLog(domain *happydns.Domain, log *happydns.DomainLog) (err error) {
	start := time.Now()
	err = s.inner.DeleteDomainLog(domain, log)
	observeStorage("delete", "domain_log", start, err)
	return
}

// Insight

func (s *instrumentedStorage) InsightsRun() (err error) {
	start := time.Now()
	err = s.inner.InsightsRun()
	observeStorage("run", "insight", start, err)
	return
}

func (s *instrumentedStorage) LastInsightsRun() (t *time.Time, id happydns.Identifier, err error) {
	start := time.Now()
	t, id, err = s.inner.LastInsightsRun()
	observeStorage("get", "insight", start, err)
	return
}

// Provider

func (s *instrumentedStorage) ListAllProviders() (ret happydns.Iterator[happydns.ProviderMessage], err error) {
	start := time.Now()
	ret, err = s.inner.ListAllProviders()
	observeStorage("list", "provider", start, err)
	return
}

func (s *instrumentedStorage) ListProviders(user *happydns.User) (ret happydns.ProviderMessages, err error) {
	start := time.Now()
	ret, err = s.inner.ListProviders(user)
	observeStorage("list", "provider", start, err)
	return
}

func (s *instrumentedStorage) GetProvider(prvdid happydns.Identifier) (ret *happydns.ProviderMessage, err error) {
	start := time.Now()
	ret, err = s.inner.GetProvider(prvdid)
	observeStorage("get", "provider", start, err)
	return
}

func (s *instrumentedStorage) CreateProvider(prvd *happydns.Provider) (err error) {
	start := time.Now()
	err = s.inner.CreateProvider(prvd)
	observeStorage("create", "provider", start, err)
	return
}

func (s *instrumentedStorage) UpdateProvider(prvd *happydns.Provider) (err error) {
	start := time.Now()
	err = s.inner.UpdateProvider(prvd)
	observeStorage("update", "provider", start, err)
	return
}

func (s *instrumentedStorage) DeleteProvider(prvdid happydns.Identifier) (err error) {
	start := time.Now()
	err = s.inner.DeleteProvider(prvdid)
	observeStorage("delete", "provider", start, err)
	return
}

func (s *instrumentedStorage) ClearProviders() (err error) {
	start := time.Now()
	err = s.inner.ClearProviders()
	observeStorage("delete", "provider", start, err)
	return
}

// Session

func (s *instrumentedStorage) ListAllSessions() (ret happydns.Iterator[happydns.Session], err error) {
	start := time.Now()
	ret, err = s.inner.ListAllSessions()
	observeStorage("list", "session", start, err)
	return
}

func (s *instrumentedStorage) GetSession(sessionid string) (ret *happydns.Session, err error) {
	start := time.Now()
	ret, err = s.inner.GetSession(sessionid)
	observeStorage("get", "session", start, err)
	return
}

func (s *instrumentedStorage) ListAuthUserSessions(user *happydns.UserAuth) (ret []*happydns.Session, err error) {
	start := time.Now()
	ret, err = s.inner.ListAuthUserSessions(user)
	observeStorage("list", "session", start, err)
	return
}

func (s *instrumentedStorage) ListUserSessions(userid happydns.Identifier) (ret []*happydns.Session, err error) {
	start := time.Now()
	ret, err = s.inner.ListUserSessions(userid)
	observeStorage("list", "session", start, err)
	return
}

func (s *instrumentedStorage) UpdateSession(session *happydns.Session) (err error) {
	start := time.Now()
	err = s.inner.UpdateSession(session)
	observeStorage("update", "session", start, err)
	return
}

func (s *instrumentedStorage) DeleteSession(sessionid string) (err error) {
	start := time.Now()
	err = s.inner.DeleteSession(sessionid)
	observeStorage("delete", "session", start, err)
	return
}

func (s *instrumentedStorage) ClearSessions() (err error) {
	start := time.Now()
	err = s.inner.ClearSessions()
	observeStorage("delete", "session", start, err)
	return
}

// CheckResult / CheckSchedule / CheckExecution

func (s *instrumentedStorage) ListCheckResults(checkName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, limit int) (ret []*happydns.CheckResult, err error) {
	start := time.Now()
	ret, err = s.inner.ListCheckResults(checkName, targetType, targetId, limit)
	observeStorage("list", "check_result", start, err)
	return
}

func (s *instrumentedStorage) ListCheckResultsByPlugin(userId happydns.Identifier, checkName string, limit int) (ret []*happydns.CheckResult, err error) {
	start := time.Now()
	ret, err = s.inner.ListCheckResultsByPlugin(userId, checkName, limit)
	observeStorage("list", "check_result", start, err)
	return
}

func (s *instrumentedStorage) ListCheckResultsByUser(userId happydns.Identifier, limit int) (ret []*happydns.CheckResult, err error) {
	start := time.Now()
	ret, err = s.inner.ListCheckResultsByUser(userId, limit)
	observeStorage("list", "check_result", start, err)
	return
}

func (s *instrumentedStorage) GetCheckResult(checkName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, resultId happydns.Identifier) (ret *happydns.CheckResult, err error) {
	start := time.Now()
	ret, err = s.inner.GetCheckResult(checkName, targetType, targetId, resultId)
	observeStorage("get", "check_result", start, err)
	return
}

func (s *instrumentedStorage) CreateCheckResult(result *happydns.CheckResult) (err error) {
	start := time.Now()
	err = s.inner.CreateCheckResult(result)
	observeStorage("create", "check_result", start, err)
	return
}

func (s *instrumentedStorage) DeleteCheckResult(checkName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, resultId happydns.Identifier) (err error) {
	start := time.Now()
	err = s.inner.DeleteCheckResult(checkName, targetType, targetId, resultId)
	observeStorage("delete", "check_result", start, err)
	return
}

func (s *instrumentedStorage) DeleteOldCheckResults(checkName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, keepCount int) (err error) {
	start := time.Now()
	err = s.inner.DeleteOldCheckResults(checkName, targetType, targetId, keepCount)
	observeStorage("delete", "check_result", start, err)
	return
}

func (s *instrumentedStorage) ListEnabledCheckerSchedules() (ret []*happydns.CheckerSchedule, err error) {
	start := time.Now()
	ret, err = s.inner.ListEnabledCheckerSchedules()
	observeStorage("list", "check_schedule", start, err)
	return
}

func (s *instrumentedStorage) ListCheckerSchedulesByUser(userId happydns.Identifier) (ret []*happydns.CheckerSchedule, err error) {
	start := time.Now()
	ret, err = s.inner.ListCheckerSchedulesByUser(userId)
	observeStorage("list", "check_schedule", start, err)
	return
}

func (s *instrumentedStorage) ListCheckerSchedulesByTarget(targetType happydns.CheckScopeType, targetId happydns.Identifier) (ret []*happydns.CheckerSchedule, err error) {
	start := time.Now()
	ret, err = s.inner.ListCheckerSchedulesByTarget(targetType, targetId)
	observeStorage("list", "check_schedule", start, err)
	return
}

func (s *instrumentedStorage) GetCheckerSchedule(scheduleId happydns.Identifier) (ret *happydns.CheckerSchedule, err error) {
	start := time.Now()
	ret, err = s.inner.GetCheckerSchedule(scheduleId)
	observeStorage("get", "check_schedule", start, err)
	return
}

func (s *instrumentedStorage) CreateCheckerSchedule(schedule *happydns.CheckerSchedule) (err error) {
	start := time.Now()
	err = s.inner.CreateCheckerSchedule(schedule)
	observeStorage("create", "check_schedule", start, err)
	return
}

func (s *instrumentedStorage) UpdateCheckerSchedule(schedule *happydns.CheckerSchedule) (err error) {
	start := time.Now()
	err = s.inner.UpdateCheckerSchedule(schedule)
	observeStorage("update", "check_schedule", start, err)
	return
}

func (s *instrumentedStorage) DeleteCheckerSchedule(scheduleId happydns.Identifier) (err error) {
	start := time.Now()
	err = s.inner.DeleteCheckerSchedule(scheduleId)
	observeStorage("delete", "check_schedule", start, err)
	return
}

func (s *instrumentedStorage) ListActiveCheckExecutions() (ret []*happydns.CheckExecution, err error) {
	start := time.Now()
	ret, err = s.inner.ListActiveCheckExecutions()
	observeStorage("list", "check_execution", start, err)
	return
}

func (s *instrumentedStorage) GetCheckExecution(executionId happydns.Identifier) (ret *happydns.CheckExecution, err error) {
	start := time.Now()
	ret, err = s.inner.GetCheckExecution(executionId)
	observeStorage("get", "check_execution", start, err)
	return
}

func (s *instrumentedStorage) CreateCheckExecution(execution *happydns.CheckExecution) (err error) {
	start := time.Now()
	err = s.inner.CreateCheckExecution(execution)
	observeStorage("create", "check_execution", start, err)
	return
}

func (s *instrumentedStorage) UpdateCheckExecution(execution *happydns.CheckExecution) (err error) {
	start := time.Now()
	err = s.inner.UpdateCheckExecution(execution)
	observeStorage("update", "check_execution", start, err)
	return
}

func (s *instrumentedStorage) DeleteCheckExecution(executionId happydns.Identifier) (err error) {
	start := time.Now()
	err = s.inner.DeleteCheckExecution(executionId)
	observeStorage("delete", "check_execution", start, err)
	return
}

func (s *instrumentedStorage) CheckSchedulerRun() (err error) {
	start := time.Now()
	err = s.inner.CheckSchedulerRun()
	observeStorage("run", "check_schedule", start, err)
	return
}

func (s *instrumentedStorage) LastCheckSchedulerRun() (t *time.Time, err error) {
	start := time.Now()
	t, err = s.inner.LastCheckSchedulerRun()
	observeStorage("get", "check_schedule", start, err)
	return
}

// CheckerConfiguration

func (s *instrumentedStorage) ListAllCheckerConfigurations() (ret happydns.Iterator[happydns.CheckerOptions], err error) {
	start := time.Now()
	ret, err = s.inner.ListAllCheckerConfigurations()
	observeStorage("list", "check_config", start, err)
	return
}

func (s *instrumentedStorage) ListCheckerConfiguration(name string) (ret []*happydns.CheckerOptionsPositional, err error) {
	start := time.Now()
	ret, err = s.inner.ListCheckerConfiguration(name)
	observeStorage("list", "check_config", start, err)
	return
}

func (s *instrumentedStorage) GetCheckerConfiguration(name string, a *happydns.Identifier, b *happydns.Identifier, c *happydns.Identifier) (ret []*happydns.CheckerOptionsPositional, err error) {
	start := time.Now()
	ret, err = s.inner.GetCheckerConfiguration(name, a, b, c)
	observeStorage("get", "check_config", start, err)
	return
}

func (s *instrumentedStorage) UpdateCheckerConfiguration(name string, a *happydns.Identifier, b *happydns.Identifier, c *happydns.Identifier, opts happydns.CheckerOptions) (err error) {
	start := time.Now()
	err = s.inner.UpdateCheckerConfiguration(name, a, b, c, opts)
	observeStorage("update", "check_config", start, err)
	return
}

func (s *instrumentedStorage) DeleteCheckerConfiguration(name string, a *happydns.Identifier, b *happydns.Identifier, c *happydns.Identifier) (err error) {
	start := time.Now()
	err = s.inner.DeleteCheckerConfiguration(name, a, b, c)
	observeStorage("delete", "check_config", start, err)
	return
}

func (s *instrumentedStorage) ClearCheckerConfigurations() (err error) {
	start := time.Now()
	err = s.inner.ClearCheckerConfigurations()
	observeStorage("delete", "check_config", start, err)
	return
}

// User

func (s *instrumentedStorage) ListAllUsers() (ret happydns.Iterator[happydns.User], err error) {
	start := time.Now()
	ret, err = s.inner.ListAllUsers()
	observeStorage("list", "user", start, err)
	return
}

func (s *instrumentedStorage) GetUser(userid happydns.Identifier) (ret *happydns.User, err error) {
	start := time.Now()
	ret, err = s.inner.GetUser(userid)
	observeStorage("get", "user", start, err)
	return
}

func (s *instrumentedStorage) GetUserByEmail(email string) (ret *happydns.User, err error) {
	start := time.Now()
	ret, err = s.inner.GetUserByEmail(email)
	observeStorage("get", "user", start, err)
	return
}

func (s *instrumentedStorage) CreateOrUpdateUser(user *happydns.User) (err error) {
	start := time.Now()
	err = s.inner.CreateOrUpdateUser(user)
	observeStorage("update", "user", start, err)
	return
}

func (s *instrumentedStorage) DeleteUser(userid happydns.Identifier) (err error) {
	start := time.Now()
	err = s.inner.DeleteUser(userid)
	observeStorage("delete", "user", start, err)
	return
}

func (s *instrumentedStorage) ClearUsers() (err error) {
	start := time.Now()
	err = s.inner.ClearUsers()
	observeStorage("delete", "user", start, err)
	return
}

// Zone

func (s *instrumentedStorage) ListAllZones() (ret happydns.Iterator[happydns.ZoneMessage], err error) {
	start := time.Now()
	ret, err = s.inner.ListAllZones()
	observeStorage("list", "zone", start, err)
	return
}

func (s *instrumentedStorage) GetZoneMeta(zoneid happydns.Identifier) (ret *happydns.ZoneMeta, err error) {
	start := time.Now()
	ret, err = s.inner.GetZoneMeta(zoneid)
	observeStorage("get", "zone", start, err)
	return
}

func (s *instrumentedStorage) GetZone(zoneid happydns.Identifier) (ret *happydns.ZoneMessage, err error) {
	start := time.Now()
	ret, err = s.inner.GetZone(zoneid)
	observeStorage("get", "zone", start, err)
	return
}

func (s *instrumentedStorage) CreateZone(zone *happydns.Zone) (err error) {
	start := time.Now()
	err = s.inner.CreateZone(zone)
	observeStorage("create", "zone", start, err)
	return
}

func (s *instrumentedStorage) UpdateZone(zone *happydns.Zone) (err error) {
	start := time.Now()
	err = s.inner.UpdateZone(zone)
	observeStorage("update", "zone", start, err)
	return
}

func (s *instrumentedStorage) DeleteZone(zoneid happydns.Identifier) (err error) {
	start := time.Now()
	err = s.inner.DeleteZone(zoneid)
	observeStorage("delete", "zone", start, err)
	return
}

func (s *instrumentedStorage) ClearZones() (err error) {
	start := time.Now()
	err = s.inner.ClearZones()
	observeStorage("delete", "zone", start, err)
	return
}
