package out

// AccessDeniedGuidance prints standardised TAPS guidance when an access error is detected.
func AccessDeniedGuidance(profile string) {
	ErrorFHighlight("  Profile used: %s", profile)
	ErrorFHighlight("  %s", "You may need to request elevated access via TAPS, or check:")
	ErrorFHighlight("    %s %s", "- You have an active SSO session:", "aws sso login --profile "+profile)
	ErrorFHighlight("    %s %s", "- The session actually has access :", "aws sts get-caller-identity --profile "+profile)
	ErrorFHighlight("    %s", "- You are in the correct eligibility group for this role/account")
	ErrorFHighlight("    %s", "- If elevated access is required, request it via TAPS before retrying")
}
