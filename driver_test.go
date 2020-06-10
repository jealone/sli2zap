package zap

import (
	"testing"

	"github.com/jealone/sli4go"
	"go.uber.org/zap"
)

func TestZap(t *testing.T) {
	logger, err := zap.NewProduction()

	if err != nil {
		t.Fatal(err)
	}

	registerZap(logger)
	sli4go.Info("test zap log")
}
