package openkeeper

import (
	"encoding/json"
	"fmt"
	"openkeeper/config"
	"openkeeper/oathkeeper"
	"openkeeper/transformers/oas3"
	tomltransformer "openkeeper/transformers/toml"
	"os"
)

func Process(cfg config.Config) error {
	ctx := cfg.Oathkeeper
	allRules := []oathkeeper.Rule{}

	//
	for name, oas3Config := range cfg.OpenAPI3 {
		file, err := os.Open(oas3Config.File)
		if err != nil {
			return err
		}
		rules, err := oas3.FromStream(ctx, oas3Config.Config, file)
		if err != nil {
			return err
		}
		for _, rule := range rules {
			rule.ID = name + ":" + rule.ID
			allRules = append(allRules, rule)
		}
	}

	//
	for name, tomlConfig := range cfg.TOML {
		file, err := os.Open(tomlConfig.File)
		if err != nil {
			return err
		}
		rules, err := tomltransformer.FromStream(ctx, tomlConfig.Config, file)
		if err != nil {
			return err
		}
		for _, rule := range rules {
			rule.ID = name + ":" + rule.ID
			allRules = append(allRules, rule)
		}
	}

	data, _ := json.Marshal(allRules)
	fmt.Println(string(data))

	return nil
}
