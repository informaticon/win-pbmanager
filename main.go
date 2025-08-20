package main

import (
	"log/slog"

	"github.com/informaticon/dev.win.base.pbmanager/cmd"
	logging "github.com/informaticon/lib.go.base.logging"
	"github.com/informaticon/lib.go.base.logging/rule"
	"github.com/informaticon/lib.go.base.logging/sender/eventlog"
	"github.com/informaticon/lib.go.base.logging/sender/std"
	"github.com/informaticon/lib.go.base.logging/transformer/pretty"
)

func main() {
	slog.SetDefault(slog.New(logging.New(nil,
		rule.New().
			Transform(pretty.New()).
			Send(std.New()),
		rule.New().
			Send(eventlog.New("dev.win.base.pbmanager", eventlog.WithExternal())),
	)))

	cmd.Execute()
}
