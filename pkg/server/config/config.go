package config

import (
	"fmt"
	"os"
	"reflect"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

// HTTPConfig holds Draupnir's HTTP configuration
type HTTPConfig struct {
	SecureListenAddress   string `toml:"listen_address" required:"false"`
	InsecureListenAddress string `toml:"insecure_listen_address" required:"false"`
	TLSCertificatePath    string `toml:"tls_certificate" required:"false"`
	TLSPrivateKeyPath     string `toml:"tls_private_key" required:"false"`
}

// OAuthConfig holds Draupnir's OAuth configuration
type OAuthConfig struct {
	RedirectURL  string `toml:"redirect_url"`
	ClientID     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
}

// Config holds all Draupnir configuration
type Config struct {
	DatabaseURL            string      `toml:"database_url"`
	DataPath               string      `toml:"data_path"`
	Environment            string      `toml:"environment"`
	SharedSecret           string      `toml:"shared_secret"`
	TrustedUserEmailDomain string      `toml:"trusted_user_email_domain"`
	PublicHostname         string      `toml:"public_hostname"`
	SentryDsn              string      `toml:"sentry_dsn" required:"false"`
	MinInstancePort        uint16      `toml:"min_instance_port"`
	MaxInstancePort        uint16      `toml:"max_instance_port"`
	HTTPConfig             HTTPConfig  `toml:"http"`
	OAuthConfig            OAuthConfig `toml:"oauth"`
	CleanInterval          string      `toml:"clean_interval"`
	EnableWhitelisting     bool        `toml:"enable_ip_whitelisting" required:"false"`
	WhitelisterInterval    string      `toml:"whitelist_reconcile_interval"`
	TrustedProxyCIDRs      []string    `toml:"trusted_proxy_cidrs" required:"false"`
	UseXForwardedFor       bool        `toml:"use_x_forwarded_for" required:"false"`
}

// Load parses and validates the server config file located at `path`
func Load(path string) (Config, error) {
	var config Config
	file, err := os.Open(path)
	if err != nil {
		return config, errors.Wrap(err, fmt.Sprintf("No configuration file found at %s", path))
	}

	_, err = toml.DecodeReader(file, &config)
	if err != nil {
		return config, errors.Wrap(err, "Could not parse configuration file")
	}

	err = validateConfig(config)
	if err != nil {
		return config, errors.Wrap(err, "Invalid configuration")
	}

	return config, nil
}

func validateConfig(cfg Config) error {
	cfgValue := reflect.ValueOf(&cfg).Elem()
	cfgType := reflect.TypeOf(cfg)
	emptyFields := emptyConfigFields(cfgValue, cfgType)
	if len(emptyFields) > 0 {
		return fmt.Errorf("Missing required fields: %v", emptyFields)
	}
	return nil
}

func emptyConfigFields(val reflect.Value, ty reflect.Type) []string {
	emptyFields := []string{}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		tag := ty.Field(i).Tag
		empty := reflect.Zero(field.Type())
		if tag.Get("required") == "false" {
			continue
		}
		if reflect.DeepEqual(field.Interface(), empty.Interface()) {
			emptyFields = append(emptyFields, tag.Get("toml"))
		}
		if field.Type().Kind() == reflect.Struct {
			emptySubFields := emptyConfigFields(field, ty.Field(i).Type)
			for i := 0; i < len(emptySubFields); i++ {
				emptyFields = append(emptyFields, fmt.Sprintf("%s.%s", tag.Get("toml"), emptySubFields[i]))
			}
		}
	}

	return emptyFields
}
