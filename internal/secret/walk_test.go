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
	type inIntMap struct {
		M map[int]happydns.Secret `json:"m"`
	}
	type inPointerMap struct {
		M map[string]*happydns.Secret `json:"m"`
	}
	type inStructMap struct {
		M map[string]walkInner `json:"m"`
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
		"map with int keys":      &inIntMap{M: map[int]happydns.Secret{1: {}}},
		"map of pointers":        &inPointerMap{},
		"map of structs holding": &inStructMap{},
		"slice":                  &inSlice{},
		"array":                  &inArray{},
		"pointer":                &viaPointer{},
		"interface struct":       &inInterface{I: walkInner{}},
	} {
		err := Walk(obj, func(string, *happydns.Secret) error { return nil })
		if !errors.Is(err, ErrUnsupportedSecret) {
			t.Errorf("%s: Walk error = %v, want ErrUnsupportedSecret", name, err)
		}
	}
}

func TestWalkHandlesRecursiveTypes(t *testing.T) {
	type node struct {
		Children []node `json:"children"`
		Name     string `json:"name"`
	}
	type secretNode struct {
		Children []secretNode    `json:"children"`
		Key      happydns.Secret `json:"key"`
	}
	type root struct {
		Tree  node            `json:"tree"`
		Token happydns.Secret `json:"token"`
	}
	type secretRoot struct {
		T secretNode `json:"t"`
	}

	if got := collect(t, &root{Tree: node{Children: []node{{Name: "a"}}}}); !slices.Equal(got, []string{"token"}) {
		t.Errorf("paths = %v, want [token]", got)
	}

	err := Walk(&secretRoot{}, func(string, *happydns.Secret) error { return nil })
	if !errors.Is(err, ErrUnsupportedSecret) {
		t.Errorf("Walk error = %v, want ErrUnsupportedSecret", err)
	}
}

func TestWalkFollowsEmbeddedPointers(t *testing.T) {
	type holder struct {
		*walkEmbedded
		Token happydns.Secret `json:"token"`
	}

	if got := collect(t, &holder{walkEmbedded: &walkEmbedded{}}); !slices.Equal(got, []string{"appkey", "token"}) {
		t.Errorf("paths = %v, want [appkey token]", got)
	}
	if got := collect(t, &holder{}); !slices.Equal(got, []string{"token"}) {
		t.Errorf("paths with a nil embedded pointer = %v, want [token]", got)
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

// A typed nil in an interface holds no object, as a nil pointer field.
func TestWalkSkipsTypedNilInInterface(t *testing.T) {
	type holder struct {
		Body any `json:"Provider"`
	}

	if got := collect(t, &holder{Body: (*walkInner)(nil)}); len(got) != 0 {
		t.Errorf("paths = %v, want none", got)
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

type mapHolder struct {
	Headers map[string]happydns.Secret `json:"headers,omitempty"`
	Token   happydns.Secret            `json:"token"`
}

// A secret in a map with string keys is reached under its key, quoted so that
// a key holding a dot or a bracket cannot pass for another path, in key
// order so that two walks visit the secrets in the same order.
func TestWalkFollowsStringKeyedMaps(t *testing.T) {
	obj := &mapHolder{Headers: map[string]happydns.Secret{
		"b":      happydns.NewSecret("2"),
		"a":      happydns.NewSecret("1"),
		"x.y":    happydns.NewSecret("3"),
		`z"]["w`: happydns.NewSecret("4"),
	}}

	got := collect(t, obj)
	want := []string{`headers["a"]`, `headers["b"]`, `headers["x.y"]`, `headers["z\"][\"w"]`, "token"}
	if !slices.Equal(got, want) {
		t.Errorf("paths = %v, want %v", got, want)
	}

	if got := collect(t, &mapHolder{}); !slices.Equal(got, []string{"token"}) {
		t.Errorf("paths of a nil map = %v, want [token]", got)
	}
}

func TestWalkCanMutateMapValues(t *testing.T) {
	obj := &mapHolder{Headers: map[string]happydns.Secret{"a": happydns.NewSecret("1")}}

	err := Walk(obj, func(path string, s *happydns.Secret) error {
		*s = happydns.NewSecret("v-" + path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if v := obj.Headers["a"].Reveal(); v != `v-headers["a"]` {
		t.Errorf(`headers["a"] = %q`, v)
	}
}
