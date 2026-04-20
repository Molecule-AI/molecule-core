package handlers

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// requiredEnvSchema is the subset of config.yaml we read to decide which env
// vars must be present before a container launch. It maps the YAML path
// `runtime_config.required_env: [...]` which is the same shape the workspace
// adapter's preflight reads inside the container (workspace/preflight.py).
//
// Mirroring the check server-side lets us fail fast with a readable error
// instead of letting the container crash-loop and the workspace sit in
// `provisioning` until a sweeper or the user intervenes.
type requiredEnvSchema struct {
	RuntimeConfig struct {
		RequiredEnv []string `yaml:"required_env"`
	} `yaml:"runtime_config"`
}

// missingRequiredEnv returns the list of env var names declared in the
// workspace's config.yaml under `runtime_config.required_env` that are NOT
// present (or are empty) in the assembled envVars map. Returns an empty
// slice when the config declares no requirements or when all are satisfied.
//
// A parse failure returns no missing vars — config.yaml shape is enforced by
// the in-container preflight, and the server's job here is only to catch the
// common "forgot to add the OAuth token secret" footgun, not to be a second
// config validator.
func missingRequiredEnv(configFiles map[string][]byte, envVars map[string]string) []string {
	if len(configFiles) == 0 {
		return nil
	}
	raw, ok := configFiles["config.yaml"]
	if !ok || len(raw) == 0 {
		return nil
	}
	var schema requiredEnvSchema
	if err := yaml.Unmarshal(raw, &schema); err != nil {
		return nil
	}
	if len(schema.RuntimeConfig.RequiredEnv) == 0 {
		return nil
	}
	var missing []string
	for _, name := range schema.RuntimeConfig.RequiredEnv {
		if v, ok := envVars[name]; !ok || v == "" {
			missing = append(missing, name)
		}
	}
	return missing
}

// formatMissingEnvError builds the user-facing message for a provision
// failure caused by unset required env vars. Kept stable because it's
// rendered verbatim in the canvas Events tab and Details banner.
func formatMissingEnvError(missing []string) string {
	if len(missing) == 1 {
		return fmt.Sprintf(
			"missing required env var %q — add it under Config → Env Vars (or as a Global secret) and retry",
			missing[0],
		)
	}
	return fmt.Sprintf(
		"missing required env vars %q — add them under Config → Env Vars (or as Global secrets) and retry",
		missing,
	)
}
