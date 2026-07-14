package protocol

const (
	CmdAuthLogonChallenge = 0x00
	CmdAuthLogonProof     = 0x01
	CmdRealmList          = 0x10
)

const (
	AuthSuccess         = 0x00
	AuthSuccessSurvey   = 0x0E
	AuthFailUnknownAcct = 0x04
)

func AuthIsSuccess(result byte) bool {
	return result == AuthSuccess || result == AuthSuccessSurvey
}

func AuthMessage(result byte) string {
	switch result {
	case AuthSuccess, AuthSuccessSurvey:
		return "Success!"
	case 0x03:
		return "Your account has been banned!"
	case 0x04:
		return "Login failed. Possibly incorrect username or password. Wait a moment and try again!"
	case 0x05:
		return "Incorrect username or password!"
	case 0x06:
		return "Your account is already online. Wait a moment and try again!"
	case 0x09:
		return "Invalid realm version for this server! Is your realm_build in config correct?"
	case 0x0A:
		return "Old realm version for this server! Is your realm_build in config correct?"
	case 0x0C:
		return "Your account has been suspended!"
	case 0x0D:
		return "Login failed! You do not have access to this server!"
	default:
		return "Failed to login to realm server! Error code."
	}
}
