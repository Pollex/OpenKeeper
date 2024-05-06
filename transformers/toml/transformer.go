package tomltransformer

import (
	"fmt"
	"io"
	"openkeeper/oathkeeper"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Rule struct {
	Description    string
	Domains        []string
	Path           string
	Methods        []string
	Authenticators []string
	Authorizer     string
	Mutators       []string
	Errors         []string
}

type Root struct {
	Authenticators map[string]oathkeeper.RuleHandler
	Mutators       map[string]oathkeeper.RuleHandler
	Errors         []oathkeeper.RuleErrorHandler
	Domains        []string

	DefaultAuthenticators []string
	DefaultAuthorizer     string
	DefaultMutators       []string
	DefaultError          []string

	Rules map[string]Rule
}

type Transformer struct {
	ctx  oathkeeper.Context
	cfg  Config
	root Root
}

func FromStream(ctx oathkeeper.Context, cfg Config, value io.Reader) ([]oathkeeper.Rule, error) {
	var root Root
	if err := toml.NewDecoder(value).Decode(&root); err != nil {
		return nil, fmt.Errorf("error reading toml file: %w", err)
	}

	// Create transformer
	t := Transformer{
		ctx:  ctx,
		cfg:  cfg,
		root: root,
	}
	rules, err := t.Transform()
	if err != nil {
		return nil, fmt.Errorf("error transforming: %w", err)
	}

	return rules, nil
}

func (t *Transformer) Transform() ([]oathkeeper.Rule, error) {
	// Merge definitions into context
	for authName, auth := range t.root.Authenticators {
		t.ctx.Authenticators[authName] = auth
	}
	for mutName, mut := range t.root.Mutators {
		t.ctx.Mutators[mutName] = mut
	}
	t.ctx.Errors = append(t.ctx.Errors, t.root.Errors...)

	rules := []oathkeeper.Rule{}
	for ruleID := range t.root.Rules {
		rule, err := t.completeRule(ruleID)
		if err != nil {
			return nil, fmt.Errorf("could not transform %s: %w", ruleID, err)
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (t *Transformer) completeRule(ruleID string) (oathkeeper.Rule, error) {
	rule, ok := t.root.Rules[ruleID]
	if !ok {
		return oathkeeper.Rule{}, fmt.Errorf("no rule with id: %s", ruleID)
	}

	authenticators, err := t.determineAuthenticators(rule)
	if !ok {
		return oathkeeper.Rule{}, fmt.Errorf("could not determine authenticator: %w", err)
	}
	mutators, err := t.determineMutators(rule)
	if !ok {
		return oathkeeper.Rule{}, fmt.Errorf("could not determine mutator: %w", err)
	}
	authorizer := oathkeeper.RuleHandler{Handler: "allow"}
	errorHandlers := t.ctx.Errors

	ruleMatch := oathkeeper.RuleMatch{
		URL:     t.createRuleMatchURL(rule),
		Methods: rule.Methods,
	}

	return oathkeeper.Rule{
		ID:             ruleID,
		Description:    rule.Description,
		Authenticators: authenticators,
		Authorizer:     authorizer,
		Mutators:       mutators,
		Errors:         errorHandlers,
		Match:          ruleMatch,
	}, nil
}

func (t *Transformer) createRuleMatchURL(rule Rule) string {
	domains := t.cfg.Domains
	if rule.Domains != nil {
		domains = rule.Domains
	}
	return urlsMatching(domains) + rule.Path
}

func urlsMatching(domains []string) string {
	if len(domains) == 1 {
		return strings.TrimSuffix(domains[0], "/")
	}

	for ix := range domains {
		domains[ix] = strings.ReplaceAll(strings.TrimSuffix(domains[ix], "/"), ".", "\\.")
	}

	return "<(" + strings.Join(domains, "|") + ")>"
}

func (t *Transformer) determineAuthenticators(rule Rule) ([]oathkeeper.RuleHandler, error) {
	if rule.Authenticators != nil {
		return t.matchAuthenticators(rule.Authenticators)
	}
	return t.matchAuthenticators(t.root.DefaultAuthenticators)
}

func (t *Transformer) matchAuthenticators(names []string) ([]oathkeeper.RuleHandler, error) {
	handlers := []oathkeeper.RuleHandler{}
	// TODO: use scheme value to extract more info
	for _, name := range names {
		authenticator, ok := t.ctx.Authenticators[name]
		if !ok {
			return nil, fmt.Errorf("no Authenticator matching SecurityScheme '%s'", name)
		}
		handlers = append(handlers, authenticator)
	}

	return handlers, nil
}

func (t *Transformer) determineMutators(rule Rule) ([]oathkeeper.RuleHandler, error) {
	if rule.Mutators != nil {
		return t.matchMutators(rule.Mutators)
	}
	return t.matchMutators(t.root.DefaultMutators)
}

func (t *Transformer) matchMutators(names []string) ([]oathkeeper.RuleHandler, error) {
	handlers := []oathkeeper.RuleHandler{}
	// TODO: use scheme value to extract more info
	for _, name := range names {
		mutator, ok := t.ctx.Mutators[name]
		if !ok {
			return nil, fmt.Errorf("no Mutator matching name '%s'", name)
		}
		handlers = append(handlers, mutator)
	}

	return handlers, nil
}
