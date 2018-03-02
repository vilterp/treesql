package log

import (
	"context"
	"fmt"
	"log"
	"strings"
)

const (
	ConnIDKey    = "ConnID"
	ChannelIDKey = "ChanID"
)

func ctxToString(ctx context.Context) string {
	var tags []string
	if connID := ctx.Value(ConnIDKey); connID != nil {
		tags = append(tags, fmt.Sprintf("conn=%d", connID))
	}
	if stmtID := ctx.Value(ChannelIDKey); stmtID != nil {
		tags = append(tags, fmt.Sprintf("stmt=%d", stmtID))
	}
	return fmt.Sprintf("[%s]", strings.Join(tags, ","))
}

func Println(l Loggable, args ...interface{}) {
	ctx := l.Ctx()
	var allArgs []interface{}
	allArgs = append(allArgs, ctxToString(ctx))
	allArgs = append(allArgs, args...)
	log.Println(allArgs...)
}

func Printf(l Loggable, format string, args ...interface{}) {
	ctx := l.Ctx()
	log.Printf("%s %s", ctxToString(ctx), fmt.Sprintf(format, args...))
}

type Loggable interface {
	Ctx() context.Context
}
