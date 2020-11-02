package sli2zap

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func newLoggerConfig() *LogConfig {
	logConf := &LogConfig{}
	path, _ := filepath.Abs("config/log.yml")

	file, _ := os.Open(path)
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	_ = decoder.Decode(logConf)
	return logConf
}

func TestNewLogger(t *testing.T) {

	logger := NewLogger(newLoggerConfig(), zap.Fields(zap.Duration("test", time.Second)))

	logger.Info("test")

}

func BenchmarkNewLogger(b *testing.B) {
	b.ReportAllocs()
	logger := NewLogger(newLoggerConfig(), zap.Fields(zap.Duration("test", time.Second)))
	for i := 0; i < b.N; i++ {
		logger.Info("test")
	}
}
