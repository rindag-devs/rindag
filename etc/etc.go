package etc

import (
	"os"

	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v3"
)

var (
	// Paths to the config file and the fallback config files.
	Paths = []string{"config.yml", "/etc/rindag.yml"}
	// Config is the global configuration
	Config *Configuration
)

// Configuration is the Configuration structure.
type Configuration struct {
	LogLevel string `yaml:"log_level"`

	Judges map[string]struct {
		Host  string `yaml:"host"`
		Token string `yaml:"token"`
	} `yaml:"judges"`

	Compile struct {
		Cmd []string `yaml:"cmd"`
	} `yaml:"compile"`

	Testlib struct {
		Path string `yaml:"path"`
	} `yaml:"testlib"`

	Validator struct {
		Compile struct {
			Cmd         []string `yaml:"cmd"`
			TimeLimit   uint64   `yaml:"time_limit"`
			MemoryLimit uint64   `yaml:"memory_limit"`
			StderrLimit int64    `yaml:"stderr_limit"`
		} `yaml:"compile"`

		Run struct {
			TimeLimit   uint64 `yaml:"time_limit"`
			MemoryLimit uint64 `yaml:"memory_limit"`
			StderrLimit int64  `yaml:"stderr_limit"`
		} `yaml:"run"`
	} `yaml:"validator"`

	Checker struct {
		BuiltinPath string `yaml:"builtin_path"`
		Compile     struct {
			Cmd         []string `yaml:"cmd"`
			TimeLimit   uint64   `yaml:"time_limit"`
			MemoryLimit uint64   `yaml:"memory_limit"`
			StderrLimit int64    `yaml:"stderr_limit"`
		} `yaml:"compile"`

		Run struct {
			TimeLimit   uint64 `yaml:"time_limit"`
			MemoryLimit uint64 `yaml:"memory_limit"`
			StderrLimit int64  `yaml:"stderr_limit"`
		} `yaml:"run"`
	} `yaml:"checker"`

	Generator struct {
		Compile struct {
			Cmd         []string `yaml:"cmd"`
			TimeLimit   uint64   `yaml:"time_limit"`
			MemoryLimit uint64   `yaml:"memory_limit"`
			StderrLimit int64    `yaml:"stderr_limit"`
		} `yaml:"compile"`

		Run struct {
			TimeLimit   uint64 `yaml:"time_limit"`
			MemoryLimit uint64 `yaml:"memory_limit"`
			StderrLimit int64  `yaml:"stderr_limit"`
		} `yaml:"run"`
	} `yaml:"generator"`

	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	} `yaml:"database"`

	Storage struct {
		// type of storage (local or minio).
		Type  string `yaml:"type"`
		Local struct {
			// path to the storage directory (like /var/lib/rindag/storage).
			Path string `yaml:"path"`
		} `yaml:"local"`
		MinIO struct {
			Endpoint        string `yaml:"endpoint"`
			AccessKeyID     string `yaml:"access_key_id"`
			SecretAccessKey string `yaml:"secret_access_key"`
			UseSSL          bool   `yaml:"use_ssl"`
			Bucket          string `yaml:"bucket"`
		} `yaml:"minio"`
	} `yaml:"storage"`
}

// LoadConfig loads the configuration from the given file.
func LoadConfig(file string) interface{} {
	f, err := os.Open(file)
	if err != nil {
		log.WithError(err).WithField("file", file).Fatal("failed to open config file")
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.WithError(err).Error("Failed to close config file")
		}
	}(f)
	decoder := yaml.NewDecoder(f)
	config := Configuration{}
	err = decoder.Decode(&config)
	if err != nil {
		log.WithError(err).Fatal("Failed to decode config file")
	}
	return &config
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
	// Load global configuration.
	for _, file := range Paths {
		if _, err := os.Stat(file); err != nil {
			continue
		}
		log.Info("Loading config file: ", file)
		Config = LoadConfig(file).(*Configuration)
	}
	if Config == nil {
		log.Fatal("No config file found")
	}
	// Set log level.
	setLogLevel(Config.LogLevel)

	log.Info("Loaded config")
}
