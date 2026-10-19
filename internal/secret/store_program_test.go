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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"git.happydns.org/happyDomain/model"
)

// programFixture is a program store over a directory of its own, with the
// directory of owner, ready to hold their programs.
type programFixture struct {
	dir   string
	owner happydns.Identifier
	store *programStore
}

func newProgramFixture(t *testing.T) programFixture {
	t.Helper()
	f := programFixture{dir: t.TempDir(), owner: happydns.Identifier{0x01, 0x02}}
	if err := os.Mkdir(f.userDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := newProgramStoreIn(f.dir)
	if err != nil {
		t.Fatal(err)
	}
	f.store = store.(*programStore)
	return f
}

func (f programFixture) userDir() string {
	return filepath.Join(f.dir, f.owner.String())
}

// script writes a shell script of the owner of f.
func (f programFixture) script(t *testing.T, name, body string) {
	t.Helper()
	path := filepath.Join(f.userDir(), name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
}

func programParams(program string) json.RawMessage {
	b, _ := json.Marshal(map[string]string{programParam: program})
	return b
}

func (f programFixture) unlocked(t *testing.T, program string) UnlockedSafe {
	t.Helper()
	safe := &happydns.Safe{Id: happydns.Identifier{0x05}, Owner: f.owner, Kind: KindProgram}
	if err := f.store.Create(context.Background(), safe, programParams(program)); err != nil {
		t.Fatalf("Create(%q): %v", program, err)
	}
	u, err := f.store.Unlock(context.Background(), safe)
	if err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	return u
}

func TestProgramStoreNeedsItsDirectory(t *testing.T) {
	if _, err := newProgramStoreIn(""); !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("newProgramStoreIn(no directory) = %v, want ErrStoreUnavailable", err)
	}
	store, err := newProgramStoreIn(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.(StartupChecker).CheckStartup(context.Background()); err == nil {
		t.Error("CheckStartup accepted a missing directory")
	}
	if err := newProgramFixture(t).store.CheckStartup(context.Background()); err != nil {
		t.Errorf("CheckStartup = %v", err)
	}

	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if store, err := newProgramStoreIn(file); err != nil {
		t.Fatal(err)
	} else if err := store.(StartupChecker).CheckStartup(context.Background()); err == nil {
		t.Error("CheckStartup accepted a file")
	}
}

func TestProgramStoreSealsReferencesAndRunsTheProgram(t *testing.T) {
	ctx := context.Background()
	f := newProgramFixture(t)
	f.script(t, "get", `printf 's3cr3t-%s\n' "$1"`)
	u := f.unlocked(t, "get")
	sc := objectContext()

	payload, err := u.Seal(ctx, sc, []byte("ovh/key"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if strings.Contains(string(payload), "s3cr3t") {
		t.Error("the payload holds the secret")
	}
	if ref, err := f.store.Reference(payload); err != nil || ref != "ovh/key" {
		t.Errorf("Reference = %q, %v", ref, err)
	}
	if got, err := u.Open(ctx, sc, payload); err != nil || string(got) != "s3cr3t-ovh/key" {
		t.Errorf("Open = %q, %v; want the output without its final newline", got, err)
	}
}

// The reference is one argument, never seen by a shell.
func TestProgramStoreRunsNoShell(t *testing.T) {
	f := newProgramFixture(t)
	f.script(t, "get", `printf '%s|%s' "$#" "$1"`)
	u := f.unlocked(t, "get")

	ref := `a b; touch pwned $(id)`
	got, err := u.Open(context.Background(), objectContext(), []byte(ref))
	if err != nil || string(got) != "1|"+ref {
		t.Errorf("Open = %q, %v; want the reference as a single argument", got, err)
	}
	if _, err := os.Stat(filepath.Join(f.userDir(), "pwned")); err == nil {
		t.Error("the reference went through a shell")
	}
}

// The program gets where the secret belongs, and none of the environment of
// happyDomain, which may hold its own secrets.
func TestProgramStoreEnvironment(t *testing.T) {
	t.Setenv("SOME_SERVER_SECRET", "leaked")
	f := newProgramFixture(t)
	f.script(t, "env", `printf '%s %s %s %s %s' "$HAPPYDOMAIN_USER" "$HAPPYDOMAIN_OBJECT_TYPE" "$HAPPYDOMAIN_OBJECT_ID" "$HAPPYDOMAIN_FIELD" "${SOME_SERVER_SECRET:-none}"`)
	u := f.unlocked(t, "env")
	sc := objectContext()
	sc.Owner = f.owner
	sc.Field = "ApiKey"

	got, err := u.Open(context.Background(), sc, []byte("ref"))
	want := strings.Join([]string{f.owner.String(), sc.ObjectType, sc.ObjectId, "ApiKey", "none"}, " ")
	if err != nil || string(got) != want {
		t.Errorf("Open = %q, %v; want %q", got, err, want)
	}
}

func TestProgramStoreFailures(t *testing.T) {
	ctx := context.Background()
	f := newProgramFixture(t)
	f.script(t, "fail", `echo "no secret for $1" >&2; exit 3`)
	f.script(t, "empty", `echo`)
	f.script(t, "big", `head -c 70000 /dev/zero | tr '\0' x`)
	f.script(t, "slow", `sleep 30`)
	f.script(t, "get", `printf 'v'`)

	for name, program := range map[string]string{"failing": "fail", "empty output": "empty", "too much output": "big"} {
		u := f.unlocked(t, program)
		_, err := u.Seal(ctx, objectContext(), []byte("pa55w0rd-typed-in"))
		if err == nil {
			t.Errorf("Seal(%s) succeeded", name)
			continue
		}
		// What is entered as a reference may be a secret, typed in by
		// mistake: no error repeats it, nor what the program says of it.
		if strings.Contains(err.Error(), "pa55w0rd") {
			t.Errorf("%s: the error repeats the reference: %v", name, err)
		}
	}

	f.store.timeout = 200 * time.Millisecond
	start := time.Now()
	if _, err := f.unlocked(t, "slow").Open(ctx, objectContext(), []byte("ref")); err == nil || !strings.Contains(err.Error(), "did not answer") {
		t.Errorf("Open(slow) = %v, want a timeout", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Error("the program was not stopped at its timeout")
	}

	u := f.unlocked(t, "get")
	for name, ref := range map[string]string{"empty": "", "nul": "a\x00b", "too long": strings.Repeat("x", maxProgramReference+1)} {
		if _, err := u.Seal(ctx, objectContext(), []byte(ref)); err == nil {
			t.Errorf("Seal(%s reference) succeeded", name)
		}
	}
}

// A keystore runs a program of the directory of its owner, never one
// elsewhere: not of another user, nor of the system.
func TestProgramStoreStaysInItsDirectory(t *testing.T) {
	ctx := context.Background()
	f := newProgramFixture(t)
	other := filepath.Join(f.dir, "other-user")
	if err := os.Mkdir(other, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "theirs"), []byte("#!/bin/sh\necho theirs\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(other, "theirs"), filepath.Join(f.userDir(), "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.userDir(), "noexec"), []byte("#!/bin/sh\necho x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(f.userDir(), "dir"), 0o700); err != nil {
		t.Fatal(err)
	}

	for name, program := range map[string]string{
		"none":             "",
		"missing":          "missing",
		"parent":           "../other-user/theirs",
		"absolute":         "/bin/sh",
		"escaping symlink": "link",
		"not executable":   "noexec",
		"directory":        "dir",
	} {
		safe := &happydns.Safe{Id: happydns.Identifier{0x05}, Owner: f.owner, Kind: KindProgram}
		if err := f.store.Create(ctx, safe, programParams(program)); err == nil {
			t.Errorf("Create(%s) succeeded", name)
		}
	}

	// Replaced since, by a link to elsewhere: it no longer runs.
	f.script(t, "get", `printf 'v'`)
	u := f.unlocked(t, "get")
	if err := os.Remove(filepath.Join(f.userDir(), "get")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(other, "theirs"), filepath.Join(f.userDir(), "get")); err != nil {
		t.Fatal(err)
	}
	if got, err := u.Open(ctx, objectContext(), []byte("ref")); err == nil {
		t.Errorf("Open through a link to elsewhere = %q", got)
	}

	stranger := &happydns.Safe{Id: happydns.Identifier{0x06}, Owner: happydns.Identifier{0x09}, Kind: KindProgram}
	err := f.store.Create(ctx, stranger, programParams("get"))
	if err == nil || !strings.Contains(err.Error(), stranger.Owner.String()) || strings.Contains(err.Error(), f.dir) {
		t.Errorf("Create(no directory) = %v, want it naming the directory to create by the identifier of its owner, not where the server keeps it", err)
	}
}

// A program stopped at its timeout takes with it what it started: no
// process is left behind for each credential used.
func TestProgramStoreStopsWhatTheProgramStarted(t *testing.T) {
	f := newProgramFixture(t)
	pidFile := filepath.Join(t.TempDir(), "pid")
	f.script(t, "slow", `sleep 30 & echo $! > `+pidFile+`; wait`)
	f.store.timeout = 500 * time.Millisecond

	if _, err := f.unlocked(t, "slow").Open(context.Background(), objectContext(), []byte("ref")); err == nil {
		t.Fatal("a program running past its timeout succeeded")
	}
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for syscall.Kill(pid, 0) == nil {
		if time.Now().After(deadline) {
			syscall.Kill(pid, syscall.SIGKILL)
			t.Fatal("the child of the program outlived its timeout")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// A program answering, then exiting with success, answers the secret even
// though a process it started keeps its output open, as an agent may.
func TestProgramStoreAnswersDespiteAChildKeepingItsOutput(t *testing.T) {
	f := newProgramFixture(t)
	pidFile := filepath.Join(t.TempDir(), "pid")
	f.script(t, "get", `sleep 30 & echo $! > `+pidFile+`; echo v`)
	t.Cleanup(func() {
		if raw, err := os.ReadFile(pidFile); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil {
				syscall.Kill(pid, syscall.SIGKILL)
			}
		}
	})

	got, err := f.unlocked(t, "get").Open(context.Background(), objectContext(), []byte("ref"))
	if err != nil || string(got) != "v" {
		t.Errorf("Open = %q, %v; want the answer of the program", got, err)
	}
}

// A program that fails is not rescued by a child keeping its output open.
func TestProgramStoreFailsDespiteAChildKeepingItsOutput(t *testing.T) {
	f := newProgramFixture(t)
	pidFile := filepath.Join(t.TempDir(), "pid")
	f.script(t, "get", `sleep 30 & echo $! > `+pidFile+`; echo partial; exit 1`)
	t.Cleanup(func() {
		if raw, err := os.ReadFile(pidFile); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil {
				syscall.Kill(pid, syscall.SIGKILL)
			}
		}
	})

	if got, err := f.unlocked(t, "get").Open(context.Background(), objectContext(), []byte("ref")); err == nil {
		t.Errorf("Open = %q, want the failure of the program", got)
	}
}

// Only the final line ending goes: a secret may end with whitespace.
func TestProgramStoreTrimsOnlyTheFinalNewline(t *testing.T) {
	f := newProgramFixture(t)
	for want, body := range map[string]string{
		"v":    `printf 'v\r\n'`,
		"v\n":  `printf 'v\n\n'`,
		"v ":   `printf 'v '`,
		"a\nb": `printf 'a\nb\n'`,
	} {
		f.script(t, "get", body)
		got, err := f.unlocked(t, "get").Open(context.Background(), objectContext(), []byte("ref"))
		if err != nil || string(got) != want {
			t.Errorf("Open(%s) = %q, %v; want %q", body, got, err, want)
		}
	}
}

// Parameters come from the database: a safe naming a program outside the
// directory of its owner, or none, does not open, whatever Create checked.
func TestProgramStoreRefusesBrokenParameters(t *testing.T) {
	f := newProgramFixture(t)
	for name, keyring := range map[string][]happydns.WrappedKeyset{
		"unreadable": {{KEK: KindProgram, Params: json.RawMessage(`{`)}},
		"no program": {{KEK: "other", Params: programParams("get")}},
		"empty":      {{KEK: KindProgram, Params: programParams("")}},
		"parent":     {{KEK: KindProgram, Params: programParams("../other-user/theirs")}},
		"absolute":   {{KEK: KindProgram, Params: programParams("/bin/sh")}},
	} {
		safe := &happydns.Safe{Id: happydns.Identifier{0x05}, Owner: f.owner, Kind: KindProgram, Keyring: keyring}
		if _, err := f.store.Unlock(context.Background(), safe); err == nil {
			t.Errorf("Unlock(%s) succeeded", name)
		}
	}
}

// A symbolic link staying in the directory of its owner runs, and so does a
// directory of its owner that is itself a link, as an administrator may make it.
func TestProgramStoreFollowsLinksStayingInItsDirectory(t *testing.T) {
	ctx := context.Background()
	f := newProgramFixture(t)
	f.script(t, "real", `printf 'v'`)
	if err := os.Mkdir(filepath.Join(f.userDir(), "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../real", filepath.Join(f.userDir(), "bin", "get")); err != nil {
		t.Fatal(err)
	}
	if got, err := f.unlocked(t, "bin/get").Open(ctx, objectContext(), []byte("ref")); err != nil || string(got) != "v" {
		t.Errorf("Open through a link staying in the directory = %q, %v", got, err)
	}

	elsewhere := t.TempDir()
	if err := os.WriteFile(filepath.Join(elsewhere, "get"), []byte("#!/bin/sh\nprintf w\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	g := programFixture{dir: f.dir, owner: happydns.Identifier{0x03}, store: f.store}
	if err := os.Symlink(elsewhere, g.userDir()); err != nil {
		t.Fatal(err)
	}
	if got, err := g.unlocked(t, "get").Open(ctx, objectContext(), []byte("ref")); err != nil || string(got) != "w" {
		t.Errorf("Open in a directory that is a link = %q, %v", got, err)
	}
}

func TestProgramStoreRefusesASafeOfAnotherKind(t *testing.T) {
	f := newProgramFixture(t)
	f.script(t, "get", `printf 'v'`)
	safe := &happydns.Safe{Id: happydns.Identifier{0x05}, Owner: f.owner, Kind: KindProgram}
	if err := f.store.Create(context.Background(), safe, programParams("get")); err != nil {
		t.Fatal(err)
	}
	safe.Kind = KindFile
	if _, err := f.store.Unlock(context.Background(), safe); err == nil {
		t.Error("a safe of another kind opened")
	}
}

func TestProgramKindIsBuiltin(t *testing.T) {
	if _, ok := builtinStores[KindProgram]; !ok {
		t.Fatalf("no builtin store %q", KindProgram)
	}
	store, err := newProgramStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := store.(Creator); !ok {
		t.Error("program is not a Creator")
	}
	if r, ok := store.(Referencing); !ok || r.ReferenceHint() == "" {
		t.Error("program is not Referencing, with a hint")
	}
	if _, ok := store.(StartupChecker); !ok {
		t.Error("program does not check its directory at startup")
	}
	if _, ok := store.(Interactive); ok {
		t.Error("program is Interactive")
	}
	if !opensUnattended(store) {
		t.Error("program does not open unattended")
	}
}

func TestProgramKeystoreThroughManager(t *testing.T) {
	ctx := context.Background()
	f := newProgramFixture(t)
	f.script(t, "get", `printf 's3cr3t-%s\n' "$1"`)
	sc := objectContext()
	sc.Owner = f.owner
	m, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: testInstanceKey(t), Safes: newMemSafeStorage(), Defaults: newMemDefaults()}.WithStores(map[string]StoreFactory{
		KindProgram: func(Config) (Store, error) { return newProgramStoreIn(f.dir) },
	}))
	if err != nil {
		t.Fatal(err)
	}

	ks, err := m.CreateKeystore(ctx, sc.Owner, KindProgram, "My program", programParams("get"))
	if err != nil {
		t.Fatal(err)
	}
	if !ks.References {
		t.Errorf("keystore = %+v, want it keeping references", ks)
	}

	obj := &managedObject{ApiKey: happydns.NewSecret("ovh")}
	if err := m.SealObjectInto(ctx, sc, obj, Destination{Keystore: ks.Ref}); err != nil {
		t.Fatalf("SealObjectInto: %v", err)
	}
	shown := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}
	if err := m.ShowReferences(ctx, sc, shown); err != nil || !shown.ApiKey.IsReference() {
		t.Fatalf("ShowReferences = %v, reference shown: %v", err, shown.ApiKey.IsReference())
	}
	back := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}
	if err := m.OpenObject(ctx, sc, back); err != nil || back.ApiKey.Reveal() != "s3cr3t-ovh" {
		t.Errorf("OpenObject = %q, %v", back.ApiKey.Reveal(), err)
	}
}
