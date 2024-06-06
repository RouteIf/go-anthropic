package anthropic

const (
	ModelClaudeInstant1Dot2    = "claude-instant-1.2"
	ModelClaude2Dot0           = "claude-2.0"
	ModelClaude2Dot1           = "claude-2.1"
	ModelClaude3Opus20240229   = "claude-3-opus-20240229"
	ModelClaude3Sonnet20240229 = "claude-3-sonnet-20240229"
	ModelClaude3Haiku20240307  = "claude-3-haiku-20240307"
)

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

func translateVertexModel(model string) string {
	switch model {
	case ModelClaude3Haiku20240307:
		return "claude-3-haiku@20240307"
	case ModelClaude3Opus20240229:
		return "claude-3-opus@20240229"
	case ModelClaude3Sonnet20240229:
		return "claude-3-sonnet@20240229"
	default:
		return model
	}
}
