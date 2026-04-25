# CI Lab Cycle

Этот раздел закрывает baseline-задачу по автоматическому циклу:

1. поднять lab;
2. прогнать smoke;
3. собрать артефакты;
4. снести стенд.

## Локальный запуск цикла

```powershell
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/ci/run-lab-cycle.ps1
```

## GitHub Actions шаблон

Файл-шаблон:

- `deploy/lab-k8s-terraform/ci/github-actions-lab.yml`

Его можно перенести в `.github/workflows/` после финального согласования runner-окружения.

