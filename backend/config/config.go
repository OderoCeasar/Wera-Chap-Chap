package config

import (
	"fmt"
	"reflect"
	"time"

	"github.com/spf13/viper"
)

// DefaultMigrationURL points golang-migrate at the migrations shipped beside
// the binary. It backs MIGRATION_URL so a plain `go run .` migrates without
// any environment setup.
const DefaultMigrationURL = "file://db/migrations"

// Config stores all configuration of the application.
// The values are read by viper from a config file or environment variable.
type Config struct {
	Environment    string `mapstructure:"ENVIRONMENT"`
	Port           string `mapstructure:"PORT"`
	FrontendOrigin string `mapstructure:"FRONTEND_ORIGIN"`
	DatabaseURL    string `mapstructure:"DATABASE_URL"`
	MigrationURL   string `mapstructure:"MIGRATION_URL"`

	// JWTSecret signs both token types. AccessTokenDuration and
	// RefreshTokenDuration are durations ("15m", "336h") rather than the
	// separate minutes/hours integers the pre-restructure config used, so the
	// unit lives in the value instead of in the variable name.
	JWTSecret            string        `mapstructure:"JWT_SECRET"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`

	MpesaConsumerKey    string `mapstructure:"MPESA_CONSUMER_KEY"`
	MpesaConsumerSecret string `mapstructure:"MPESA_CONSUMER_SECRET"`
	MpesaShortcode      string `mapstructure:"MPESA_SHORTCODE"`
	MpesaPasskey        string `mapstructure:"MPESA_PASSKEY"`
	MpesaCallbackURL    string `mapstructure:"MPESA_CALLBACK_URL"`

	S3Endpoint        string `mapstructure:"S3_ENDPOINT"`
	S3Region          string `mapstructure:"S3_REGION"`
	S3Bucket          string `mapstructure:"S3_BUCKET"`
	S3AccessKeyID     string `mapstructure:"S3_ACCESS_KEY_ID"`
	S3SecretAccessKey string `mapstructure:"S3_SECRET_ACCESS_KEY"`

	// SeedDemo fills an empty database with browsable taskers and tasks. Local
	// convenience only -- see the seed package.
	SeedDemo bool `mapstructure:"SEED_DEMO"`
}

// LoadConfig reads configuration from an app.env file when present and from
// environment variables. Under docker compose there is no app.env file --
// configuration arrives entirely through the environment -- so a missing file
// is not an error.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	// viper.AutomaticEnv() alone does not make Unmarshal populate the struct
	// from the environment: Unmarshal only walks keys viper already knows about
	// (from a config file or explicit binds). Bind each field's env var so an
	// env-only setup unmarshals correctly instead of to an empty struct.
	bindEnvVars()

	viper.SetDefault("ENVIRONMENT", localEnv)
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("FRONTEND_ORIGIN", "http://localhost:5173")
	viper.SetDefault("DATABASE_URL", "postgres://wera:wera@localhost:5432/wera_chap_chap?sslmode=disable")
	viper.SetDefault("MIGRATION_URL", DefaultMigrationURL)
	viper.SetDefault("ACCESS_TOKEN_DURATION", 15*time.Minute)
	viper.SetDefault("REFRESH_TOKEN_DURATION", 14*24*time.Hour)
	viper.SetDefault("S3_REGION", "auto")

	// app.env is optional. Read it when present, but treat "file not found" as
	// a non-error so env vars still apply. Any other read error is real.
	if err = viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return config, err
		}
		err = nil
	}

	err = viper.Unmarshal(&config)
	return config, err
}

// Addr is the listen address the HTTP server binds.
func (c Config) Addr() string { return fmt.Sprintf(":%s", c.Port) }

// localEnv is the ENVIRONMENT value for local development, where a throwaway
// signing key is acceptable. Every other value is treated as an environment
// that must carry a real secret.
const localEnv = "local"

// insecureJWTSecrets are the signing keys that ship in this repo -- the
// compose default, the .env.example placeholder, and the empty string. They are
// public by definition, so a real environment using one lets anyone who read
// the source mint a token for any user id and role.
var insecureJWTSecrets = map[string]struct{}{
	"":                                  {},
	"change-me-in-dev":                  {},
	"dev-secret-change-me":              {},
	"replace-with-a-long-random-secret": {},
}

// IsLocalEnv reports whether this is the local/dev environment.
func (c Config) IsLocalEnv() bool {
	return c.Environment == localEnv
}

// HasInsecureJWTSecret reports whether JWT_SECRET is one of the public values
// that ship in this repo. main uses it to refuse to boot a non-local
// environment with a forgeable key.
func (c Config) HasInsecureJWTSecret() bool {
	_, bad := insecureJWTSecrets[c.JWTSecret]
	return bad
}

// bindEnvVars binds each Config field's `mapstructure` env var name so that
// viper.Unmarshal picks up values from the environment even when no config file
// is present.
func bindEnvVars() {
	t := reflect.TypeOf(Config{})
	for i := 0; i < t.NumField(); i++ {
		if tag := t.Field(i).Tag.Get("mapstructure"); tag != "" {
			_ = viper.BindEnv(tag)
		}
	}
}
