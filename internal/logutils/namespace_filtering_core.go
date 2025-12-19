package logutils

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type namespaceFilteringCore struct {
	*filteringCore
	*namespacesTree
}

func newNamespaceFilteringCore(core zapcore.Core) *namespaceFilteringCore {
	namespacesTree := newNamespacesTree()

	levelEnablerFunc := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		minLvl := namespacesTree.MinLevel()
		if minLvl != zapcore.InvalidLevel {
			return minLvl.Enabled(lvl)
		}
		return zapcore.LevelOf(core).Enabled(lvl)
	})

	filterFunc := func(ent zapcore.Entry) bool {
		namespaceLvl := namespacesTree.LevelFor(ent.LoggerName)
		if namespaceLvl != zapcore.InvalidLevel {
			return namespaceLvl.Enabled(ent.Level)
		}
		return zapcore.LevelOf(core).Enabled(ent.Level)
	}

	return &namespaceFilteringCore{
		filteringCore:  newFilteringCore(core, levelEnablerFunc, filterFunc),
		namespacesTree: namespacesTree,
	}
}
