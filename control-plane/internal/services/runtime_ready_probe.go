package services

type HTTPRuntimeReadyProbe struct {
	Checker HTTPHealthChecker
}

func NewHTTPRuntimeReadyProbe(url string, token string) *HTTPRuntimeReadyProbe {
	return &HTTPRuntimeReadyProbe{
		Checker: HTTPHealthChecker{URL: url, Token: token},
	}
}

func (p *HTTPRuntimeReadyProbe) Probe() error {
	if p == nil {
		return nil
	}
	return p.Checker.Check(nil)
}
