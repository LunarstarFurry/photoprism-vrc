package ollama

const (
	// EngineName is the canonical identifier for Ollama-based vision services.
	EngineName = "ollama"
	// ApiFormat identifies Ollama-compatible request and response payloads.
	ApiFormat = "ollama"
	// APIKeyEnv defines the environment variable used for Ollama API tokens.
	APIKeyEnv = "OLLAMA_API_KEY" //nolint:gosec // environment variable name, not a secret
	// APIKeyFileEnv defines the file-based fallback environment variable for Ollama API tokens.
	APIKeyFileEnv = "OLLAMA_API_KEY_FILE" //nolint:gosec // environment variable name, not a secret
	// APIKeyPlaceholder is the `${VAR}` form injected when no explicit key is provided.
	APIKeyPlaceholder = "${" + APIKeyEnv + "}"
)
