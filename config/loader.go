package config

import (
	"fmt"

	"github.com/knadh/koanf/parsers/toml/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

var k = koanf.New(".")

var defaults = Config{}

func Load(location string) (Config, error) {
	var config Config

	if err := k.Load(structs.Provider(defaults, ""), nil); err != nil {
		return config, fmt.Errorf("could not load default configuration values: %w", err)
	}

	if err := k.Load(file.Provider(location), toml.Parser()); err != nil {
		return config, fmt.Errorf("could not load config file at '%s': %w", location, err)
	}

	if err := k.Unmarshal("", &config); err != nil {
		return config, fmt.Errorf("could not unmarshal configuration: %w", err)
	}

	return config, nil
}
