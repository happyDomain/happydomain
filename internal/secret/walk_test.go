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
	"errors"
	"slices"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/model"
)

type walkInner struct {
	Token   happydns.Secret `json:"token"`
	NotMine string          `json:"notmine"`
}

type walkEmbedded struct {
	AppKey happydns.Secret `json:"appkey"`
}

type walkNamedEmbedded struct {
	Key happydns.Secret `json:"key"`
}

type walkUnexportedEmbedded struct {
	Hidden happydns.Secret `json:"hiddenkey"`
}

type walkObject struct {
	walkEmbedded
	walkUnexportedEmbedded
	walkNamedEmbedded `json:"named"`

	ApiKey     happydns.Secret `json:"api_key"`
	NoTag      happydns.Secret
	Skipped    happydns.Secret `json:"-"`
	Dash       happydns.Secret `json:"-,"`
	unexported happydns.Secret
	Inner      walkInner  `json:"inner"`
	Ptr        *walkInner `json:"ptr,omitempty"`
	NilPtr     *walkInner `json:"nilptr,omitempty"`
	Plain      string     `json:"plain"`
}

func collect(t *testing.T, obj any) []string {
	t.Helper()

	var paths []string
	err := Walk(obj, func(path string, s *happydns.Secret) error {
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		t.Fatalf("Walk error = %v", err)
	}
	return paths
}

func TestWalkReportsJSONPaths(t *testing.T) {
	obj := &walkObject{Ptr: &walkInner{}}

	got := collect(t, obj)
	want := []string{"appkey", "hiddenkey", "named.key", "api_key", "NoTag", "-", "inner.token", "ptr.token"}

	if !slices.Equal(got, want) {
		t.Errorf("paths = %v, want %v", got, want)
	}
}

func TestWalkCanMutate(t *testing.T) {
	obj := &walkObject{Ptr: &walkInner{}}

	err := Walk(obj, func(path string, s *happydns.Secret) error {
		*s = happydns.NewSecret("v-" + path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for path, s := range map[string]happydns.Secret{
		"appkey":      obj.AppKey,
		"hiddenkey":   obj.Hidden,
		"named.key":   obj.walkNamedEmbedded.Key,
		"api_key":     obj.ApiKey,
		"inner.token": obj.Inner.Token,
		"ptr.token":   obj.Ptr.Token,
	} {
		if s.Reveal() != "v-"+path {
			t.Errorf("%s = %q, want %q", path, s.Reveal(), "v-"+path)
		}
	}
	if !obj.Skipped.IsEmpty() || !obj.unexported.IsEmpty() {
		t.Error("skipped fields must not be touched")
	}
}

func TestWalkRequiresPointerToStruct(t *testing.T) {
	for name, obj := range map[string]any{
		"nil":            nil,
		"struct value":   walkObject{},
		"nil pointer":    (*walkObject)(nil),
		"pointer to int": new(int),
	} {
		if err := Walk(obj, func(string, *happydns.Secret) error { return nil }); err == nil {
			t.Errorf("%s: Walk succeeded, want an error", name)
		}
	}
}

func TestWalkRefusesSecretsInContainers(t *testing.T) {
	type inMap struct {
		M map[string]happydns.Secret `json:"m"`
	}
	type inSlice struct {
		S []walkInner `json:"s"`
	}
	type inArray struct {
		A [2]happydns.Secret `json:"a"`
	}
	type viaPointer struct {
		P *happydns.Secret `json:"p"`
	}
	type inInterface struct {
		I any `json:"i"`
	}

	for name, obj := range map[string]any{
		"map":              &inMap{},
		"slice":            &inSlice{},
		"array":            &inArray{},
		"pointer":          &viaPointer{},
		"interface struct": &inInterface{I: walkInner{}},
	} {
		err := Walk(obj, func(string, *happydns.Secret) error { return nil })
		if err == nil {
			t.Errorf("%s: Walk succeeded, want an error", name)
		}
	}
}

func TestWalkAllowsContainersWithoutSecrets(t *testing.T) {
	type harmless struct {
		M     map[string]string `json:"m"`
		S     []byte            `json:"s"`
		I     any               `json:"i"`
		Token happydns.Secret   `json:"token"`
	}

	got := collect(t, &harmless{I: "a string"})
	if !slices.Equal(got, []string{"token"}) {
		t.Errorf("paths = %v, want [token]", got)
	}
}

func TestWalkFollowsInterfacePointers(t *testing.T) {
	type holder struct {
		Body any `json:"Provider"`
	}

	inner := &walkInner{}
	got := collect(t, &holder{Body: inner})
	if !slices.Equal(got, []string{"Provider.token"}) {
		t.Errorf("paths = %v, want [Provider.token]", got)
	}
}

func TestWalkCallbackErrorAborts(t *testing.T) {
	boom := errors.New("boom")
	calls := 0

	err := Walk(&walkObject{}, func(path string, s *happydns.Secret) error {
		calls++
		if path == "api_key" {
			return boom
		}
		return nil
	})

	if !errors.Is(err, boom) {
		t.Fatalf("Walk error = %v, want it to wrap the callback error", err)
	}
	if !strings.Contains(err.Error(), "api_key") {
		t.Errorf("Walk error = %q, want it to name the field", err)
	}
	if calls != 4 {
		t.Errorf("callback called %d times, want the walk to stop at api_key (4)", calls)
	}
}

type walkNode struct {
	Name  string          `json:"name"`
	Token happydns.Secret `json:"token"`
	Next  *walkNode       `json:"next,omitempty"`
}

type walkPlainNode struct {
	Name string         `json:"name"`
	Next *walkPlainNode `json:"next,omitempty"`
}

type walkWithPlainCycle struct {
	Token happydns.Secret `json:"token"`
	Ring  *walkPlainNode  `json:"ring"`
}

func TestWalkEndsOnCycleWithoutSecret(t *testing.T) {
	a := &walkPlainNode{Name: "a"}
	b := &walkPlainNode{Name: "b", Next: a}
	a.Next = b

	got := collect(t, &walkWithPlainCycle{Ring: a})
	if !slices.Equal(got, []string{"token"}) {
		t.Errorf("paths = %v, want [token]", got)
	}
}

func TestWalkRefusesCycleThroughSecret(t *testing.T) {
	a := &walkNode{Name: "a"}
	a.Next = &walkNode{Name: "b", Next: a}

	err := Walk(a, func(string, *happydns.Secret) error { return nil })
	if !errors.Is(err, ErrUnsupportedSecret) {
		t.Errorf("Walk on a cycle holding secrets = %v, want ErrUnsupportedSecret", err)
	}
}

func TestWalkRefusesSecretReachedTwice(t *testing.T) {
	type aliased struct {
		A *walkInner `json:"a"`
		B *walkInner `json:"b"`
		I any        `json:"i"`
	}

	shared := &walkInner{}
	for name, obj := range map[string]*aliased{
		"two fields":          {A: shared, B: shared},
		"field and interface": {A: shared, I: shared},
	} {
		// The same secret under two paths would be sealed for one and read
		// back for the other: its associated data cannot match both.
		err := Walk(obj, func(string, *happydns.Secret) error { return nil })
		if !errors.Is(err, ErrUnsupportedSecret) {
			t.Errorf("%s: Walk = %v, want ErrUnsupportedSecret", name, err)
		}
	}
}

func TestWalkAllowsSharedStructWithoutSecret(t *testing.T) {
	type sharing struct {
		A     *walkPlainNode  `json:"a"`
		B     *walkPlainNode  `json:"b"`
		Token happydns.Secret `json:"token"`
	}

	shared := &walkPlainNode{Name: "shared"}
	got := collect(t, &sharing{A: shared, B: shared})
	if !slices.Equal(got, []string{"token"}) {
		t.Errorf("paths = %v, want [token]", got)
	}
}
