package openai

const (
	// EngineName is the canonical identifier for OpenAI-based vision services.
	EngineName = "openai"
	// ApiFormat identifies OpenAI-compatible request and response payloads.
	ApiFormat = "openai"
	// APIKeyEnv defines the environment variable used for OpenAI API tokens.
	APIKeyEnv = "OPENAI_API_KEY" //nolint:gosec // environment variable name, not a secret
	// APIKeyFileEnv defines the file-based fallback environment variable for OpenAI API tokens.
	APIKeyFileEnv = "OPENAI_API_KEY_FILE" //nolint:gosec // environment variable name, not a secret
	// APIKeyPlaceholder is the `${VAR}` form injected when no explicit key is provided.
	APIKeyPlaceholder = "${" + APIKeyEnv + "}"
)
