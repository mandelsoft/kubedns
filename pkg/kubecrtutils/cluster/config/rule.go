package config

type IdentityProvider interface {
	RequireIdentity()
}

type Personalization struct {
	Name        string
	Description string
}

type Personalizable interface {
	PersonalizedWith(name *Personalization) Rule
}

type Rule interface {
	GetConfig(opts *ConfigOptions) (*Config, error)
}
