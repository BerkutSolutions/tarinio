package pipeline

import (
	"crypto/sha256"
	"encoding/hex"

	internal "waf/compiler/internal/compiler"
)

type RevisionInput = internal.RevisionInput
type SiteInput = internal.SiteInput
type UpstreamInput = internal.UpstreamInput
type TLSConfigInput = internal.TLSConfigInput
type CertificateInput = internal.CertificateInput
type WAFPolicyInput = internal.WAFPolicyInput
type WAFMode = internal.WAFMode
type AccessPolicyInput = internal.AccessPolicyInput
type RateLimitPolicyInput = internal.RateLimitPolicyInput
type CustomRateLimitRuleInput = internal.CustomRateLimitRuleInput
type APIPositiveEndpointPolicyInput = internal.APIPositiveEndpointPolicyInput
type AntibotChallengeRuleInput = internal.AntibotChallengeRuleInput
type EasyProfileInput = internal.EasyProfileInput
type ArtifactKind = internal.ArtifactKind
type ArtifactOutput = internal.ArtifactOutput
type RevisionBundle = internal.RevisionBundle
type CommandExecutor = internal.CommandExecutor
type RuntimeSyntaxRunner = internal.RuntimeSyntaxRunner
type CandidateStager = internal.CandidateStager
type AtomicActivator = internal.AtomicActivator
type ActivePointer = internal.ActivePointer
type HealthChecker = internal.HealthChecker
type ReloadHealthRunner = internal.ReloadHealthRunner
type ReloadHealthResult = internal.ReloadHealthResult
type RollbackRunner = internal.RollbackRunner
type RollbackResult = internal.RollbackResult

const (
	ArtifactKindNginxConfig = internal.ArtifactKindNginxConfig
	ArtifactKindModSecurity = internal.ArtifactKindModSecurity
	ArtifactKindCRSConfig   = internal.ArtifactKindCRSConfig
	ArtifactKindTLSRef      = internal.ArtifactKindTLSRef

	WAFModeDetection  = internal.WAFModeDetection
	WAFModePrevention = internal.WAFModePrevention
)

func RenderSiteUpstreamArtifacts(sites []SiteInput, upstreams []UpstreamInput) ([]ArtifactOutput, error) {
	return internal.RenderSiteUpstreamArtifacts(sites, upstreams)
}

func RenderTLSArtifacts(sites []SiteInput, tlsConfigs []TLSConfigInput, certificates []CertificateInput) ([]ArtifactOutput, error) {
	return internal.RenderTLSArtifacts(sites, tlsConfigs, certificates)
}

func RenderWAFArtifacts(sites []SiteInput, policies []WAFPolicyInput) ([]ArtifactOutput, error) {
	return internal.RenderWAFArtifacts(sites, policies)
}

func RenderAccessRateLimitArtifacts(sites []SiteInput, accessPolicies []AccessPolicyInput, rateLimitPolicies []RateLimitPolicyInput) ([]ArtifactOutput, error) {
	return internal.RenderAccessRateLimitArtifacts(sites, accessPolicies, rateLimitPolicies)
}

func RenderEasyArtifacts(sites []SiteInput, profiles []EasyProfileInput) ([]ArtifactOutput, error) {
	return internal.RenderEasyArtifacts(sites, profiles)
}

func RenderEasyRateLimitArtifacts(sites []SiteInput, upstreams []UpstreamInput, profiles []EasyProfileInput) ([]ArtifactOutput, error) {
	return internal.RenderEasyRateLimitArtifacts(sites, upstreams, profiles)
}

func AssembleRevisionBundle(revision RevisionInput, artifacts ...[]ArtifactOutput) (*RevisionBundle, error) {
	return internal.AssembleRevisionBundle(revision, artifacts...)
}

func ValidateRevisionBundle(bundle *RevisionBundle) error {
	return internal.ValidateRevisionBundle(bundle)
}

func LoadActivePointer(root string) (*ActivePointer, error) {
	return internal.LoadActivePointer(root)
}

func NewArtifact(path string, kind ArtifactKind, content []byte) ArtifactOutput {
	sum := sha256.Sum256(content)
	return ArtifactOutput{
		Path:     path,
		Kind:     kind,
		Content:  content,
		Checksum: hex.EncodeToString(sum[:]),
	}
}
