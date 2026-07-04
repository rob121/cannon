package paths

const (
	AuthLogin  = "/auth/login"
	AuthLogout = "/auth/logout"
	AuthOAuth  = "/auth/oauth/*"
	AuthPasskeyLogin = "/auth/passkey/login/*"
)

const (
	AccountVerify       = "/account/verify/*"
	AccountVerifyResend = "/account/verify/resend"
	AccountResetRequest = "/account/reset-password"
	AccountResetSubmit  = "/account/reset-password/*"
	AccountMFAChallenge = "/account/mfa-challenge/*"
	AccountSecurity     = "/account/security"
	AccountProfile      = "/account/profile"
	AccountSecurityTOTP = "/account/security/totp/*"
	AccountSecurityPasskey = "/account/security/passkey/*"
)
