package zapang

// Config holds configuration for the application logger.
type Config struct {
	// Level is the minimum enabled logging level.
	// Valid values: debug, info, warn, error, dpanic, panic, fatal
	Level string `yaml:"level" json:"level" mapstructure:"level"`

	// Environment controls logger behavior.
	// "local" - only human-readable console output
	// "dev", "prod" - human-readable console + optional JSON export
	Environment string `yaml:"environment" json:"environment" mapstructure:"environment"`

	// ExportPath is an optional path for JSON log export (only for dev/prod).
	// Can be a file path or "stdout"/"stderr".
	// If empty, JSON export is disabled.
	ExportPath string `yaml:"export_path" json:"export_path" mapstructure:"export_path"`

	// Sampling configures log sampling for high-throughput applications.
	Sampling *SamplingConfig `yaml:"sampling,omitempty" json:"sampling" mapstructure:"sampling"`

	// DisableCaller stops annotating logs with the calling function's file name and line number.
	DisableCaller bool `yaml:"disable_caller" json:"disable_caller" mapstructure:"disable_caller"`

	// DisableStacktrace disables automatic stacktrace capturing.
	DisableStacktrace bool `yaml:"disable_stacktrace" json:"disable_stacktrace" mapstructure:"disable_stacktrace"`

	// StacktraceLevel is the minimum level at which stacktraces are captured.
	// Valid values: debug, info, warn, error, dpanic, panic, fatal
	StacktraceLevel string `yaml:"stacktrace_level" json:"stacktrace_level" mapstructure:"stacktrace_level"`
}

// SamplingConfig sets a sampling policy for repeated log entries.
type SamplingConfig struct {
	// Initial is the number of entries with the same level and message to log per second.
	Initial int `yaml:"initial"`

	// Thereafter is the number of entries to drop for each duplicate after Initial.
	Thereafter int `yaml:"thereafter"`
}

// DefaultLoggerConfig returns a sensible default configuration.
func DefaultLoggerConfig() Config {
	return Config{
		Level:             "info",
		Environment:       EnvLocal,
		ExportPath:        "",
		DisableCaller:     false,
		DisableStacktrace: false,
		StacktraceLevel:   "error",
		Sampling: &SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
	}
}
