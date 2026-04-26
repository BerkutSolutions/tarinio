# K8s Lab: WAF + Ingress + 3 виртуальных хоста

Этот контур проверяет практическую интеграцию:

- TARINIO WAF принимает клиентский трафик.
- Kubernetes (`k3s`) поднимается в отдельном Docker-контейнере.
- Ingress в Kubernetes маршрутизирует запросы по `Host` на 3 разных приложения.
- WAF направляет трафик в Kubernetes через site/upstream.

Контур полностью изолирован и не требует локально установленного Kubernetes.

## Зачем нужен отдельный профиль

- Он отделён от `default`, потому что `default` не включает Kubernetes.
- На каждый запуск создаются случайные имена проекта/ресурсов, чтобы не конфликтовать с preprod и локальными стендами.
- По умолчанию после теста выполняется полная очистка контейнеров/сетей/томов.

## Что разворачивается

1. Базовый стек WAF из `deploy/compose/default/docker-compose.yml`
2. Kubernetes overlay из `deploy/compose/k8s-lab/docker-compose.yml`
3. Kubernetes-манифест приложений и ingress из `deploy/compose/k8s-lab/manifests/lab-apps.yaml`

Хосты в lab-сценарии:

- `example.com` -> `site-a`
- `domen.example.com` -> `site-b`
- `ex.example.com` -> `site-c`

Каждое приложение отдаёт уникальную строку-маркер, чтобы легко проверить корректную маршрутизацию.

## Ключевые детали реализации

- Локальный `kubectl` не нужен.
- Все команды к кластеру выполняются внутри контейнера `k3s` через `docker exec ...`.
- Скрипт автоматически определяет рабочий префикс (`kubectl` или `k3s kubectl`) и использует его для всех команд к кластеру.
- Upstream в lab настроен как `http://k3s:80` для стабильного smoke-прогона.
- Ревизии применяются последовательно: 3 ревизии, по одной на каждый сайт.

## Smoke-скрипт

Скрипт: `scripts/smoke-k8s-ingress-waf.ps1`

Что делает скрипт:

1. Генерирует случайный `smoke id`.
2. Подбирает случайные свободные HTTP/HTTPS порты runtime.
3. Генерирует per-run `.env` и compose override со случайными именами.
4. Выполняет `docker compose up -d --build` для base + `k8s-lab`.
5. Ждёт готовности control-plane.
6. При необходимости выполняет bootstrap admin.
7. Ждёт готовности `k3s` (через автоопределённый kubectl-префикс).
8. Применяет манифест (через автоопределённый kubectl-префикс).
9. Ждёт rollout `site-a`, `site-b`, `site-c`.
10. Создаёт/обновляет в WAF site/upstream для каждого хоста.
11. После каждого хоста выполняет compile+apply (всего 3 ревизии).
12. Проверяет доступность через runtime с явным `Host` header.
13. Сохраняет JSON-отчёт.
14. Выполняет очистку (если не указан `-NoCleanup`).

## Предусловия

- Docker Desktop запущен.
- Текущая shell-сессия видит команду `docker`.
- На хосте достаточно ресурсов.

Рекомендуемый минимум:

- 6 vCPU
- 8 GB RAM
- 20 GB свободного места

## Запуск

Из корня репозитория:

```powershell
./scripts/smoke-k8s-ingress-waf.ps1
```

Запуск без автоочистки (для ручной диагностики):

```powershell
./scripts/smoke-k8s-ingress-waf.ps1 -NoCleanup
```

## Ожидаемый результат

Скрипт печатает JSON и сохраняет файл:

- `.work/k8s-ingress-waf-smoke/result-<smoke-id>.json`

Критерии успешного прогона:

- `passed: true`
- 3 успешные проверки `checks`
- 3 id ревизий в `revisions`

Ожидаемые маркеры ответа:

- `K8S-LAB SITE A: example.com`
- `K8S-LAB SITE B: domen.example.com`
- `K8S-LAB SITE C: ex.example.com`

## Ручная проверка

Используйте `runtime_http_port` из result-файла:

```powershell
curl.exe -H "Host: example.com" http://127.0.0.1:<runtime_http_port>/
curl.exe -H "Host: domen.example.com" http://127.0.0.1:<runtime_http_port>/
curl.exe -H "Host: ex.example.com" http://127.0.0.1:<runtime_http_port>/
```

Каждый запрос должен вернуть свой маркер.

## Диагностика

1. Проверить Docker:
   - `docker version`
2. Проверить состояние compose:
   - `docker compose -p <project> ... ps`
3. Проверить `k3s` внутри контейнера:
   - `docker exec <k3s-container> sh -lc "kubectl get nodes"` (или `k3s kubectl get nodes`, если в вашем образе используется этот вариант)
4. Проверить объекты ingress:
   - `docker exec <k3s-container> sh -lc "kubectl -n waf-k8s-lab get pod,svc,ingress"`
5. Проверить сущности WAF:
   - `waf-cli --json sites list`
   - `waf-cli --json upstreams list`
6. Проверить ревизии:
   - compile возвращает `revision.id`
   - apply не должен быть в `failed`
7. Считать result-файл основным источником статуса pass/fail.

## Очистка

- Поведение по умолчанию: автоочистка в конце.
- Для запусков с `-NoCleanup`:

```powershell
docker compose -p <project> `
  -f deploy/compose/default/docker-compose.yml `
  -f deploy/compose/k8s-lab/docker-compose.yml `
  down -v --remove-orphans
```

## Файлы сценария

- Compose overlay: `deploy/compose/k8s-lab/docker-compose.yml`
- Kubernetes манифест: `deploy/compose/k8s-lab/manifests/lab-apps.yaml`
- Smoke-скрипт: `scripts/smoke-k8s-ingress-waf.ps1`
