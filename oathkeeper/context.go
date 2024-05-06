package oathkeeper

type Context struct {
	Authenticators map[string]RuleHandler
	Mutators       map[string]RuleHandler
	Errors         []RuleErrorHandler
	Rules          map[string]Rule
}

func EmptyContext() Context {
	return Context{
		Authenticators: make(map[string]RuleHandler),
		Mutators:       make(map[string]RuleHandler),
		Errors:         make([]RuleErrorHandler, 0),
		Rules:          make(map[string]Rule),
	}
}
