package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type ConfigurationValidator interface {
	Validate() error
}

type Configuration struct {
	Debug      bool
	Verbosity  int
	SharedPath string `validate:"required"`
	Temporal   Temporal
	Worker     WorkerConfig
}

type Temporal struct {
	// Address is the Temporal server host and port (default: "localhost:7233").
	Address string

	// Namespace is the Temporal namespace the preprocessing worker should run
	// in (default: "default").
	Namespace string

	// TaskQueue is the Temporal task queue from which the preprocessing worker
	// will pull tasks.
	TaskQueue string `validate:"required"`

	// WorkflowName is the name of the preprocessing Temporal workflow.
	WorkflowName string `validate:"required"`
}

type WorkerConfig struct {
	// MaxConcurrentSessions limits the number of workflow sessions the
	// preprocessing worker can handle simultaneously (default: 1).
	MaxConcurrentSessions int `validate:"gte=1"`
}

func (c Configuration) Validate() error {
	var errs error

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(c); err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			switch e.Tag() {
			case "required":
				errs = errors.Join(errs, fmt.Errorf("%s: missing required value", e.Field()))
			default:
				errs = errors.Join(errs, e)
			}
		}
	}

	return errs
}

func Read(config *Configuration, configFile string) (found bool, configFileUsed string, err error) {
	v := viper.New()

	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.config/")
	v.AddConfigPath("/etc")
	v.SetConfigName("preprocessing")
	v.SetEnvPrefix("PREPROCESSING")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Defaults.
	v.SetDefault("Worker.MaxConcurrentSessions", 1)

	if configFile != "" {
		// Viper will not return a viper.ConfigFileNotFoundError error when
		// SetConfigFile() is passed a path to a file that doesn't exist, so we
		// need to check ourselves.
		if _, err := os.Stat(configFile); errors.Is(err, os.ErrNotExist) {
			return false, "", fmt.Errorf("configuration file not found: %s", configFile)
		}

		v.SetConfigFile(configFile)
	}

	if err = v.ReadInConfig(); err != nil {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			return false, "", err
		default:
			return true, "", fmt.Errorf("failed to read configuration file: %w", err)
		}
	}

	err = v.Unmarshal(config)
	if err != nil {
		return true, "", fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	if err := config.Validate(); err != nil {
		return true, "", errors.Join(
			fmt.Errorf("invalid configuration:"),
			err,
		)
	}

	return true, v.ConfigFileUsed(), nil
}
