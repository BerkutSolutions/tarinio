---
sidebar_position: 1
---

# Документы по развертыванию

Раздел собран как production-шпаргалка по основным контурам развертывания TARINIO v1.3.5+.

## Состав раздела

- [Развертывание через Docker](docker.md)
- [Развертывание в Kubernetes](kubernetes.md)
- [Развертывание через Terraform](terraform.md)

## Принципы раздела

- фокус на production-практике, а не на локальном lab;
- команды приведены для двух оболочек: `sh` и `PowerShell`;
- документы короткие и операционные, чтобы использовать их как runbook-шпаргалки.

## Что нового в v1.3.x

- `vault` теперь обязательный компонент стека — хранит mTLS-сертификаты;
- добавлен сервис `tarinio-sentinel` (DDoS-модель, JA3, credential stuffing);
- runtime требует `NET_ADMIN` + `NET_RAW` capabilities;
- новые тома: `waf-certificates-data`, `waf-l4-adaptive`, `waf-sentinel-state`;
- новые поля профиля сайта: `mtls_*`, `upstream_mtls_*`, `security_websocket`, `geo_time_windows`, `virtual_patches`, `blacklist_ja3`, `http_strict_parsing`, `health_check_*`.
