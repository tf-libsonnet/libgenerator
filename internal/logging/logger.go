package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Level          zap.AtomicLevel
	Encoding       string
	NoColor        bool
	WithStackTrace bool
}

func NewConfig(lvl string) (*Config, error) {
	atomicLvl, err := zap.ParseAtomicLevel(lvl)
	if err != nil {
		return nil, err
	}
	cfg := &Config{Level: atomicLvl}
	return cfg, nil
}

// GetZapConfig returns the configuration for zap.
func GetZapConfig(c *Config) zap.Config {
	zapC := zap.NewProductionConfig()
	zapC.EncoderConfig.TimeKey = ""
	zapC.EncoderConfig.CallerKey = ""
	zapC.EncoderConfig.StacktraceKey = ""

	if c == nil {
		zapC.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zapC.Encoding = "console"
		return zapC
	}

	zapC.Level = c.Level
	if c.Encoding != "" {
		zapC.Encoding = c.Encoding
	}
	if !c.NoColor {
		zapC.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	if c.WithStackTrace {
		zapC.EncoderConfig.StacktraceKey = "trace"
	}

	return zapC
}

// GetSugardLoggerWithoutConfig returns a generic zap sugared logger without configuration. This is useful for emitting
// log messages before the initialization stage.
func GetSugaredLoggerWithoutConfig() (*zap.SugaredLogger, error) {
	zapC := GetZapConfig(&Config{})
	logger, err := zapC.Build()
	if err != nil {
		return nil, err
	}
	sugar := logger.Sugar()
	return sugar, nil
}

// GetSugardLoggerForTest returns a generic zap sugared logger without configuration that should be used during testing.
// Namely, this initializes the logger to maximize logging output.
func GetSugaredLoggerForTest() *zap.SugaredLogger {
	return GetSugaredLogger(nil)
}

// GetLogger returns the zap logger object. This can be used to customize the logger object.
func GetLogger(c *Config) *zap.Logger {
	zapC := GetZapConfig(c)
	logger := zap.Must(zapC.Build())
	return logger
}

// GetSugaredLogger returns the zap sugared logger which can be used to emit log messages from the app.
func GetSugaredLogger(c *Config) *zap.SugaredLogger {
	return GetLogger(c).Sugar()
}
