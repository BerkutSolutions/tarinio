## [1.2.4] - 22.06.2026

### Core
- Исправлен healthcheck страницы `/healthcheck`: probe `API запросов` больше не валит `API дашборда` ответом `502`, если runtime request telemetry временно недоступен или деградировал.
- runtime/nginx: убран лишний `set $waf_site_id` из site nginx template, чтобы healthcheck и прямые запросы по IP/невалидному host не создавали warning `using uninitialized "waf_site_id" variable while logging request` и не раздували runtime-логи.
