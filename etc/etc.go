package etc

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	// Config is the global configuration
	Config *Configuration
)

// Configuration is the Configuration structure.
type Configuration struct {
	LogLevel string `mapstructure:"log_level"`

	Judges map[string]struct {
		Host  string `mapstructure:"host"`
		Token string `mapstructure:"token"`
	} `mapstructure:"judges"`

	Compile struct {
		Cmd         []string `mapstructure:"cmd"`
		TimeLimit   uint64   `mapstructure:"time_limit"`
		MemoryLimit uint64   `mapstructure:"memory_limit"`
		StderrLimit int64    `mapstructure:"stderr_limit"`
	} `mapstructure:"compile"`

	Testlib struct {
		Path string `mapstructure:"path"`
	} `mapstructure:"testlib"`

	Validator struct {
		Compile struct {
			Args []string `mapstructure:"args"`
		} `mapstructure:"compile"`

		Run struct {
			TimeLimit   uint64 `mapstructure:"time_limit"`
			MemoryLimit uint64 `mapstructure:"memory_limit"`
			StderrLimit int64  `mapstructure:"stderr_limit"`
		} `mapstructure:"run"`
	} `mapstructure:"validator"`

	Checker struct {
		BuiltinPath string `mapstructure:"builtin_path"`
		Compile     struct {
			Args []string `mapstructure:"args"`
		} `mapstructure:"compile"`

		Run struct {
			TimeLimit   uint64 `mapstructure:"time_limit"`
			MemoryLimit uint64 `mapstructure:"memory_limit"`
			StderrLimit int64  `mapstructure:"stderr_limit"`
		} `mapstructure:"run"`
	} `mapstructure:"checker"`

	Generator struct {
		Compile struct {
			Args []string `mapstructure:"args"`
		} `mapstructure:"compile"`

		Run struct {
			TimeLimit   uint64 `mapstructure:"time_limit"`
			MemoryLimit uint64 `mapstructure:"memory_limit"`
			StderrLimit int64  `mapstructure:"stderr_limit"`
		} `mapstructure:"run"`
	} `mapstructure:"generator"`

	Database struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		DBName   string `mapstructure:"dbname"`
		UseSSL   bool   `mapstructure:"use_ssl"`
	} `mapstructure:"database"`

	Storage struct {
		// Type is the type of storage (local or minio).
		Type  string `mapstructure:"type"`
		Local struct {
			// Path is the path to the storage directory.
			Path string `mapstructure:"path"`
		} `mapstructure:"local"`
		MinIO struct {
			Endpoint        string `mapstructure:"endpoint"`
			AccessKeyID     string `mapstructure:"access_key_id"`
			SecretAccessKey string `mapstructure:"secret_access_key"`
			UseSSL          bool   `mapstructure:"use_ssl"`
			Bucket          string `mapstructure:"bucket"`
		} `mapstructure:"minio"`
	} `mapstructure:"storage"`
}

func setLogLevel(level string) {
	switch level {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "panic":
		log.SetLevel(log.PanicLevel)
	default:
		log.WithField("level", level).Fatal("Invalid log level")
	}
}

func init() {
	viper.SetConfigName("config")
	viper.AddConfigPath("/etc/rindag/")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		log.WithError(err).Fatal("Failed to read config")
	}
	if err := viper.UnmarshalExact(&Config); err != nil {
		log.Fatal(err)
	}
	setLogLevel(Config.LogLevel)
	log.Info("Loaded config")
}
