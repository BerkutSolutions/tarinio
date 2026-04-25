# Kubernetes Guardrails (Lab Baseline)

Этот документ закрывает задачу `Набор безопасных guardrails для lab-контура`.

## Что включено

- namespace pod security labels (`baseline`);
- network policies:
  - default deny ingress;
  - same-namespace ingress allow;
  - точечные политики для `postgres` и `opensearch`;
- security context baseline (где возможно):
  - `seccompProfile: RuntimeDefault`;
  - `allowPrivilegeEscalation: false`.

## Как применить

```powershell
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/apply-profile-guardrails.ps1
```

## Как проверить

```powershell
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/smoke-guardrails.ps1
```

## Ограничения

- это `experimental/lab` baseline, не production hardening complete;
- для `runtime` сохранены capability `NET_ADMIN` и `NET_RAW` по функциональным требованиям продукта;
- ingress/egress-модель упрощена и должна детализироваться под конкретный production perimeter.

