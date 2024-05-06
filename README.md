# OpenKeeper

OpenKeeper is a tool for generating ORY Oathkeeper rules from various sources, such as OpenAPI specifications and TOML files. It provides a convenient way to define and manage access control rules for your APIs.

## Installation

```
go install github.com/pollex/openkeeper@latest
```

## Usage

To generate a ruleset, use the `generate` command with a config file:

```
openkeeper generate --config config.toml
```

## Configuration

The configuration file is a TOML file that defines the sources and settings for generating the rules. Here's an example:

```toml
# Setup the noop authenticator
[Oathkeeper.Authenticators.Noop]
Handler = "noop"

# Setup the noop mutator
[Oathkeeper.Mutators.Noop]
Handler = "noop"

[openapi3.PetStoreExample]
File = "petstore.yaml"
Domains = ["https://pet.store.com/api"]

[toml.ExtraRules]
File = "rules.toml"
Domains = ["https://auth.pet.store.com"]
```

### OpenAPI 3

To specify what handler to use per SecurityScheme, use the `x-oathkeeper-authenticator` extension field:

```yaml
components:
  securitySchemes:
    CookieSession:
      type: apiKey
      in: cookie
      name: SID
      x-oathkeeper-authenticator: cookie_session
    APIKey:
      type: http
      scheme: Bearer
      x-oathkeeper-authenticator: bearer_token
```

To specify what mutators to apply, use the `x-oathkeeper-mutator` extension field. Either on the root-level to define a default, or per operation:

```yaml
x-oathkeeper-mutators:
  - Hydrate
  - IDToken
```

The given entries for x-oathkeeper-authenticator and x-oathkeeper-mutator must be defined in the config.toml file. 

### TOML Rules

Rules can also be specified in TOML, such as the one below:

```toml
[Rules.Dashboard]
Description = "Dashboard (sub)paths"
Authenticators = ["CookieSession"]
Mutators = ["Hydrate", "IDToken"]
Path = "/dashboard<(/.+)?>"
Methods = ["GET","POST","PATCH","PUT","DELETE","OPTION"]

[Rules."InternalTraffic"]
Description = "Internal traffic should already contains JWT"
Domains = ["http://a.different.domain.for.this.one"]
Authenticators = ["Noop"]
Mutators = ["Noop"]
Path = "/<.*>"
Methods = ["GET","POST","PATCH","PUT","DELETE","OPTION"]
```

## Development

A transformer is given an `oathkeeper.Context` struct which contains information about what Oathkeeper Handlers (Authenticators, Authorizer, Mutators and Error) are available.


## License

```
The MIT License (MIT)

Copyright © 2024 Tim van Osch

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```