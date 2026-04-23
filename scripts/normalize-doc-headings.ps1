$ErrorActionPreference = 'Stop'

$utf8NoBom = New-Object System.Text.UTF8Encoding($false)

$headingMap = [ordered]@{
  'docs/ru/api.md' = '# API'
  'docs/ru/architecture.md' = '# Архитектура'
  'docs/ru/backups.md' = '# Бэкапы и восстановление'
  'docs/ru/benchmarks.md' = '# Набор бенчмарков'
  'docs/ru/cli-commands.md' = '# CLI-команды'
  'docs/ru/compatibility-matrix.md' = '# Матрица совместимости'
  'docs/ru/compliance-mapping.md' = '# Карта соответствия требованиям'
  'docs/ru/cookbook.md' = '# Практические сценарии эксплуатации'
  'docs/ru/deploy.md' = '# Развёртывание'
  'docs/ru/disaster-recovery.md' = '# Руководство по аварийному восстановлению'
  'docs/ru/ha.md' = '# Высокая доступность и multi-node'
  'docs/ru/hardening.md' = '# Руководство по усилению защиты'
  'docs/ru/index.md' = '# Русская wiki'
  'docs/ru/README.md' = '# Русская wiki'
  'docs/ru/limitations.md' = '# Известные ограничения и границы продукта'
  'docs/ru/navigator.md' = '# Навигатор по документации'
  'docs/ru/observability.md' = '# Наблюдаемость'
  'docs/ru/reference-architectures.md' = '# Референсные архитектуры'
  'docs/ru/release-policy.md' = '# Политика заметок о релизах'
  'docs/ru/runbook.md' = '# Runbook'
  'docs/ru/security.md' = '# Безопасность'
  'docs/ru/sizing.md' = '# Руководство по планированию ресурсов'
  'docs/ru/support-lifecycle.md' = '# Политика поддержки и жизненного цикла'
  'docs/ru/threat-model.md' = '# Модель угроз'
  'docs/ru/troubleshooting.md' = '# Руководство по устранению неполадок'
  'docs/ru/ui.md' = '# Интерфейс и сценарии работы'
  'docs/ru/upgrade.md' = '# Обновление и откат'
  'docs/ru/architecture-source/adr-001-runtime-control-plane-split.md' = '# ADR 001. Разделение runtime и control-plane'
  'docs/ru/architecture-source/adr-002-config-compilation-model.md' = '# ADR 002. Модель компиляции конфигурации'
  'docs/ru/architecture-source/adr-003-config-rollout-and-rollback.md' = '# ADR 003. Выкатка и откат конфигурации'
  'docs/ru/architecture-source/core-domain-model.md' = '# Базовая доменная модель'
  'docs/ru/architecture-source/easy-site-profile-contract.md' = '# Контракт быстрого профиля сайта'
  'docs/ru/architecture-source/index.md' = '# Архитектурные документы'
  'docs/ru/architecture-source/logging-and-reporting-model.md' = '# Модель логирования и отчётности'
  'docs/ru/architecture-source/mvp-deployment-topology.md' = '# Топология MVP-развёртывания'
  'docs/ru/architecture-source/mvp-ui-information-architecture.md' = '# Информационная архитектура MVP UI'
  'docs/ru/architecture-source/stage-1-execution-backlog.md' = '# План выполнения stage 1'
  'docs/ru/architecture-source/stage-1-scope-freeze.md' = '# Фиксация scope для stage 1'
  'docs/ru/architecture-source/standalone-product-boundary.md' = '# Границы standalone-продукта'
  'docs/ru/operator-source/index.md' = '# Исходные операторские заметки'
  'docs/ru/operator-source/anti-ddos-runbook.md' = '# Руководство оператора: Anti-DDoS'
  'docs/ru/operator-source/letsencrypt-dns.md' = '# Эксплуатация Let''s Encrypt DNS-01'
  'docs/ru/operator-source/owasp-crs.md' = '# Эксплуатация OWASP CRS'
  'docs/ru/operator-source/runtime-filesystem-contract.md' = '# Контракт файловой системы runtime'
  'docs/ru/operator-source/runtime-l4-guard.md' = '# Runtime L4 Guard'
  'docs/ru/operator-source/stage-1-e2e-validation.md' = '# Сквозная валидация stage 1'
  'docs/ru/operator-source/waf-tuning-guide.md' = '# Практическое руководство по тюнингу WAF'
  'docs/eng/api.md' = '# API'
  'docs/eng/architecture.md' = '# Architecture'
  'docs/eng/backups.md' = '# Backups and Restore'
  'docs/eng/benchmarks.md' = '# Benchmark Pack'
  'docs/eng/cli-commands.md' = '# CLI Commands'
  'docs/eng/compatibility-matrix.md' = '# Compatibility Matrix'
  'docs/eng/compliance-mapping.md' = '# Compliance Mapping'
  'docs/eng/cookbook.md' = '# Operations Cookbook'
  'docs/eng/deploy.md' = '# Deployment'
  'docs/eng/disaster-recovery.md' = '# Disaster Recovery Guide'
  'docs/eng/ha.md' = '# HA and Multi-Node'
  'docs/eng/hardening.md' = '# Hardening Guide'
  'docs/eng/index.md' = '# English Wiki'
  'docs/eng/README.md' = '# English Wiki'
  'docs/eng/limitations.md' = '# Known Limitations and Product Boundaries'
  'docs/eng/navigator.md' = '# Documentation Navigator'
  'docs/eng/observability.md' = '# Observability'
  'docs/eng/reference-architectures.md' = '# Reference Architectures'
  'docs/eng/release-policy.md' = '# Release Notes Policy'
  'docs/eng/runbook.md' = '# Runbook'
  'docs/eng/security.md' = '# Security'
  'docs/eng/sizing.md' = '# Sizing Guide'
  'docs/eng/support-lifecycle.md' = '# Support and Lifecycle Policy'
  'docs/eng/threat-model.md' = '# Threat Model'
  'docs/eng/troubleshooting.md' = '# Troubleshooting Guide'
  'docs/eng/ui.md' = '# Interface and Operator Workflows'
  'docs/eng/upgrade.md' = '# Upgrade and Rollback'
  'docs/eng/architecture-source/index.md' = '# Architecture Documents'
  'docs/eng/operator-source/index.md' = '# Operator Source Notes'
}

$lineMap = @{
  'Wiki baseline: `2.0.9`' = 'Эта страница относится к текущей ветке документации.'
  'Version baseline: `2.0.9`' = 'This page belongs to the current documentation branch.'
  'Базовая версия: `2.0.9`' = 'Эта страница относится к текущей ветке документации.'
  'Версия: `2.0.9`' = 'Эта страница относится к текущей ветке документации.'
  'Wiki baseline: `2.0.9`  ' = 'Эта страница относится к текущей ветке документации.'
  'Version baseline: `2.0.9`  ' = 'This page belongs to the current documentation branch.'
}

$ruBranchLine = 'Эта страница относится к текущей ветке документации.'
$enBranchLine = 'This page belongs to the current documentation branch.'

foreach ($path in $headingMap.Keys) {
  $fullPath = Join-Path (Get-Location) $path
  $content = [System.IO.File]::ReadAllText($fullPath, [System.Text.Encoding]::UTF8)
  $content = $content -replace "`r`n", "`n"
  $lines = New-Object System.Collections.Generic.List[string]
  foreach ($line in ($content -split "`n", 0, 'SimpleMatch')) {
    [void]$lines.Add($line)
  }

  if ($lines.Count -gt 0) {
    $lines[0] = $headingMap[$path]
  }

  for ($i = 1; $i -lt [Math]::Min($lines.Count, 6); $i++) {
    if ($lineMap.ContainsKey($lines[$i])) {
      $lines[$i] = $lineMap[$lines[$i]]
    }
  }

  for ($i = 1; $i -lt [Math]::Min($lines.Count, 4); $i++) {
    if ($lines[$i] -match '^\?{3,}') {
      if ($path -like 'docs/ru/*') {
        $lines[$i] = $ruBranchLine
      } elseif ($path -like 'docs/eng/*') {
        $lines[$i] = $enBranchLine
      }
    }
  }

  $normalized = [string]::Join("`n", $lines)
  [System.IO.File]::WriteAllText($fullPath, $normalized, $utf8NoBom)
}

[System.IO.File]::WriteAllText((Join-Path (Get-Location) 'docs/ru/operators/_category_.json'), '{"label":"Операторские руководства","position":50,"collapsed":true}', $utf8NoBom)
[System.IO.File]::WriteAllText((Join-Path (Get-Location) 'docs/ru/oss/_category_.json'), '{"label":"Документы OSS","position":90,"collapsed":true}', $utf8NoBom)
[System.IO.File]::WriteAllText((Join-Path (Get-Location) 'docs/eng/operators/_category_.json'), '{"label":"Operator Guides","position":50,"collapsed":true}', $utf8NoBom)
[System.IO.File]::WriteAllText((Join-Path (Get-Location) 'docs/eng/oss/_category_.json'), '{"label":"OSS Documents","position":90,"collapsed":true}', $utf8NoBom)
