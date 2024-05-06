package oas3

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"openkeeper/oathkeeper"
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/samber/lo"
)

const (
	vendorExtensionAuthenticatorName  = "x-oathkeeper-authenticator"
	vendorExtensionMutatorSchemesName = "x-oathkeeper-mutatorSchemes"
	vendorExtensionMutatorsName       = "x-oathkeeper-mutators"
)

var methodPriority = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

type OASTransformer struct {
	root *openapi3.T
	ctx  oathkeeper.Context
	cfg  Config
}

func FromStream(ctx oathkeeper.Context, cfg Config, stream io.Reader) ([]oathkeeper.Rule, error) {
	root, err := openapi3.NewLoader().LoadFromIoReader(stream)
	if err != nil {
		return nil, fmt.Errorf("OAS3 transformer failed: %w", err)
	}
	return Transform(ctx, cfg, root)
}

func Transform(ctx oathkeeper.Context, cfg Config, root *openapi3.T) ([]oathkeeper.Rule, error) {
	t := OASTransformer{
		ctx:  ctx,
		cfg:  cfg,
		root: root,
	}

	t.createAuthenticatorLookup()
	t.createMutatorLookup()

	rules, err := t.Transform()
	if err != nil {
		return nil, fmt.Errorf("error transforming: %w", err)
	}

	return rules, nil
}

// createAuthenticatorLookup creates a lookup map of authenticators from different sources
// including the config.toml, but also straight from the openapi spec vendor extensions
func (t *OASTransformer) createAuthenticatorLookup() {
	for schemeName, scheme := range t.root.Components.SecuritySchemes {
		handlerNamePtr, ok := scheme.Value.Extensions[vendorExtensionAuthenticatorName]
		if !ok {
			continue
		}
		handlerName, ok := handlerNamePtr.(string)
		if !ok {
			log.Printf("SecurityScheme '%s' has '%s' set but is not a string\n", schemeName, vendorExtensionAuthenticatorName)
			continue
		}
		t.ctx.Authenticators[schemeName] = oathkeeper.RuleHandler{
			Handler: handlerName,
		}
	}
}

func (t *OASTransformer) createMutatorLookup() {
	mutatorSchemesPtr, ok := t.root.Components.Extensions[vendorExtensionMutatorSchemesName]
	if !ok {
		return
	}
	mutatorSchemes, ok := mutatorSchemesPtr.(map[string]any)
	if !ok {
		return
	}

	for schemeName, schemePtr := range mutatorSchemes {
		scheme, ok := schemePtr.(map[string]any)
		if !ok {
			continue
		}
		handlerName, ok := scheme["handler"].(string)
		if !ok {
			continue
		}
		t.ctx.Mutators[schemeName] = oathkeeper.RuleHandler{
			Handler: handlerName,
		}
	}
}

func (t *OASTransformer) Transform() ([]oathkeeper.Rule, error) {
	// for determinism, use sorted keys
	paths := t.root.Paths.InMatchingOrder()
	rules := []oathkeeper.Rule{}

	for _, path := range paths {
		pathItem := t.root.Paths.Map()[path]
		for _, method := range methodPriority {
			operation := pathItem.GetOperation(method)
			if operation == nil {
				continue
			}
			// Rule from operation
			rule := t.createRuleFromOperation(method, path, operation)
			rules = append(rules, rule)
		}
	}

	return rules, nil
}

var RegexParameterCheck = regexp.MustCompile(`({(.*)})`)

func (t *OASTransformer) createRuleFromOperation(method, path string, op *openapi3.Operation) oathkeeper.Rule {
	ruleID := op.OperationID
	if ruleID == "" {
		log.Fatalf("couldn't create Rule ID for '%s', probably missing OperationID\n", path)
	}
	match := createRuleMatcher(t.cfg.Domains, method, path, op)
	description := op.Description
	authorizer := oathkeeper.RuleHandler{Handler: "allow"}
	authenticators, err := t.determineAuthenticators(op)
	if err != nil {
		log.Fatalf("could not determine authenticators for ruleID '%s', because: %s\n", ruleID, err)
	}
	mutators, err := t.determineMutators(op)
	if err != nil {
		log.Fatalf("could not determine mutators for ruleID '%s', because: %s\n", ruleID, err)
	}
	errors := t.ctx.Errors

	return oathkeeper.Rule{
		ID:             ruleID,
		Match:          match,
		Description:    description,
		Authorizer:     authorizer,
		Authenticators: authenticators,
		Mutators:       mutators,
		Errors:         errors,
	}
}

func (t *OASTransformer) determineMutators(op *openapi3.Operation) ([]oathkeeper.RuleHandler, error) {
	localSchemes, ok := mutatorSchemeNamesFromExtensions(op.Extensions)
	if ok {
		return t.matchMutators(localSchemes)
	}
	defaultSchemes, _ := mutatorSchemeNamesFromExtensions(t.root.Extensions)
	return t.matchMutators(defaultSchemes)
}

func (t *OASTransformer) matchMutators(mutators []string) ([]oathkeeper.RuleHandler, error) {
	rules := []oathkeeper.RuleHandler{}
	for _, mutatorName := range mutators {
		ruleHandler, ok := t.ctx.Mutators[mutatorName]
		if !ok {
			return nil, fmt.Errorf("no Mutator matching MutatorScheme '%s'", mutatorName)
		}
		rules = append(rules, ruleHandler)
	}
	return rules, nil
}

func mutatorSchemeNamesFromExtensions(extension map[string]any) ([]string, bool) {
	defaultSchemesPtr, ok := extension[vendorExtensionMutatorsName]
	if !ok {
		return nil, false
	}
	defaultSchemesAnys, ok := defaultSchemesPtr.([]any)
	if !ok {
		return nil, false
	}
	schemeNames := []string{}
	for _, nameAny := range defaultSchemesAnys {
		name, ok := nameAny.(string)
		if !ok {
			continue
		}
		schemeNames = append(schemeNames, name)
	}
	return schemeNames, true
}

func (t *OASTransformer) determineAuthenticators(op *openapi3.Operation) ([]oathkeeper.RuleHandler, error) {
	localSchemes := op.Security
	if localSchemes != nil && len(*localSchemes) > 0 {
		return t.matchAuthenticators(*localSchemes)
	}
	return t.matchAuthenticators(t.root.Security)
}

func (t *OASTransformer) matchAuthenticators(requirements openapi3.SecurityRequirements) ([]oathkeeper.RuleHandler, error) {
	handlers := []oathkeeper.RuleHandler{}
	// TODO: use scheme value to extract more info
	for _, requirement := range requirements {
		schemes := lo.Keys(requirement)
		for _, schemeName := range schemes {
			authenticator, ok := t.ctx.Authenticators[schemeName]
			if !ok {
				return nil, fmt.Errorf("no Authenticator matching SecurityScheme '%s'", schemeName)
			}
			handlers = append(handlers, authenticator)
		}
	}

	return handlers, nil
}

func createRuleMatcher(domains []string, method, path string, op *openapi3.Operation) oathkeeper.RuleMatch {
	ruleURLParts := []string{}

	for _, part := range strings.Split(path, "/") {
		if matches := RegexParameterCheck.FindStringSubmatch(part); len(matches) > 0 {
			parameter := op.Parameters.GetByInAndName("path", matches[2])
			if parameter == nil {
				// AAAHHHH
				log.Fatalf("Path '%s' expected parameter '%s' but it was not specified in the spec...\n", path, matches[2])
			}
			ruleURLParts = append(ruleURLParts, createParameterMatcher(parameter))
		} else {
			ruleURLParts = append(ruleURLParts, part)
		}
	}

	return oathkeeper.RuleMatch{
		URL:     urlsMatching(domains) + strings.Join(ruleURLParts, "/"),
		Methods: []string{method},
	}
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

func createParameterMatcher(parameter *openapi3.Parameter) string {
	return "<[a-zA-Z0-9-_%]*>"
}
