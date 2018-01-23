// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package pathmap

import (
	"bytes"
	"fmt"
	"sort"
)

// PathMap associates Paths to a values. It allows wildcards. The
// primary use of PathMap is to be able to register handlers to paths
// that can be efficiently looked up every time a path is updated.
//
// For example:
//
// m.Set({"interfaces", "*", "adminStatus"}, AdminStatusHandler)
// m.Set({"interface", "Management1", "adminStatus"}, Management1AdminStatusHandler)
//
// m.Visit({"interfaces", "Ethernet3/32/1", "adminStatus"}, HandlerExecutor)
// >> AdminStatusHandler gets passed to HandlerExecutor
// m.Visit({"interfaces", "Management1", "adminStatus"}, HandlerExecutor)
// >> AdminStatusHandler and Management1AdminStatusHandler gets passed to HandlerExecutor
//
// Note, Visit performance is typically linearly with the length of
// the path. But, it can be as bad as O(2^len(Path)) when TreeMap
// nodes have children and a wildcard associated with it. For example,
// if these paths were registered:
//
// m.Set([]string{"foo", "bar", "baz"}, 1)
// m.Set([]string{"*", "bar", "baz"}, 2)
// m.Set([]string{"*", "*", "baz"}, 3)
// m.Set([]string{"*", "*", "*"}, 4)
// m.Set([]string{"foo", "*", "*"}, 5)
// m.Set([]string{"foo", "bar", "*"}, 6)
// m.Set([]string{"foo", "*", "baz"}, 7)
// m.Set([]string{"*", "bar", "*"}, 8)
//
// m.Visit([]{"foo","bar","baz"}, Foo) // 2^3 nodes traversed
//
// This shouldn't be a concern with our paths because it is likely
// that a TreeMap node will either have a wildcard or children, not
// both. A TreeMap node that corresponds to a collection will often be a
// wildcard, otherwise it will have specific children.
type PathMap interface {
	// Visit calls f for every registration in the PathMap that
	// matches path. For example,
	//
	// m.Set({"foo", "bar"}, 1)
	// m.Set({"*", "bar"}, 2)
	//
	// m.Visit({"foo", "bar"}, Printer)
	// >> Calls Printer(1) and Printer(2)
	Visit(path []string, f VisitorFunc) error

	// VisitPrefix calls f for every registration in the PathMap that
	// is a prefix of path. For example,
	//
	// m.Set({}, 0)
	// m.Set({"foo"}, 1)
	// m.Set({"foo", "bar"}, 2)
	// m.Set({"foo", "quux"}, 3)
	// m.Set({"*", "bar"}, 4)
	//
	// m.VisitPrefix({"foo", "bar", "baz"}, Printer)
	// >> Calls Printer on values 0, 1, 2, and 4
	VisitPrefix(path []string, f VisitorFunc) error

	// Get returns the mapping for path. This returns the exact
	// mapping for path. For example, if you register two paths
	//
	// m.Set({"foo", "bar"}, 1)
	// m.Set({"*", "bar"}, 2)
	//
	// m.Get({"foo", "bar"}) => 1
	// m.Get({"*", "bar"}) => 2
	Get(path []string) interface{}

	// Set a mapping of path to value. Path may contain wildcards. Set
	// replaces what was there before.
	Set(path []string, v interface{})

	// Delete removes the mapping for path
	Delete(path []string) bool
}

// Wildcard is a special string representing any possible path
const Wildcard string = "*"

type node struct {
	val      interface{}
	wildcard *node
	children map[string]*node
}

// New creates a new PathMap
func New() PathMap {
	return &node{}
}

// VisitorFunc is the func type passed to Visit
type VisitorFunc func(v interface{}) error

// Visit calls f for every matching registration in the PathMap
func (n *node) Visit(path []string, f VisitorFunc) error {
	for i, element := range path {
		if n.wildcard != nil {
			if err := n.wildcard.Visit(path[i+1:], f); err != nil {
				return err
			}
		}
		next, ok := n.children[element]
		if !ok {
			return nil
		}
		n = next
	}
	if n.val == nil {
		return nil
	}
	return f(n.val)
}

// VisitPrefix calls f for every registered path that is a prefix of
// the path
func (n *node) VisitPrefix(path []string, f VisitorFunc) error {
	for i, element := range path {
		// Call f on each node we visit
		if n.val != nil {
			if err := f(n.val); err != nil {
				return err
			}
		}
		if n.wildcard != nil {
			if err := n.wildcard.VisitPrefix(path[i+1:], f); err != nil {
				return err
			}
		}
		next, ok := n.children[element]
		if !ok {
			return nil
		}
		n = next
	}
	if n.val == nil {
		return nil
	}
	// Call f on the final node
	return f(n.val)
}

// Get returns the mapping for path
func (n *node) Get(path []string) interface{} {
	for _, element := range path {
		if element == Wildcard {
			if n.wildcard == nil {
				return nil
			}
			n = n.wildcard
			continue
		}
		next, ok := n.children[element]
		if !ok {
			return nil
		}
		n = next
	}
	return n.val
}

// Set a mapping of path to value. Path may contain wildcards. Set
// replaces what was there before.
func (n *node) Set(path []string, v interface{}) {
	for _, element := range path {
		if element == Wildcard {
			if n.wildcard == nil {
				n.wildcard = &node{}
			}
			n = n.wildcard
			continue
		}
		if n.children == nil {
			n.children = map[string]*node{}
		}
		next, ok := n.children[element]
		if !ok {
			next = &node{}
			n.children[element] = next
		}
		n = next
	}
	n.val = v
}

// Delete removes the mapping for path
func (n *node) Delete(path []string) bool {
	nodes := make([]*node, len(path)+1)
	for i, element := range path {
		nodes[i] = n
		if element == Wildcard {
			if n.wildcard == nil {
				return false
			}
			n = n.wildcard
			continue
		}
		next, ok := n.children[element]
		if !ok {
			return false
		}
		n = next
	}
	n.val = nil
	nodes[len(path)] = n

	// See if we can delete any node objects
	for i := len(path); i > 0; i-- {
		n = nodes[i]
		if n.val != nil || n.wildcard != nil || len(n.children) > 0 {
			break
		}
		parent := nodes[i-1]
		element := path[i-1]
		if element == Wildcard {
			parent.wildcard = nil
		} else {
			delete(parent.children, element)
		}

	}
	return true
}

func (n *node) String() string {
	var b bytes.Buffer
	n.write(&b, "")
	return b.String()
}

func (n *node) write(b *bytes.Buffer, indent string) {
	if n.val != nil {
		b.WriteString(indent)
		fmt.Fprintf(b, "Val: %v", n.val)
		b.WriteString("\n")
	}
	if n.wildcard != nil {
		b.WriteString(indent)
		fmt.Fprintf(b, "Child %q:\n", Wildcard)
		n.wildcard.write(b, indent+"  ")
	}
	children := make([]string, 0, len(n.children))
	for name := range n.children {
		children = append(children, name)
	}
	sort.Strings(children)

	for _, name := range children {
		child := n.children[name]
		b.WriteString(indent)
		fmt.Fprintf(b, "Child %q:\n", name)
		child.write(b, indent+"  ")
	}
}
