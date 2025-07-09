package env

type Environment struct {
	Name         string
	FoundationNS string
	Namespaces   []string
}

// TBD (past POC) read from config instead
func ReadEnvironments() (error, []string, map[string]Environment) {
	environments := []Environment{
		{
			Name:         "zerotrust",
			FoundationNS: "foundation-cluster-zerotrust",
			Namespaces:   []string{"not", "used"},
		},
		// can't name a schema "default" - it's a reserved keyword in PG
		{
			Name:         "default_env",
			FoundationNS: "foundation-env-default",
		},
		{
			Name:         "qa",
			FoundationNS: "foundation-env-qa",
		},
	}

	namespaces := make([]string, len(environments))
	envMap := make(map[string]Environment)

	for i, env := range environments {
		namespaces[i] = env.FoundationNS
		envMap[env.FoundationNS] = env
	}

	return nil, namespaces, envMap

}
