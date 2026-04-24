package easysiteprofiles

import "strings"

// ApplyServiceProfilePreset applies built-in baseline parameters for the selected profile.
// The returned object is intended as a starting template that operators can further customize.
func ApplyServiceProfilePreset(profile EasySiteProfile, preset string) EasySiteProfile {
	preset = strings.ToLower(strings.TrimSpace(preset))
	if preset == "" {
		preset = ServiceProfileBalanced
	}
	profile.FrontService.Profile = preset

	switch preset {
	case ServiceProfileStrict:
		profile.FrontService.SecurityMode = SecurityModeBlock
		profile.SecurityBehaviorAndLimits.UseBadBehavior = true
		profile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes = []int{400, 401, 403, 404, 405, 429, 444}
		profile.SecurityBehaviorAndLimits.BadBehaviorThreshold = 60
		profile.SecurityBehaviorAndLimits.BadBehaviorCountTimeSeconds = 60
		profile.SecurityBehaviorAndLimits.BadBehaviorBanTimeSeconds = 900
		profile.SecurityBehaviorAndLimits.UseLimitReq = true
		profile.SecurityBehaviorAndLimits.LimitReqURL = "/"
		profile.SecurityBehaviorAndLimits.LimitReqRate = "80r/s"
		profile.SecurityBehaviorAndLimits.UseLimitConn = true
		profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP1 = 120
		profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP2 = 220
		profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP3 = 220
		profile.SecurityAntibot.AntibotChallenge = AntibotChallengeJavascript
		profile.SecurityModSecurity.UseModSecurity = true
		profile.SecurityModSecurity.UseModSecurityCRSPlugins = true
	case ServiceProfileCompat:
		profile.FrontService.SecurityMode = SecurityModeMonitor
		profile.SecurityBehaviorAndLimits.UseBadBehavior = false
		profile.SecurityBehaviorAndLimits.UseLimitReq = false
		profile.SecurityBehaviorAndLimits.UseLimitConn = true
		profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP1 = 300
		profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP2 = 500
		profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP3 = 500
		profile.SecurityAntibot.AntibotChallenge = AntibotChallengeNo
		profile.SecurityModSecurity.UseModSecurity = true
		profile.SecurityModSecurity.UseModSecurityCRSPlugins = true
	case ServiceProfileAPI:
		profile.FrontService.SecurityMode = SecurityModeBlock
		profile.HTTPBehavior.AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"}
		profile.HTTPHeaders.UseCORS = true
		profile.HTTPHeaders.CORSAllowedOrigins = []string{"*"}
		profile.SecurityBehaviorAndLimits.UseLimitReq = true
		profile.SecurityBehaviorAndLimits.LimitReqURL = "/api/"
		profile.SecurityBehaviorAndLimits.LimitReqRate = "200r/s"
		profile.SecurityAntibot.AntibotChallenge = AntibotChallengeNo
		profile.SecurityAPIPositive.UseAPIPositiveSecurity = true
		profile.SecurityAPIPositive.EnforcementMode = APIPositiveEnforcementMonitor
		profile.SecurityAPIPositive.DefaultAction = APIPositiveDefaultActionAllow
	case ServiceProfilePublicEdge:
		profile.FrontService.SecurityMode = SecurityModeBlock
		profile.SecurityBehaviorAndLimits.UseBadBehavior = true
		profile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes = []int{400, 401, 403, 404, 405, 429, 444}
		profile.SecurityBehaviorAndLimits.BadBehaviorThreshold = 80
		profile.SecurityBehaviorAndLimits.BadBehaviorCountTimeSeconds = 60
		profile.SecurityBehaviorAndLimits.BadBehaviorBanTimeSeconds = 600
		profile.SecurityBehaviorAndLimits.UseBlacklist = true
		profile.SecurityBehaviorAndLimits.UseDNSBL = true
		profile.SecurityBehaviorAndLimits.UseLimitReq = true
		profile.SecurityBehaviorAndLimits.LimitReqURL = "/"
		profile.SecurityBehaviorAndLimits.LimitReqRate = "100r/s"
		profile.SecurityAntibot.AntibotChallenge = AntibotChallengeJavascript
		profile.SecurityModSecurity.UseModSecurity = true
		profile.SecurityModSecurity.UseModSecurityCRSPlugins = true
	default:
		profile.FrontService.Profile = ServiceProfileBalanced
		profile.FrontService.SecurityMode = SecurityModeBlock
	}

	return profile
}
