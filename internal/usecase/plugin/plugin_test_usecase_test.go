package plugin_test

import (
	"sort"
	"testing"

	uc "git.happydns.org/happyDomain/internal/usecase/plugin"
	"git.happydns.org/happyDomain/model"
)

func TestSortByPluginName(t *testing.T) {
	slice := []*happydns.PluginOptionsPositional{
		{PluginName: "zeta"},
		{PluginName: "alpha"},
		{PluginName: "beta"},
	}

	sort.Sort(uc.ByOptionPosition(slice))

	got := []string{slice[0].PluginName, slice[1].PluginName, slice[2].PluginName}
	want := []string{"alpha", "beta", "zeta"}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("expected %v, got %v", want, got)
			break
		}
	}
}

func TestNilBeforeNonNil(t *testing.T) {
	uid, _ := happydns.NewRandomIdentifier()
	slice := []*happydns.PluginOptionsPositional{
		{PluginName: "alpha", UserId: &uid},
		{PluginName: "alpha", UserId: nil},
	}

	sort.Sort(uc.ByOptionPosition(slice))

	if slice[0].UserId != nil {
		t.Errorf("expected nil UserId first, got %+v", slice[0].UserId)
	}
}

func TestDomainIdOrder(t *testing.T) {
	did, _ := happydns.NewRandomIdentifier()
	slice := []*happydns.PluginOptionsPositional{
		{PluginName: "alpha", UserId: nil, DomainId: &did},
		{PluginName: "alpha", UserId: nil, DomainId: nil},
	}

	sort.Sort(uc.ByOptionPosition(slice))

	if slice[0].DomainId != nil {
		t.Errorf("expected nil DomainId first, got %+v", slice[0].DomainId)
	}
}

func TestServiceIdOrder(t *testing.T) {
	sid, _ := happydns.NewRandomIdentifier()
	slice := []*happydns.PluginOptionsPositional{
		{PluginName: "alpha", UserId: nil, DomainId: nil, ServiceId: &sid},
		{PluginName: "alpha", UserId: nil, DomainId: nil, ServiceId: nil},
	}

	sort.Sort(uc.ByOptionPosition(slice))

	if slice[0].ServiceId != nil {
		t.Errorf("expected nil ServiceId first, got %+v", slice[0].ServiceId)
	}
}

func TestStableGrouping(t *testing.T) {
	uid, _ := happydns.NewRandomIdentifier()

	slice := []*happydns.PluginOptionsPositional{
		{PluginName: "alpha", UserId: &uid},
		{PluginName: "alpha", UserId: &uid},
	}

	sort.Sort(uc.ByOptionPosition(slice))
	if slice[0].PluginName != slice[1].PluginName {
		t.Errorf("expected grouping, got %+v vs %+v", slice[0], slice[1])
	}
}
