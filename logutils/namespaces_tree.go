package logutils

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"go.uber.org/zap/zapcore"
)

var namespacesRegex = regexp.MustCompile(`^([a-zA-Z0-9_\.]+:(debug|info|warn|error))(,[a-zA-Z0-9_\.]+:(debug|info|warn|error))*$`)
var errInvalidNamespacesFormat = fmt.Errorf("invalid namespaces format")

type namespacesTree struct {
	namespaces map[string]*namespaceNode
	minLevel   zapcore.Level
	mu         sync.RWMutex
}

type namespaceNode struct {
	level    zapcore.Level
	children map[string]*namespaceNode
	explicit bool
}

func newNamespacesTree() *namespacesTree {
	return &namespacesTree{
		namespaces: make(map[string]*namespaceNode),
		minLevel:   zapcore.InvalidLevel,
	}
}

// buildNamespacesTree creates a new tree instance from a string of namespaces.
// The namespaces string should be a comma-separated list of namespace:level pairs,
// where level is one of: "debug", "info", "warn", "error".
//
// Example: "namespace1:debug,namespace2.namespace3:error"
func buildNamespacesTree(namespaces string) (*namespacesTree, error) {
	tree := newNamespacesTree()

	err := tree.Rebuild(namespaces)
	if err != nil {
		return nil, err
	}

	return tree, nil
}

// LevelFor finds the log level for the given namespace by searching for the closest match in the tree.
// In case of no match, it returns zapcore.InvalidLevel.
func (t *namespacesTree) LevelFor(namespace string) zapcore.Level {
	t.mu.RLock()
	defer t.mu.RUnlock()

	currentNode := t.namespaces
	var lastFound *namespaceNode

	for _, name := range strings.Split(namespace, ".") {
		if node, exists := currentNode[name]; exists {
			if node.explicit {
				lastFound = node
			}
			currentNode = node.children
		} else {
			break
		}
	}

	if lastFound != nil {
		return lastFound.level
	}

	// Default to zapcore.InvalidLevel if no match found.
	return zapcore.InvalidLevel
}

// Rebuild updates the tree with a new set of namespaces.
func (t *namespacesTree) Rebuild(namespaces string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if namespaces == "" {
		t.namespaces = make(map[string]*namespaceNode)
		t.minLevel = zapcore.InvalidLevel
		return nil
	}

	if !namespacesRegex.MatchString(namespaces) {
		return errInvalidNamespacesFormat
	}

	for _, ns := range strings.Split(namespaces, ",") {
		parts := strings.Split(ns, ":")
		if len(parts) != 2 {
			return errInvalidNamespacesFormat
		}

		lvl, err := lvlFromString(parts[1])
		if err != nil {
			return err
		}

		t.addNamespace(parts[0], lvl)

		if t.minLevel == zapcore.InvalidLevel || lvl < t.minLevel {
			t.minLevel = lvl
		}
	}

	return nil
}

func (t *namespacesTree) MinLevel() zapcore.Level {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.minLevel
}

// addNamespace adds a namespace with a specific logging level to the tree.
func (t *namespacesTree) addNamespace(namespace string, level zapcore.Level) {
	currentParent := t.namespaces
	var currentNode *namespaceNode

	for _, name := range strings.Split(namespace, ".") {
		if node, exists := currentParent[name]; exists {
			currentNode = node
		} else {
			newNode := &namespaceNode{
				children: make(map[string]*namespaceNode),
			}
			currentParent[name] = newNode
			currentNode = newNode
		}
		currentParent = currentNode.children
	}

	currentNode.level = level
	currentNode.explicit = true
}
