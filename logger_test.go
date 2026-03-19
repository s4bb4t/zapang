package zapang

import (
	"bytes"
	"context"
	"testing"

	"github.com/go-faster/errors"

	"go.uber.org/zap"
)

func TestRealExample(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := New(ctx, "serviceName", Config{
		Level:       "debug",
		Environment: "local",
	}, nil)

	err := errors.New("failed to parse..")
	err = errors.Wrap(err, "parse some")
	err = errors.Wrap(err, "handle parse request")

	l.Error("test error:", zap.Error(err))
}

func TestExportJSON(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var buf bytes.Buffer
	l := New(ctx, "serviceName", Config{
		Level:       "debug",
		Environment: "prod",
		ExportPath:  "stderr",
	}, &buf)

	err := errors.New("failed to parse..")
	err = errors.Wrap(err, "parse some")
	err = errors.Wrap(err, "handle parse request")

	l.Error("test error:", zap.Error(err))

	t.Log("=== writer output ===")
	t.Log(buf.String())
}
