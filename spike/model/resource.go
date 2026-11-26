package model

import (
	"errors"
	"fmt"
	"strings"
)

// Resource is a canonical path: lowercase segments of [a-z0-9._-], separated
// by "/", with no "." or ".." segments and no empty segments. It is either
// exact ("api/billing") or a subtree ("api/**"), where "**" may appear only
// as the last segment.
type Resource struct{ path string }

var ErrBadResource = errors.New("non-canonical resource")

// ParseResource refuses anything that isn't already canonical, rather than
// cleaning it up: a non-canonical path is treated as a crafted input.
func ParseResource(s string) (Resource, error) {
	if s == "" || strings.HasPrefix(s, "/") || strings.HasSuffix(s, "/") {
		return Resource{}, fmt.Errorf("%w: %q", ErrBadResource, s)
	}
	segs := strings.Split(s, "/")
	for i, seg := range segs {
		if seg == "**" {
			if i != len(segs)-1 || i == 0 {
				return Resource{}, fmt.Errorf("%w: %q", ErrBadResource, s)
			}
			continue
		}
		if seg == "" || seg == "." || seg == ".." || !validSegment(seg) {
			return Resource{}, fmt.Errorf("%w: %q", ErrBadResource, s)
		}
	}
	return Resource{path: s}, nil
}

func validSegment(seg string) bool {
	for _, r := range seg {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-'
		if !ok {
			return false
		}
	}
	return true
}

func (r Resource) String() string { return r.path }
func (r Resource) IsZero() bool   { return r.path == "" }

// IsSubtree reports whether r ends with the "**" marker.
func (r Resource) IsSubtree() bool { return strings.HasSuffix(r.path, "/**") }

// prefix is the subtree's root, without "/**".
func (r Resource) prefix() string { return strings.TrimSuffix(r.path, "/**") }

// Covers reports whether every resource in o is also in r. An exact path
// covers only itself; a subtree covers its root, everything below it, and
// any narrower subtree.
func (r Resource) Covers(o Resource) bool {
	if !r.IsSubtree() {
		return r.path == o.path
	}
	p := r.prefix()
	op := o.prefix()
	return op == p || strings.HasPrefix(op, p+"/")
}

// Overlap returns the narrower of two resources when one covers the other.
// Two resources where neither covers the other have no overlap in this
// model, because only exact paths and single subtrees exist.
func Overlap(a, b Resource) (Resource, bool) {
	switch {
	case a.Covers(b):
		return b, true
	case b.Covers(a):
		return a, true
	}
	return Resource{}, false
}
