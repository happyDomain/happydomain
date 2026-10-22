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
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.happydns.org/happyDomain/internal/vault"
	"git.happydns.org/happyDomain/model"
)

const (
	// KindVault is a personal keystore keeping no secret: each reference
	// names a secret in the KV v2 engine of the Vault of the instance,
	// below a path of its owner.
	KindVault = "vault"

	// maxVaultReference bounds what an owner may send as a path to Vault.
	maxVaultReference = 1024

	// vaultTimeout bounds each request to Vault.
	vaultTimeout = 10 * time.Second
)

var errInvalidVaultReference = errors.New(`a reference names a secret in Vault, then its key: "dns/ovh#app_secret", or "dns/ovh?version=3#app_secret" for a version`)

// vaultRef is a reference to a key of a version of a KV v2 secret.
type vaultRef struct {
	path string

	// version is 0 for the latest.
	version int

	key string
}

// parseVaultReference reads "<path>[?version=<n>]#<key>". The path is
// relative, without "." nor ".." segments, so that it stays below the path
// it is joined to.
func parseVaultReference(ref string) (vaultRef, error) {
	if len(ref) > maxVaultReference || strings.ContainsRune(ref, 0) {
		return vaultRef{}, errInvalidVaultReference
	}
	i := strings.LastIndexByte(ref, '#')
	if i < 0 {
		return vaultRef{}, errInvalidVaultReference
	}
	r := vaultRef{path: ref[:i], key: ref[i+1:]}
	if p, v, ok := strings.Cut(r.path, "?"); ok {
		n, err := strconv.Atoi(strings.TrimPrefix(v, "version="))
		if !strings.HasPrefix(v, "version=") || err != nil || n < 1 {
			return vaultRef{}, errInvalidVaultReference
		}
		r.path, r.version = p, n
	}
	if r.key == "" || r.path == "" {
		return vaultRef{}, errInvalidVaultReference
	}
	for _, seg := range strings.Split(r.path, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return vaultRef{}, errInvalidVaultReference
		}
	}
	return r, nil
}

// vaultReference returns the reference payload carries, once checked.
func vaultReference(payload []byte) (string, error) {
	if _, err := parseVaultReference(string(payload)); err != nil {
		return "", err
	}
	return string(payload), nil
}

// vaultUnlocked reads the secrets references name through read, which reads
// a version of the secret at a path relative to what the keystore reaches.
type vaultUnlocked struct {
	read func(ctx context.Context, path string, version int) (map[string]any, error)
}

// Seal keeps ref, once sure it names a secret in Vault.
func (u *vaultUnlocked) Seal(ctx context.Context, _ SecretContext, ref []byte) ([]byte, error) {
	if _, err := u.resolve(ctx, string(ref)); err != nil {
		return nil, err
	}
	return bytes.Clone(ref), nil
}

// Open reads the secret payload names.
func (u *vaultUnlocked) Open(ctx context.Context, _ SecretContext, payload []byte) ([]byte, error) {
	return u.resolve(ctx, string(payload))
}

// resolve returns the value of the key ref names. What ref holds may be a
// secret entered by mistake: no error repeats it.
func (u *vaultUnlocked) resolve(ctx context.Context, ref string) ([]byte, error) {
	r, err := parseVaultReference(ref)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, vaultTimeout)
	defer cancel()

	data, err := u.read(ctx, r.path, r.version)
	if err != nil {
		return nil, err
	}
	v, ok := data[r.key]
	if !ok {
		return nil, errors.New("the secret in Vault has no such key")
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return nil, errors.New("the key of the secret in Vault does not hold a string")
	}
	return []byte(s), nil
}

// appRoleAuth keeps the token an AppRole logs in to, renewing it before it
// expires, and logging in again once it no longer is valid.
type appRoleAuth struct {
	client       *vault.Client
	mount        string
	roleID       string
	secretID     string
	now          func() time.Time
	mu           sync.Mutex
	token        vault.Token
	renewAt      time.Time
	expires      time.Time
	neverExpires bool
}

func (a *appRoleAuth) set(t vault.Token) {
	a.token = t
	now := a.now()
	// Vault gives a lease of 0 to a token without a TTL.
	a.neverExpires = t.LeaseSeconds <= 0
	lease := time.Duration(t.LeaseSeconds) * time.Second
	a.expires = now.Add(lease)
	// Renewing at two thirds leaves a third of the lease to retry.
	a.renewAt = now.Add(lease * 2 / 3)
}

// get returns a token, logging in or renewing it when needed.
func (a *appRoleAuth) get(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := a.now()
	if a.token.Token != "" {
		if a.neverExpires || now.Before(a.renewAt) {
			return a.token.Token, nil
		}
		if a.token.Renewable && now.Before(a.expires) {
			// A renewal that fails falls through to a new login.
			if t, err := a.client.RenewSelf(ctx, a.token.Token); err == nil {
				a.set(t)
				return t.Token, nil
			}
		}
	}

	t, err := a.client.LoginAppRole(ctx, a.mount, a.roleID, a.secretID)
	if err != nil {
		return "", fmt.Errorf("unable to log in to Vault: %w", err)
	}
	a.set(t)
	return t.Token, nil
}

// drop forgets token, when it is still the one kept: the next get logs in.
func (a *appRoleAuth) drop(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.token.Token == token {
		a.token = vault.Token{}
	}
}

// read reads a version of the secret at path. Vault answers 403 both to a
// token it no longer knows and to a path its policy denies: lookup-self tells
// them apart, and only the first is worth one new login.
func (a *appRoleAuth) read(ctx context.Context, mount, path string, version int) (map[string]any, error) {
	token, err := a.get(ctx)
	if err != nil {
		return nil, err
	}
	data, err := a.client.ReadKV2(ctx, token, mount, path, version)
	if !errors.Is(err, vault.ErrForbidden) {
		return data, err
	}
	if _, lerr := a.client.LookupSelf(ctx, token); lerr == nil {
		return data, err
	}

	a.drop(token)
	if token, err = a.get(ctx); err != nil {
		return nil, err
	}
	return a.client.ReadKV2(ctx, token, mount, path, version)
}

// vaultConfig is how the Vault of the instance is reached.
type vaultConfig struct {
	addr         string
	namespace    string
	caFile       string
	mount        string
	prefix       string
	appRoleMount string
	roleID       string
	secretIDFile string
}

// instanceVault is the Vault of the instance, as configured by flags.
var instanceVault vaultConfig

func init() {
	flag.StringVar(&instanceVault.addr, "secret-vault-addr", "", "Address of the Vault (or OpenBao) server the \"vault\" secret store reads credentials from, such as https://vault.example:8200")
	flag.StringVar(&instanceVault.namespace, "secret-vault-namespace", "", "Vault namespace of -secret-vault-addr, if any")
	flag.StringVar(&instanceVault.caFile, "secret-vault-ca-file", "", "PEM file of the certificate authorities trusted for -secret-vault-addr, besides those of the system")
	flag.StringVar(&instanceVault.mount, "secret-vault-mount", "secret", "Mount of the KV v2 engine the \"vault\" secret store reads")
	flag.StringVar(&instanceVault.prefix, "secret-vault-prefix", "happydomain", "Path, in -secret-vault-mount, holding a path per user named after their identifier, below which they keep their credentials")
	flag.StringVar(&instanceVault.appRoleMount, "secret-vault-approle-mount", "approle", "Mount of the AppRole auth method HappyDomain logs in to Vault with")
	flag.StringVar(&instanceVault.roleID, "secret-vault-role-id", "", "Role ID of the AppRole HappyDomain logs in to Vault with")
	flag.StringVar(&instanceVault.secretIDFile, "secret-vault-secret-id-file", "", "File holding the secret ID of the AppRole HappyDomain logs in to Vault with")
}

// vaultStore resolves references through the Vault of the instance, logged
// in to with an AppRole: each owner reaches the secrets below their path,
// named after their identifier, in the prefix of the instance.
type vaultStore struct {
	mount  string
	prefix string
	auth   *appRoleAuth
}

func newVaultStore(Config) (Store, error) { return newVaultStoreWith(instanceVault) }

// newVaultStoreWith returns the store reading the Vault c configures.
func newVaultStoreWith(c vaultConfig) (Store, error) {
	if c.addr == "" {
		return nil, fmt.Errorf("%w: no Vault server, set -secret-vault-addr", ErrStoreUnavailable)
	}
	if c.roleID == "" || c.secretIDFile == "" {
		return nil, errors.New("the Vault of the instance needs -secret-vault-role-id and -secret-vault-secret-id-file")
	}
	sid, err := os.ReadFile(c.secretIDFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read the secret ID of the AppRole: %w", err)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if c.caFile != "" {
		pem, err := os.ReadFile(c.caFile)
		if err != nil {
			return nil, fmt.Errorf("unable to read the certificate authorities of Vault: %w", err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no certificate in %s", c.caFile)
		}
		transport.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	}
	client, err := vault.New(c.addr, &http.Client{Timeout: vaultTimeout, Transport: transport})
	if err != nil {
		return nil, err
	}
	client.Namespace = c.namespace

	return &vaultStore{
		mount:  c.mount,
		prefix: strings.Trim(c.prefix, "/"),
		auth: &appRoleAuth{
			client:   client,
			mount:    c.appRoleMount,
			roleID:   c.roleID,
			secretID: strings.TrimSpace(string(sid)),
			now:      time.Now,
		},
	}, nil
}

// CheckStartup refuses credentials Vault refuses. A Vault that cannot be
// reached does not keep happyDomain from starting: the credentials it holds
// fail until it is back.
func (s *vaultStore) CheckStartup(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, vaultTimeout)
	defer cancel()
	_, err := s.auth.get(ctx)
	if errors.Is(err, vault.ErrForbidden) {
		return err
	}
	if err != nil {
		log.Printf("secret: the Vault of the instance cannot be reached at startup, its credentials fail until it is: %s", err)
	}
	return nil
}

// OpensUnattended: the instance logs in to Vault itself.
func (s *vaultStore) OpensUnattended() bool { return true }

// ownerPath is the path below which owner keeps their secrets.
func (s *vaultStore) ownerPath(owner happydns.Identifier) string {
	if s.prefix == "" {
		return owner.String()
	}
	return s.prefix + "/" + owner.String()
}

func (s *vaultStore) ReferenceHint() string {
	path := s.mount + "/"
	if s.prefix != "" {
		path += s.prefix + "/"
	}
	return fmt.Sprintf("the path of the secret in Vault below %s<your identifier>/, then # and its key: dns/ovh#app_secret", path)
}

func (s *vaultStore) Reference(payload []byte) (string, error) { return vaultReference(payload) }

func (s *vaultStore) CreateForm() []happydns.Field { return []happydns.Field{} }

// Create needs nothing: the path of the owner in the Vault of the instance
// follows from their identifier.
func (s *vaultStore) Create(context.Context, *happydns.Safe, json.RawMessage) error { return nil }

func (s *vaultStore) Unlock(_ context.Context, safe *happydns.Safe) (UnlockedSafe, error) {
	if safe.Kind != KindVault {
		return nil, fmt.Errorf("safe %s is of kind %q, not %q", safe.Id.String(), safe.Kind, KindVault)
	}
	base := s.ownerPath(safe.Owner)
	return &vaultUnlocked{read: func(ctx context.Context, path string, version int) (map[string]any, error) {
		return s.auth.read(ctx, s.mount, base+"/"+path, version)
	}}, nil
}
