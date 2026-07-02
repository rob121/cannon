package paths

const (
	AuthLogin  = "/auth/login"
	AuthLogout = "/auth/logout"
	AuthOAuth  = "/auth/oauth/*"
)

const (
	AccountVerify       = "/account/verify/*"
	AccountVerifyResend = "/account/verify/resend"
	AccountResetRequest = "/account/reset-password"
	AccountResetSubmit  = "/account/reset-password/*"
)
