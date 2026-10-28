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

package secret

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/internal/vault"
	"git.happydns.org/happyDomain/model"
)

const (
	// KindVaultUser is a personal keystore keeping no secret: each
	// reference names a secret in the KV v2 engine of a Vault of its owner,
	// read with a token of theirs, the secret of its safe.
	KindVaultUser = "vault-user"

	// The parameters creating a keystore of KindVaultUser.
	vaultAddressParam   = "address"
	vaultTokenParam     = "token"
	vaultMountParam     = "mount"
	vaultNamespaceParam = "namespace"

	// vaultRenewRetry is how long a renewal that failed waits before it is
	// tried again.
	vaultRenewRetry = 10 * time.Minute
)

// vaultUserLocation is what a keystore of KindVaultUser keeps in clear: where
// its Vault is.
type vaultUserLocation struct {
	Address   string `json:"address"`
	Mount     string `json:"mount"`
	Namespace string `json:"namespace,omitempty"`
}

// vaultUserStore resolves references through a Vault of their owner, read
// with a token of theirs: the policies of that token tell what the keystore
// reaches. The address is checked as the endpoint of a provider is.
//
// The token is the secret of the safe, see SecretKeeping: its owner chooses
// the keystore it is sealed in, as for any secret. The keyring entry of the
// store only tells where the Vault is.
//
// Renewing only happens at unlock: a token unused for longer than its lease
// still expires, but for a periodic one.
type vaultUserStore struct {
	guard *netguard.Guard

	// http, shared by the keystores of every owner, reaches only the
	// destinations guard allows.
	http *http.Client

	now func() time.Time

	// renewAt tells, for the identifier of each safe unlocked since
	// startup, when its token is to be renewed next.
	mu      sync.Mutex
	renewAt map[string]time.Time
}

func newVaultUserStore(cfg Config) (Store, error) {
	return &vaultUserStore{
		guard:   cfg.Outbound,
		http:    cfg.Outbound.HTTPClient(vaultTimeout),
		now:     time.Now,
		renewAt: map[string]time.Time{},
	}, nil
}

// OpensUnattended: the token is kept. Whether a keystore does depends on
// the keystore its token is sealed in, which the Manager checks.
func (s *vaultUserStore) OpensUnattended() bool { return true }

func (s *vaultUserStore) ReferenceHint() string {
	return "the path of the secret in your Vault, below the mount of the keystore, then # and its key: dns/ovh#app_secret"
}

func (s *vaultUserStore) Reference(payload []byte) (string, error) { return vaultReference(payload) }

func (s *vaultUserStore) CreateForm() []happydns.Field {
	return []happydns.Field{
		{Id: vaultAddressParam, Type: "string", Label: "Vault address", Placeholder: "https://vault.example:8200", Required: true},
		{
			Id:          vaultTokenParam,
			Type:        "string",
			Label:       "Token",
			Description: "A token reading the secrets, best limited to them by its policies, and periodic: happyDomain renews it when it uses it, and a periodic token does not expire for being unused a while.",
			Secret:      true,
			Required:    true,
		},
		{Id: vaultMountParam, Type: "string", Label: "Mount of the KV v2 engine", Placeholder: "secret"},
		{Id: vaultNamespaceParam, Type: "string", Label: "Namespace"},
	}
}

// UpdateForm asks for a new token only: the secrets are referenced in the
// Vault the keystore was created for.
func (s *vaultUserStore) UpdateForm() []happydns.Field {
	return []happydns.Field{{
		Id:          vaultTokenParam,
		Type:        "string",
		Label:       "New token",
		Description: "It replaces the token of the keystore, as when Vault no longer knows it.",
		Secret:      true,
		Required:    true,
	}}
}

// SecretParam: the secret of the safe is the token.
func (s *vaultUserStore) SecretParam() string { return vaultTokenParam }

// Update checks that the Vault of the keystore knows the token params holds,
// which the Manager seals as the new secret of safe. No error repeats the
// token.
func (s *vaultUserStore) Update(ctx context.Context, safe *happydns.Safe, params json.RawMessage) error {
	_, token, err := parseTokenParams(params)
	if err != nil {
		return err
	}
	loc, err := s.entry(safe)
	if err != nil {
		return err
	}
	c, err := s.client(loc)
	if err != nil {
		return err
	}
	if err := s.checkToken(ctx, c, token); err != nil {
		return err
	}

	// The renewal schedule was the old token's: forget it so the new one is
	// renewed on its first use.
	s.mu.Lock()
	delete(s.renewAt, safe.Id.String())
	s.mu.Unlock()
	return nil
}

// client returns the client of the Vault p names.
func (s *vaultUserStore) client(p vaultUserLocation) (*vault.Client, error) {
	c, err := vault.New(p.Address, s.http)
	if err != nil {
		return nil, err
	}
	c.Namespace = p.Namespace
	return c, nil
}

// Create checks that the Vault params names knows its token, which the
// Manager seals as the secret of safe, then keeps where the Vault is in the
// keyring of safe. No error repeats the token.
func (s *vaultUserStore) Create(ctx context.Context, safe *happydns.Safe, params json.RawMessage) error {
	p, token, err := parseTokenParams(params)
	if err != nil {
		return err
	}
	kp := vaultUserLocation{Address: p[vaultAddressParam], Mount: p[vaultMountParam], Namespace: p[vaultNamespaceParam]}
	if kp.Mount == "" {
		kp.Mount = "secret"
	}
	for _, seg := range strings.Split(kp.Mount, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return errors.New("the mount names a path in Vault, such as secret")
		}
	}

	c, err := s.client(kp)
	if err != nil {
		return err
	}
	kp.Address = c.Address()
	if err := s.checkToken(ctx, c, token); err != nil {
		return err
	}
	raw, err := json.Marshal(kp)
	if err != nil {
		return err
	}
	safe.Keyring = []happydns.WrappedKeyset{{KEK: KindVaultUser, Params: raw}}
	return nil
}

// parseTokenParams returns the fields of params and the token among them.
func parseTokenParams(params json.RawMessage) (map[string]string, string, error) {
	var p map[string]string
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, "", errors.New("invalid parameters")
	}
	token := p[vaultTokenParam]
	if token == "" {
		return nil, "", errors.New("the token is required")
	}
	return p, token, nil
}

// checkToken checks that the Vault c reaches is allowed and knows token.
func (s *vaultUserStore) checkToken(ctx context.Context, c *vault.Client, token string) error {
	if _, err := s.guard.ValidateURL(ctx, c.Address()); err != nil {
		return err
	}
	return checkToken(ctx, c, token)
}

// checkToken refuses a token Vault does not know, or that is bound to stop
// working: one with a limited number of uses, each reading consuming one, or
// one expiring that cannot be renewed.
func checkToken(ctx context.Context, c *vault.Client, token string) error {
	info, err := c.LookupSelf(ctx, token)
	if err != nil {
		return err
	}
	if info.NumUses > 0 {
		return errors.New("the token has a limited number of uses: each reading of a secret consumes one")
	}
	if info.TTL > 0 && !info.Renewable {
		return errors.New("the token expires and cannot be renewed: create a renewable one, best periodic")
	}
	return nil
}

// renew renews the token of the safe id through c when it is time to. A
// renewal that fails is not reported: the token may still work, else reading
// a secret tells it.
func (s *vaultUserStore) renew(ctx context.Context, id string, c *vault.Client, token string) {
	now := s.now()
	s.mu.Lock()
	if now.Before(s.renewAt[id]) {
		s.mu.Unlock()
		return
	}
	// Reserved, so that another unlock does not renew it as well; on a
	// failure, the reservation is what delays the next attempt.
	s.renewAt[id] = now.Add(vaultRenewRetry)
	s.mu.Unlock()

	t, err := c.RenewSelf(ctx, token)
	var next time.Time
	switch {
	case errors.Is(err, vault.ErrNotRenewable), err == nil && (t.LeaseSeconds <= 0 || !t.Renewable):
		// Never expiring, or no longer renewable: asking again changes
		// nothing.
		next = time.Unix(1<<62, 0)
	case err != nil:
		return
	default:
		next = now.Add(time.Duration(t.LeaseSeconds) * time.Second * 2 / 3)
	}
	s.mu.Lock()
	s.renewAt[id] = next
	s.mu.Unlock()
}

// entry returns where the Vault of safe is, from its keyring entry.
func (s *vaultUserStore) entry(safe *happydns.Safe) (vaultUserLocation, error) {
	for _, w := range safe.Keyring {
		if w.KEK != KindVaultUser {
			continue
		}
		var p vaultUserLocation
		if err := json.Unmarshal(w.Params, &p); err != nil {
			return p, fmt.Errorf("safe %s has unreadable parameters: %w", safe.Id.String(), err)
		}
		return p, nil
	}
	return vaultUserLocation{}, fmt.Errorf("safe %s does not tell where its Vault is", safe.Id.String())
}

// Unlock fails: the Manager opens the token first, see UnlockWithSecret.
func (s *vaultUserStore) Unlock(_ context.Context, safe *happydns.Safe) (UnlockedSafe, error) {
	return nil, fmt.Errorf("safe %s opens with its token", safe.Id.String())
}

func (s *vaultUserStore) UnlockWithSecret(ctx context.Context, safe *happydns.Safe, secret []byte) (UnlockedSafe, error) {
	if safe.Kind != KindVaultUser {
		return nil, fmt.Errorf("safe %s is of kind %q, not %q", safe.Id.String(), safe.Kind, KindVaultUser)
	}
	if len(secret) == 0 {
		return nil, fmt.Errorf("safe %s has no token", safe.Id.String())
	}
	p, err := s.entry(safe)
	if err != nil {
		return nil, err
	}
	token := string(secret)
	c, err := s.client(p)
	if err != nil {
		return nil, err
	}
	s.renew(ctx, safe.Id.String(), c, token)
	return &vaultUnlocked{read: func(ctx context.Context, path string, version int) (map[string]any, error) {
		return c.ReadKV2(ctx, token, p.Mount, path, version)
	}}, nil
}
