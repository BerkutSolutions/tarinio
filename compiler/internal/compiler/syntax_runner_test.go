package compiler

import (
	"errors"
	"strings"
	"testing"
)

type fakeCommandExecutor struct {
	name    string
	args    []string
	workdir string
	err     error
	calls   int
}

func (f *fakeCommandExecutor) Run(name string, args []string, workdir string) error {
	f.name = name
	f.args = append([]string(nil), args...)
	f.workdir = workdir
	f.calls++
	return f.err
}

func TestRuntimeSyntaxRunner_ValidatesBundleAndRunsNginxSyntaxCheck(t *testing.T) {
	bundle, err := AssembleRevisionBundle(
		RevisionInput{
			ID:        "rev-001",
			Version:   1,
			CreatedAt: "2026-03-31T12:00:00Z",
		},
		[]ArtifactOutput{
			newArtifact("nginx/nginx.conf", ArtifactKindNginxConfig, []byte("events {}\nhttp {\n  include conf.d/*.conf;\n  include sites/*.conf;\n}\n")),
			newArtifact("nginx/conf.d/base.conf", ArtifactKindNginxConfig, []byte("proxy_http_version 1.1;\n")),
			newArtifact("nginx/sites/site-a.conf", ArtifactKindNginxConfig, []byte("server {\n  listen 80;\n  include /etc/waf/nginx/access/site-a.conf;\n  location / {\n    modsecurity on;\n    modsecurity_rules_file /etc/waf/modsecurity/sites/site-a.conf;\n    include /etc/waf/nginx/ratelimits/site-a.conf;\n    proxy_pass http://upstream;\n  }\n}\n")),
			newArtifact("nginx/access/site-a.conf", ArtifactKindNginxConfig, []byte("allow 203.0.113.0/24;\n")),
			newArtifact("nginx/ratelimits/site-a.conf", ArtifactKindNginxConfig, []byte("limit_req zone=site_a burst=5 nodelay;\n")),
			newArtifact("modsecurity/sites/site-a.conf", ArtifactKindModSecurity, []byte("SecRuleEngine On\n")),
		},
	)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	exec := &fakeCommandExecutor{}
	runner := RuntimeSyntaxRunner{
		NginxBinary: "nginx-test",
		Executor:    exec,
	}

	if err := runner.Validate(bundle); err != nil {
		t.Fatalf("expected syntax validation success, got %v", err)
	}
	if exec.calls != 1 {
		t.Fatalf("expected one executor call, got %d", exec.calls)
	}
	if exec.name != "nginx-test" {
		t.Fatalf("unexpected binary %s", exec.name)
	}
	if len(exec.args) != 5 || exec.args[0] != "-t" || exec.args[1] != "-p" || exec.args[3] != "-c" || exec.args[4] != "nginx.conf" {
		t.Fatalf("unexpected nginx args: %#v", exec.args)
	}
}

func TestRuntimeSyntaxRunner_RejectsMissingIncludedFile(t *testing.T) {
	bundle, err := AssembleRevisionBundle(
		RevisionInput{
			ID:        "rev-001",
			Version:   1,
			CreatedAt: "2026-03-31T12:00:00Z",
		},
		[]ArtifactOutput{
			newArtifact("nginx/nginx.conf", ArtifactKindNginxConfig, []byte("events {}\nhttp {\n  include conf.d/*.conf;\n}\n")),
			newArtifact("nginx/conf.d/base.conf", ArtifactKindNginxConfig, []byte("include /etc/waf/nginx/access/site-a.conf;\n")),
		},
	)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	runner := RuntimeSyntaxRunner{
		Executor: &fakeCommandExecutor{},
	}

	err = runner.Validate(bundle)
	if err == nil || !strings.Contains(err.Error(), "included file pattern has no matches") {
		t.Fatalf("expected missing include error, got %v", err)
	}
}

func TestRuntimeSyntaxRunner_AllowsWildcardIncludeWithoutMatches(t *testing.T) {
	bundle, err := AssembleRevisionBundle(
		RevisionInput{
			ID:        "rev-001",
			Version:   1,
			CreatedAt: "2026-03-31T12:00:00Z",
		},
		[]ArtifactOutput{
			newArtifact("nginx/nginx.conf", ArtifactKindNginxConfig, []byte("events {}\nhttp {\n  include sites/*.conf;\n}\n")),
		},
	)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	exec := &fakeCommandExecutor{}
	runner := RuntimeSyntaxRunner{
		Executor: exec,
	}

	if err := runner.Validate(bundle); err != nil {
		t.Fatalf("expected wildcard include without matches to pass, got %v", err)
	}
}

func TestRuntimeSyntaxRunner_PropagatesNginxSyntaxFailure(t *testing.T) {
	bundle, err := AssembleRevisionBundle(
		RevisionInput{
			ID:        "rev-001",
			Version:   1,
			CreatedAt: "2026-03-31T12:00:00Z",
		},
		[]ArtifactOutput{
			newArtifact("nginx/nginx.conf", ArtifactKindNginxConfig, []byte("events {}\nhttp {}\n")),
		},
	)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	runner := RuntimeSyntaxRunner{
		Executor: &fakeCommandExecutor{err: errors.New("exit status 1")},
	}

	err = runner.Validate(bundle)
	if err == nil || !strings.Contains(err.Error(), "nginx syntax validation failed") {
		t.Fatalf("expected nginx syntax failure, got %v", err)
	}
}
