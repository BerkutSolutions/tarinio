## [1.2.4] - 22.06.2026

### Core
- healthcheck `/healthcheck`: request probes no longer fail the dashboard API with `502` when runtime request telemetry is temporarily unavailable or degraded.
- runtime/nginx: removed the extra `set $waf_site_id` from the site nginx template so direct IP and invalid-host healthchecks stop generating `using uninitialized "waf_site_id" variable while logging request` warnings.
- security/go: upgraded `golang.org/x/crypto` to `v0.52.0` to close Trivy code-scanning findings and unblock `security-supply-chain`.
- runtime/nginx: management-site hostnames now proxy `/api/*` to `control-plane:8080` instead of the UI upstream, which removes `504` and upstream-timeout failures for `/api/dashboard/stats`.
- ui/dashboard: silent auto-refresh now marks dashboard stats polling as a background request, so transient `401`/`502`/`503`/`504` responses back off instead of repeatedly hammering the API from a background tab.
- dashboard: attack and blocked-attack drilldowns now stay aligned with the 24-hour metric source, so blocked requests from runtime logs no longer collapse to a tiny event-only subset in the modal details.
- ui/dashboard: widget refresh now preserves scroll position for scrollable lists instead of jumping every widget back to the top on background refresh.
- tls/acme: manual renew now sends the ACME form options to the backend, easy-site profile sync persists the ACME account email, and auto-renew defaults to enabled with profile-backed ACME settings.
