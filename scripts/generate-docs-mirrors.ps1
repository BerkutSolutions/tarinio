$ErrorActionPreference = 'Stop'

function Ensure-Dir([string]$path) {
  if (-not (Test-Path -LiteralPath $path)) {
    New-Item -ItemType Directory -Path $path | Out-Null
  }
}

$engArch = 'docs/eng/architecture-source'
$engOp = 'docs/eng/operator-source'
$ruArch = 'docs/ru/architecture-source'
$ruOp = 'docs/ru/operator-source'

Ensure-Dir $engArch
Ensure-Dir $engOp
Ensure-Dir $ruArch
Ensure-Dir $ruOp
Ensure-Dir 'src/theme/SearchPage'

Copy-Item 'docs/architecture/*.md' $engArch -Force
Copy-Item 'docs/operators/*.md' $engOp -Force

$operatorMap = @{
  'anti-ddos-runbook.md' = 'docs/ru/operators/anti-ddos-runbook.md'
  'letsencrypt-dns.md' = 'docs/ru/operators/letsencrypt-dns.md'
  'owasp-crs.md' = 'docs/ru/operators/owasp-crs.md'
  'runtime-filesystem-contract.md' = 'docs/ru/operators/runtime-filesystem-contract.md'
  'runtime-l4-guard.md' = 'docs/ru/operators/runtime-l4-guard.md'
  'stage-1-e2e-validation.md' = 'docs/ru/operators/stage-1-e2e-validation.md'
  'waf-tuning-guide.md' = 'docs/ru/operators/waf-tuning-guide.md'
}

foreach ($name in $operatorMap.Keys) {
  Copy-Item $operatorMap[$name] (Join-Path $ruOp $name) -Force
}

Set-Content -Path (Join-Path $ruOp 'index.md') -Encoding utf8 -Value @'
# Исходные операторские заметки TARINIO

Этот раздел переносит низкоуровневые операторские source-заметки внутрь русской wiki, чтобы вся эксплуатационная документация была доступна из одного дерева.

## Что здесь находится

- runbook по Anti-DDoS;
- заметки по Runtime L4 Guard;
- контракт runtime filesystem;
- эксплуатация OWASP CRS;
- заметки по Let's Encrypt DNS-01;
- гайд по тюнингу WAF;
- артефакты stage 1 validation.

Используй этот раздел, когда нужен более низкоуровневый инженерный контекст, чем в публичных operator guides.
'@

$archTitles = @{
  'adr-001-runtime-control-plane-split.md' = 'ADR 001. Разделение runtime и control-plane'
  'adr-002-config-compilation-model.md' = 'ADR 002. Модель компиляции конфигурации'
  'adr-003-config-rollout-and-rollback.md' = 'ADR 003. Выкатка и rollback конфигурации'
  'core-domain-model.md' = 'Базовая доменная модель'
  'easy-site-profile-contract.md' = 'Контракт easy site profile'
  'index.md' = 'Архитектурные исходные документы TARINIO'
  'logging-and-reporting-model.md' = 'Модель логирования и отчётности'
  'mvp-deployment-topology.md' = 'Топология MVP-развёртывания'
  'mvp-ui-information-architecture.md' = 'Информационная архитектура MVP UI'
  'stage-1-execution-backlog.md' = 'Execution backlog для stage 1'
  'stage-1-scope-freeze.md' = 'Фиксация scope для stage 1'
  'standalone-product-boundary.md' = 'Границы standalone-продукта'
}

$archSummaries = @{
  'adr-001-runtime-control-plane-split.md' = 'Документ объясняет, почему TARINIO разделяет enforcement runtime и control-plane, какие границы ответственности у этих подсистем и какие отказоустойчивые свойства из этого следуют.'
  'adr-002-config-compilation-model.md' = 'Документ описывает pipeline сборки конфигурации, почему runtime не должен интерпретировать сырые намерения оператора и как compile step снижает риск регрессий.'
  'adr-003-config-rollout-and-rollback.md' = 'Документ фиксирует правила безопасного rollout/rollback, ревизии, сериализацию apply и границы того, что считается атомарным применением конфигурации.'
  'core-domain-model.md' = 'Документ описывает ключевые сущности продукта, их связи и смысловые границы, чтобы изменения в API, UI и runtime оставались согласованными.'
  'easy-site-profile-contract.md' = 'Документ раскрывает контракт быстрых профилей сайта, их ожидаемое поведение, ограничения и связь с более низкоуровневыми policy-объектами.'
  'index.md' = 'Этот раздел объединяет ADR и source-level архитектурные документы, которые объясняют внутреннюю логику продукта и причины инженерных решений.'
  'logging-and-reporting-model.md' = 'Документ описывает, какие события и данные считаются источником правды для логирования, отчётности и аудита, и как они проходят по системе.'
  'mvp-deployment-topology.md' = 'Документ описывает допущения по топологии развёртывания, сетевые роли компонентов и границы ответственности инфраструктуры вокруг TARINIO.'
  'mvp-ui-information-architecture.md' = 'Документ объясняет структуру UI, группировку сценариев оператора и причины, почему навигация и сущности организованы именно так.'
  'stage-1-execution-backlog.md' = 'Документ перечисляет работы и инженерные приоритеты stage 1, полезные для понимания продуктовой эволюции и остаточных рисков.'
  'stage-1-scope-freeze.md' = 'Документ фиксирует, что вошло и что сознательно не вошло в stage 1, чтобы при обсуждении roadmap не терять исходные ограничения.'
  'standalone-product-boundary.md' = 'Документ описывает границы TARINIO как самостоятельного продукта: что входит в его ответственность, а что остаётся внешней инфраструктурой или интеграциями.'
}

foreach ($file in Get-ChildItem 'docs/architecture' -File -Filter '*.md') {
  $name = $file.Name
  $target = Join-Path $ruArch $name
  $slug = [IO.Path]::GetFileNameWithoutExtension($name)
  $title = $archTitles[$name]
  $summary = $archSummaries[$name]
  if ($name -eq 'index.md') {
    $content = @"
# $title

Этот раздел переносит архитектурные source-документы в русскую wiki, чтобы они были доступны из того же дерева, что и deploy, HA, troubleshooting и operators.

## Что здесь находится

- ADR по границам runtime/control-plane, compile model и rollback;
- доменная модель и продуктовые границы;
- topology, UI IA, logging/reporting model;
- backlog и scope freeze для stage 1.

Если тебе нужен полный исходный текст архитектурного документа, английская зеркальная копия доступна в [English wiki](/en/architecture-source/).
"@
  } else {
    $content = @"
# $title

$summary

## Русская копия раздела

Этот документ добавлен в русскую wiki как зеркальная точка входа для архитектурного раздела. Подробный исходный текст и полная формулировка ADR доступны в английской версии:

- [Открыть английский документ](/en/architecture-source/$slug/)

## Как использовать этот раздел

- Используй русскую страницу как навигационную и контекстную точку входа.
- Если нужна исходная формулировка решения, смотри английский оригинал.
- При обновлении архитектурного source of truth сначала синхронизируется английская версия, затем русская зеркальная страница.
"@
  }
  Set-Content -Path $target -Encoding utf8 -Value $content
}

Set-Content -Path (Join-Path $engArch '_category_.json') -Encoding utf8 -Value '{"label":"Architecture Source","position":70,"collapsed":true}'
Set-Content -Path (Join-Path $ruArch '_category_.json') -Encoding utf8 -Value '{"label":"Архитектурные документы","position":70,"collapsed":true}'
Set-Content -Path (Join-Path $engOp '_category_.json') -Encoding utf8 -Value '{"label":"Operator Source Notes","position":80,"collapsed":true}'
Set-Content -Path (Join-Path $ruOp '_category_.json') -Encoding utf8 -Value '{"label":"Исходные операторские заметки","position":80,"collapsed":true}'

Set-Content -Path 'src/theme/SearchPage/index.js' -Encoding utf8 -Value @'
import SearchPage from '@easyops-cn/docusaurus-search-local/dist/client/client/theme/SearchPage';

export default SearchPage;
'@
