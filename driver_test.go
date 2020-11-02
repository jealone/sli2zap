package sli2zap

import (
	"testing"

	"github.com/jealone/sli4go"
)

func TestZap(t *testing.T) {
	logger := NewLogger(&LogConfig{})

	RegisterZap(logger)
	sli4go.Info("test zap log")
}
