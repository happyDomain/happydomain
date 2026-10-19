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
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"git.happydns.org/happyDomain/model"
)

const (
	// KindProgram is a personal keystore keeping no secret: each reference
	// is given to a program of its owner, on the server, which answers the
	// secret.
	KindProgram = "program"

	// programParam names, in the parameters creating a keystore of
	// KindProgram, its program, relative to the directory of its owner.
	programParam = "program"

	// maxProgramReference is the length of the longest reference.
	maxProgramReference = 1024

	// maxProgramOutput is the size of the largest secret a program answers.
	maxProgramOutput = 64 * 1024

	// programPath is the PATH programs run with: none of the environment of
	// happyDomain is given to them.
	programPath = "/usr/local/bin:/usr/bin:/bin"
)

// programDir holds a directory per owner, named after their identifier, with
// the programs their keystores of KindProgram run.
var programDir string

func init() {
	flag.StringVar(&programDir, "secret-program-dir", "", "Directory holding, for the \"program\" secret store, a directory per user named after their identifier, with the programs answering their credentials; these programs run with the rights of happyDomain: enable this store only on an instance whose users are trusted as its administrator")
}

// programStore resolves references through programs of their owner: the
// program is run with the reference as its only argument, without a shell,
// and its output, without its final newline, is the secret.
//
// The program runs with the rights of happyDomain, which reads the instance
// keyset and the database: whoever writes the programs of an owner holds
// these rights. A keystore only runs programs of its owner's directory,
// following symbolic links only while they stay inside it.
type programStore struct {
	dir string

	// timeout bounds how long a program runs.
	timeout time.Duration
}

func newProgramStore(Config) (Store, error) { return newProgramStoreIn(programDir) }

// newProgramStoreIn returns the program store over dir.
func newProgramStoreIn(dir string) (Store, error) {
	if dir == "" {
		return nil, fmt.Errorf("%w: no directory for the programs of the keystores, set -secret-program-dir", ErrStoreUnavailable)
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return &programStore{dir: dir, timeout: 10 * time.Second}, nil
}

// CheckStartup refuses a directory that is not one.
func (s *programStore) CheckStartup(context.Context) error {
	fi, err := os.Stat(s.dir)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("%s is not a directory", s.dir)
	}
	return nil
}

// OpensUnattended: the program is run whenever a secret is used.
func (s *programStore) OpensUnattended() bool { return true }

func (s *programStore) ReferenceHint() string {
	return "the argument the program of the keystore answers the secret for"
}

func (s *programStore) Reference(payload []byte) (string, error) {
	ref := string(payload)
	if err := checkProgramReference(ref); err != nil {
		return "", err
	}
	return ref, nil
}

func checkProgramReference(ref string) error {
	if ref == "" || len(ref) > maxProgramReference || strings.ContainsRune(ref, 0) {
		return fmt.Errorf("a reference is the argument given to the program of the keystore, of at most %d bytes", maxProgramReference)
	}
	return nil
}

func (s *programStore) ownerDir(owner happydns.Identifier) string {
	return filepath.Join(s.dir, owner.String())
}

func (s *programStore) CreateForm() []happydns.Field {
	return []happydns.Field{{
		Id:          programParam,
		Type:        "string",
		Label:       "Program",
		Description: "The program answering the secrets, relative to your own directory on the server, which the administrator creates. It is run with the reference as its only argument, and its output is the secret.",
		Required:    true,
	}}
}

// Create points safe to the program params names, once sure it can run.
func (s *programStore) Create(_ context.Context, safe *happydns.Safe, params json.RawMessage) error {
	var p map[string]string
	if err := json.Unmarshal(params, &p); err != nil {
		return err
	}
	program := p[programParam]
	if program == "" {
		return errors.New("the program is required")
	}
	if _, err := os.Stat(s.ownerDir(safe.Owner)); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("your directory does not exist: ask the administrator to create it, named after your identifier %s", safe.Owner.String())
	}
	if _, err := s.executable(safe.Owner, program); err != nil {
		return err
	}

	raw, err := json.Marshal(map[string]string{programParam: program})
	if err != nil {
		return err
	}
	safe.Keyring = []happydns.WrappedKeyset{{KEK: KindProgram, Params: raw}}
	return nil
}

// executable returns the path of program, an executable file in the
// directory of owner, through symbolic links staying in it.
func (s *programStore) executable(owner happydns.Identifier, program string) (string, error) {
	if !validFileReference(program) {
		return "", fmt.Errorf("the program %q is not relative to your directory", program)
	}
	dir, err := filepath.EvalSymlinks(s.ownerDir(owner))
	if err != nil {
		return "", errors.New("your directory cannot be opened")
	}
	path, err := filepath.EvalSymlinks(filepath.Join(dir, program))
	if err != nil {
		return "", fmt.Errorf("the program %q cannot be found in your directory", program)
	}
	if rel, err := filepath.Rel(dir, path); err != nil || !filepath.IsLocal(rel) {
		return "", fmt.Errorf("the program %q is outside your directory", program)
	}
	fi, err := os.Stat(path)
	if err != nil || !fi.Mode().IsRegular() || fi.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("the program %q is not an executable file", program)
	}
	return path, nil
}

func (s *programStore) Unlock(_ context.Context, safe *happydns.Safe) (UnlockedSafe, error) {
	if safe.Kind != KindProgram {
		return nil, fmt.Errorf("safe %s is of kind %q, not %q", safe.Id.String(), safe.Kind, KindProgram)
	}
	for _, e := range safe.Keyring {
		if e.KEK != KindProgram {
			continue
		}
		var p map[string]string
		if err := json.Unmarshal(e.Params, &p); err != nil {
			return nil, fmt.Errorf("safe %s has unreadable parameters: %w", safe.Id.String(), err)
		}
		// Params come from the database, not through Create: checked again.
		if !validFileReference(p[programParam]) {
			return nil, fmt.Errorf("safe %s names a program outside the directory of its owner", safe.Id.String())
		}
		return &programUnlocked{store: s, owner: safe.Owner, program: p[programParam]}, nil
	}
	return nil, fmt.Errorf("safe %s has no program", safe.Id.String())
}

// programUnlocked runs the program of one keystore.
type programUnlocked struct {
	store   *programStore
	owner   happydns.Identifier
	program string
}

// Seal keeps ref, once sure the program answers a secret for it.
func (u *programUnlocked) Seal(ctx context.Context, sc SecretContext, ref []byte) ([]byte, error) {
	if _, err := u.run(ctx, sc, string(ref)); err != nil {
		return nil, err
	}
	return bytes.Clone(ref), nil
}

// Open runs the program for the reference payload holds.
func (u *programUnlocked) Open(ctx context.Context, sc SecretContext, payload []byte) ([]byte, error) {
	return u.run(ctx, sc, string(payload))
}

// run returns what the program answers for ref, without its final newline.
// What ref holds may be a secret entered by mistake: no error repeats it, nor
// what the program writes on its standard error, which is discarded.
func (u *programUnlocked) run(ctx context.Context, sc SecretContext, ref string) ([]byte, error) {
	if err := checkProgramReference(ref); err != nil {
		return nil, err
	}
	// Looked up each time: the program may have changed since.
	path, err := u.store.executable(u.owner, u.program)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, u.store.timeout)
	defer cancel()

	out := &cappedBuffer{max: maxProgramOutput}
	cmd := exec.CommandContext(ctx, path, ref)
	cmd.Dir = u.store.ownerDir(u.owner)
	cmd.Env = []string{
		"PATH=" + programPath,
		"HAPPYDOMAIN_USER=" + u.owner.String(),
		"HAPPYDOMAIN_OBJECT_TYPE=" + sc.ObjectType,
		"HAPPYDOMAIN_OBJECT_ID=" + sc.ObjectId,
		"HAPPYDOMAIN_FIELD=" + sc.Field,
	}
	cmd.Stdout = out
	// In a process group of its own, so that what the program started is
	// stopped with it at the timeout.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	// A child keeping the output open does not hold the request past the
	// timeout.
	cmd.WaitDelay = time.Second

	err = cmd.Run()
	if errors.Is(err, exec.ErrWaitDelay) && cmd.ProcessState != nil && cmd.ProcessState.Success() {
		// The program answered and exited: a process it started, such as
		// an agent, still holds its output, which is complete.
		err = nil
	}
	// Killed at the timeout, the program reports an ExitError: the deadline
	// is checked first to say why.
	switch {
	case ctx.Err() == context.DeadlineExceeded:
		return nil, fmt.Errorf("the program of the keystore did not answer within %s", u.store.timeout)
	case out.overflow:
		return nil, fmt.Errorf("the program of the keystore answered more than %d bytes", maxProgramOutput)
	case err != nil:
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return nil, fmt.Errorf("the program of the keystore failed, with status %d", exit.ExitCode())
		}
		return nil, errors.New("the program of the keystore cannot be run")
	}

	// A program may end its line with CRLF, as some password managers do.
	secret := bytes.TrimSuffix(out.Bytes(), []byte("\n"))
	secret = bytes.TrimSuffix(secret, []byte("\r"))
	if len(secret) == 0 {
		return nil, errors.New("the program of the keystore answered no secret")
	}
	return secret, nil
}

// cappedBuffer keeps what is written to it, up to max bytes, and fails past
// them. Its buffer is not embedded: its ReadFrom would be used by io.Copy in
// place of Write, unbounded.
type cappedBuffer struct {
	buf      bytes.Buffer
	max      int
	overflow bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if b.buf.Len()+len(p) > b.max {
		b.overflow = true
		return 0, errors.New("output too large")
	}
	return b.buf.Write(p)
}

func (b *cappedBuffer) Bytes() []byte { return b.buf.Bytes() }
