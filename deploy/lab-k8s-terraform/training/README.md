# Deploy Lab Training Scenarios

Этот набор закрывает задачу `Демо-сценарии для обучения команды`.

Сценарии:

1. первый deploy
2. обновление конфигурации
3. аварийный rollback
4. teardown и повторный подъём

## Запуск

```powershell
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/training/scenario-1-first-deploy.ps1
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/training/scenario-2-config-update.ps1
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/training/scenario-3-emergency-rollback.ps1
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/training/scenario-4-teardown-redeploy.ps1
```

