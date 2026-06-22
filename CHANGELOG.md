## [1.2.4] - 22.06.2026

### Core
- Исправлен healthcheck страницы `/healthcheck`: probe `API запросов` больше не валит `API дашборда` ответом `502`, если runtime request telemetry временно недоступен или деградировал.
