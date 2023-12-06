package configuration

import (
	//	"crypto/tls"
	"fmt"
	"time"

	"github.com/caarlos0/env/v6"
)

type (
	Properties struct {
		LogLevel string `env:"LOG_LEVEL" envDefault:"DEBUG"`

		KV8s     KV8sProperties       `envPrefix:"KV8S_"`
		Auth     AuthProperties       `envPrefix:"AUTH_"`
		S3       S3Properties         `envPrefix:"S3_"`
		Server   HttpServerProperties `envPrefix:"HTTP_"`
		MLServer MLServerProperties   `envPrefix:"ML_"`
	}

	AuthProperties struct {
		Host                   string        `env:"HOST" envDefault:"https://gitlab.my.com"`
		ID                     string        `env:"ID"`
		Secret                 string        `env:"SECRET"`
		Redirect               string        `env:"REDIRECT_URL" envDefault:"http://localhost:8088/callback"`
		AccessTokenCookieName  string        `env:"ACCESS_COOKIE" envDefault:"cb_access_token"`
		RefreshTokenCookieName string        `env:"REFRESH_COOKIE" envDefault:"cb_refresh_token"`
		IDTokenCookieName      string        `env:"REFRESH_COOKIE" envDefault:"cb_id_token"`
		ReadTimeout            time.Duration `env:"READ_TIMEOUT" envDefault:"5s"`
	}

	HttpServerProperties struct {
		Name        string        `env:"NAME" envDefault:"awhs"`
		NameSpace   string        `env:"NAMESPACE" envDefault:"awhs"`
		Port        string        `env:"PORT" envDefault:"8088"`
		ReadTimeout time.Duration `env:"READ_TIMEOUT" envDefault:"5s"`
	}

	MLServerProperties struct {
		Host      string `env:"NAME" envDefault:"http://localhost:9090"`
		HostAudio string `env:"NAME_AUDIO" envDefault:"http://localhost:9090"`
		HostTS    string `env:"NAME_TS" envDefault:"http://localhost:9090"`
	}

	S3Properties struct {
		Host        string        `env:"HOST" envDefault:"https://s3.minio.com"`
		Port        string        `env:"PORT" envDefault:"9000"`
		AccessKey   string        `env:"ACCESS_KEY"`
		SecretKey   string        `env:"SECRET_KEY"`
		Bucket      string        `env:"BUCKET" envDefault:"app"`
		ReadTimeout time.Duration `env:"READ_TIMEOUT" envDefault:"5s"`
	}

	KV8sProperties struct {
		CONFIG            string   `env:"CONFIG"`
		IngressNamespaces []string `env:"INGRESS_NAMESPACES" envSeparator:"," envDefault:"ingress-nginx,istio-system"`
	}
)

func ReadProperties() *Properties {
	config := &Properties{}

	if err := env.Parse(config); err != nil {
		panic(fmt.Errorf("read config error: %w", err))
	}
	fmt.Printf("config: %+v", config)
	return config
}
