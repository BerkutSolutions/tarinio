## [3.0.6] - 26.04.2026

### Изолированный k8s-ingress стенд для подключения внешних сайтов
- Добавлен новый smoke-сценарий `scripts/smoke-k8s-ingress-waf.ps1` для изолированной проверки интеграции `WAF -> Kubernetes Ingress` с тремя хостами (`example.com`, `domen.example.com`, `ex.example.com`).
- В сценарий встроен preflight-подход изоляции: случайный `compose project`, случайные container/network/volume имена, случайные host-порты runtime, чтобы не затрагивать существующий `default`/preprod-контур.
- Добавлена обязательная автоочистка после прогона: `docker compose down -v --remove-orphans` (с возможностью оставить стенд через `-NoCleanup`).
- Вынесен отдельный профиль `k8s-lab` вне `default`: `deploy/compose/k8s-lab/docker-compose.yml`.
- Добавлены k8s-манифесты тестовых приложений и ingress-правил в `deploy/compose/k8s-lab/manifests/lab-apps.yaml`.
- Добавлена документация новой связки в разделе "Подключение" (RU/EN):
  - `docs/ru/integration-docs/k8s-ingress-waf-lab.md`
  - `docs/eng/integration-docs/k8s-ingress-waf-lab.md`
