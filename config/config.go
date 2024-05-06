package config

import (
	"openkeeper/oathkeeper"
	"openkeeper/transformers/oas3"
	tomltransformer "openkeeper/transformers/toml"
)

type Config struct {
	Oathkeeper oathkeeper.Context
	OpenAPI3   map[string]struct {
		oas3.Config `koanf:",squash"`
		File        string
	}
	TOML map[string]struct {
		tomltransformer.Config `koanf:",squash"`
		File                   string
	}
}
