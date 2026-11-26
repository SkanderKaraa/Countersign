// Package model holds Countersign's shared types: identities, actions,
// resources, grants and the other records in the permission store.
//
// Several fixed rules hold by construction here (PRD NFR-5): a request has
// no approver field, and an approver is a PersonID, which can't be built
// from an agent identity.
package model

import (
	"errors"
	"fmt"
	"strings"
)

// Identity is any principal: a person (from Keycloak) or an agent or
// service (from Kubernetes). Its string form is "person:<id>" or "agent:<id>".
type Identity struct {
	kind string // "person" or "agent"
	id   string
}

// PersonID is an identity that is guaranteed to be a person. It can only be
// built by ParsePerson, so an agent identity can never become one
// (NFR-5 rules 5 and 12).
type PersonID struct{ id string }

// AgentID is an identity that is guaranteed to be an agent or service.
type AgentID struct{ id string }

var ErrBadIdentity = errors.New("malformed identity")

// ParseIdentity accepts "person:<id>" or "agent:<id>" and nothing else.
func ParseIdentity(s string) (Identity, error) {
	kind, id, ok := strings.Cut(s, ":")
	if !ok || id == "" || !validSegment(id) {
		return Identity{}, fmt.Errorf("%w: %q", ErrBadIdentity, s)
	}
	if kind != "person" && kind != "agent" {
		return Identity{}, fmt.Errorf("%w: unknown kind %q", ErrBadIdentity, kind)
	}
	return Identity{kind: kind, id: id}, nil
}

// ParsePerson succeeds only for a person identity.
func ParsePerson(s string) (PersonID, error) {
	i, err := ParseIdentity(s)
	if err != nil {
		return PersonID{}, err
	}
	if i.kind != "person" {
		return PersonID{}, fmt.Errorf("%w: %q is not a person", ErrBadIdentity, s)
	}
	return PersonID{id: i.id}, nil
}

// ParseAgent succeeds only for an agent identity.
func ParseAgent(s string) (AgentID, error) {
	i, err := ParseIdentity(s)
	if err != nil {
		return AgentID{}, err
	}
	if i.kind != "agent" {
		return AgentID{}, fmt.Errorf("%w: %q is not an agent", ErrBadIdentity, s)
	}
	return AgentID{id: i.id}, nil
}

func (i Identity) String() string { return i.kind + ":" + i.id }
func (i Identity) IsZero() bool   { return i.kind == "" }
func (i Identity) IsAgent() bool  { return i.kind == "agent" }

func (p PersonID) String() string     { return "person:" + p.id }
func (p PersonID) Identity() Identity { return Identity{kind: "person", id: p.id} }
func (p PersonID) IsZero() bool       { return p.id == "" }

func (a AgentID) String() string     { return "agent:" + a.id }
func (a AgentID) Identity() Identity { return Identity{kind: "agent", id: a.id} }
