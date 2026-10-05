package config

// DefaultDotfiles returns an empty slice of DotfileSpec.
// dots does not make assumptions about user configurations or applications.
func DefaultDotfiles() []DotfileSpec {
	return []DotfileSpec{}
}
