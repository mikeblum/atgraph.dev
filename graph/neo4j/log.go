package neo4j

import (
	"fmt"
	"log/slog"

	"github.com/mikeblum/atgraph.dev/conf"
	neo4jLog "github.com/neo4j/neo4j-go-driver/v5/neo4j/log"
)

type LogBridge struct {
	neo4jLog.BoltLogger
	*slog.Logger
}

func neo4jLogBridge(log *conf.Log) *LogBridge {
	bridge := &LogBridge{}
	bridge.Logger = log.Logger.With(
		"driver", "bolt",
	)
	return bridge
}

func (l *LogBridge) LogClientMessage(context string, msg string, args ...any) {
	l.WithGroup(context).Debug(fmt.Sprintf(msg, args...))
}

func (l *LogBridge) LogServerMessage(context string, msg string, args ...any) {
	l.WithGroup(context).Debug(fmt.Sprintf(msg, args...))
}

// verify LogBridge implements BoltLogger interface
var _ neo4jLog.BoltLogger = &LogBridge{}
