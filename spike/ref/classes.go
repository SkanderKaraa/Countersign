package ref

import (
	"sort"
	"time"

	"countersign/spike/model"
)

// isFloor reports whether an action on a resource is in the floor of four:
// production, real secrets, money or deletion (FR-5). Every classification
// that covers the resource counts, so the strictest wins: a narrower
// classification can't clear a wider one's flag. An unclassified resource
// counts as production, so it fails closed.
func (s *State) isFloor(a model.Action, r model.Resource) bool {
	if a == model.ActionDelete || a == model.ActionPay {
		return true
	}
	return floorByInfo(s.resourceInfo, r)
}

// floorByInfo reports whether the classifications put r in the floor: some
// covering classification has a flag, or none covers it at all.
func floorByInfo(infos []model.ResourceInfo, r model.Resource) bool {
	found := false
	for _, ri := range infos {
		if !ri.Resource.Covers(r) {
			continue
		}
		found = true
		if ri.Production || ri.RealSecret || ri.Money {
			return true
		}
	}
	return !found
}

// ClassOf is the action class (FR-5).
func (s *State) ClassOf(a model.Action, r model.Resource) model.Class {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.classOf(a, r)
}

// classOf is ClassOf without the lock, for use inside writes. The floor
// always wins. Otherwise an action is contained only if a contained marking
// covers it and no human-only marking does. Unmarked means human-only.
func (s *State) classOf(a model.Action, r model.Resource) model.Class {
	if s.isFloor(a, r) {
		return model.ClassHumanOnly
	}
	contained := false
	for _, m := range s.markings {
		if m.Action != a || !m.Resource.Covers(r) {
			continue
		}
		switch m.Class {
		case model.ClassContained:
			contained = true
		default:
			// Human-only, or a class that should never have been stored:
			// either way it can't make the action contained.
			return model.ClassHumanOnly
		}
	}
	if contained {
		return model.ClassContained
	}
	return model.ClassHumanOnly
}

func (s *State) isAdmin(p model.PersonID, now time.Time) bool {
	a, ok := s.admins[p]
	return ok && now.Before(a.EndsAt)
}

// hasAuthorityOver reports "authority over a resource" (glossary). In v1
// only administrators have it (Skander, 2026-11-17). Approval authority is
// not authority over the resource.
func (s *State) hasAuthorityOver(p model.PersonID, r model.Resource, now time.Time) bool {
	return s.isAdmin(p, now)
}

// canApprove reports whether p may approve this action on this resource:
// an active administrator, or a holder of unexpired approval authority for
// it (FR-7, FR-9, FR-11).
func (s *State) canApprove(p model.PersonID, a model.Action, r model.Resource, now time.Time) bool {
	if s.isAdmin(p, now) {
		return true
	}
	for _, aa := range s.authorities {
		if aa.Holder != p || !now.Before(aa.EndsAt) || !aa.Resource.Covers(r) {
			continue
		}
		for _, act := range aa.Actions {
			if act == a {
				return true
			}
		}
	}
	return false
}

// authorityScopes returns the scopes of p's unexpired approval authority for
// action a. The engine decides whether one covers the resource.
func (s *State) authorityScopes(p model.PersonID, a model.Action, now time.Time) []string {
	var out []string
	for _, aa := range s.authorities {
		if aa.Holder != p || !now.Before(aa.EndsAt) {
			continue
		}
		for _, act := range aa.Actions {
			if act == a {
				out = append(out, aa.Resource.String())
				break
			}
		}
	}
	return out
}

// approversFor lists who could approve, administrators included, leaving
// out the requester (FR-21). The list is sorted, so it doesn't depend on
// storage order.
func (s *State) approversFor(a model.Action, r model.Resource, requester model.Identity, now time.Time) []model.PersonID {
	seen := map[model.PersonID]bool{}
	var out []model.PersonID
	add := func(p model.PersonID) {
		if seen[p] || p.Identity() == requester || !s.canApprove(p, a, r, now) {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	for p := range s.admins {
		add(p)
	}
	for _, aa := range s.authorities {
		add(aa.Holder)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}
