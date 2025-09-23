package check_test

import (
	"slices"
	"testing"

	uc "git.happydns.org/happyDomain/internal/usecase/check"
	"git.happydns.org/happyDomain/model"
)

func TestSortByCheckName(t *testing.T) {
	slice := []*happydns.CheckerOptionsPositional{
		{CheckName: "zeta"},
		{CheckName: "alpha"},
		{CheckName: "beta"},
	}

	slices.SortFunc(slice, uc.CompareCheckerOptionsPositional)

	got := []string{slice[0].CheckName, slice[1].CheckName, slice[2].CheckName}
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
	slice := []*happydns.CheckerOptionsPositional{
		{CheckName: "alpha", UserId: &uid},
		{CheckName: "alpha", UserId: nil},
	}

	slices.SortFunc(slice, uc.CompareCheckerOptionsPositional)

	if slice[0].UserId != nil {
		t.Errorf("expected nil UserId first, got %+v", slice[0].UserId)
	}
}

func TestDomainIdOrder(t *testing.T) {
	did, _ := happydns.NewRandomIdentifier()
	slice := []*happydns.CheckerOptionsPositional{
		{CheckName: "alpha", UserId: nil, DomainId: &did},
		{CheckName: "alpha", UserId: nil, DomainId: nil},
	}

	slices.SortFunc(slice, uc.CompareCheckerOptionsPositional)

	if slice[0].DomainId != nil {
		t.Errorf("expected nil DomainId first, got %+v", slice[0].DomainId)
	}
}

func TestServiceIdOrder(t *testing.T) {
	sid, _ := happydns.NewRandomIdentifier()
	slice := []*happydns.CheckerOptionsPositional{
		{CheckName: "alpha", UserId: nil, DomainId: nil, ServiceId: &sid},
		{CheckName: "alpha", UserId: nil, DomainId: nil, ServiceId: nil},
	}

	slices.SortFunc(slice, uc.CompareCheckerOptionsPositional)

	if slice[0].ServiceId != nil {
		t.Errorf("expected nil ServiceId first, got %+v", slice[0].ServiceId)
	}
}

func TestStableGrouping(t *testing.T) {
	uid, _ := happydns.NewRandomIdentifier()

	slice := []*happydns.CheckerOptionsPositional{
		{CheckName: "alpha", UserId: &uid},
		{CheckName: "alpha", UserId: &uid},
	}

	slices.SortFunc(slice, uc.CompareCheckerOptionsPositional)
	if slice[0].CheckName != slice[1].CheckName {
		t.Errorf("expected grouping, got %+v vs %+v", slice[0], slice[1])
	}
}
