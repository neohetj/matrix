package log

import (
	"bytes"
	"context"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStdLogger_With(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil) // Restore default

	logger := &StdLogger{}
	ctxLogger := logger.With("key", "value", "err", "some error\nmulti line")

	ctxLogger.Errorf(context.Background(), "Main message")

	output := buf.String()
	assert.Contains(t, output, "[ERROR] Main message")
	assert.Contains(t, output, "key=value")
	assert.Contains(t, output, "err=some error\nmulti line")
}
