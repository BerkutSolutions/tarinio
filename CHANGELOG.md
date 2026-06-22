## [1.2.4] - 22.06.2026

### Безопасность
- Закрыта `critical` уязвимость npm-пакета `shell-quote` обновлением до `1.8.4`.
- Закрыты `high` уязвимости в `undici` переводом дерева зависимостей на `7.28.0`.
- Закрыты `high` уязвимости в `ws` обновлением обеих веток дерева до `7.5.11` и `8.21.0`.
- Закрыты `moderate` уязвимости в `webpack-dev-server`, `http-proxy-middleware`, `launch-editor`, `joi`, `qs`.
- Закрыта `low` уязвимость в `@babel/core` обновлением до `7.29.7`.
- Обновлены `package-lock.json` и `overrides`, чтобы исправимые advisories закрывались при обычной синхронизации зависимостей.
- Добавлен `scripts/npm-security-gate.ps1`: релизный сценарий теперь делает `npm audit fix --package-lock-only`, повторный аудит и останавливает релиз при оставшихся `moderate+` production-уязвимостях.
- Docusaurus-цепочка документации переведена в `devDependencies`, а security-проверки ориентированы на production-риск, а не на инструменты локальной docs-сборки.
- GitHub Actions workflow `security-supply-chain.yml` синхронизирован с тем же правилом и теперь проверяет production npm-зависимости через `npm audit --omit=dev`.

