## [2.0.11] - 23.04.2026

### Исправления безопасности
- Обновлены Go-зависимости для закрытия уязвимостей:
  - `github.com/go-acme/lego/v4` до `v4.34.0`
  - `github.com/go-jose/go-jose/v4` до `v4.1.4`
  - `golang.org/x/crypto` до `v0.50.0`
- Обновлён Go toolchain до `1.26.2` для устранения уязвимостей стандартной библиотеки.
- Обновлены npm-зависимости для закрытия уязвимостей:
  - `serialize-javascript` до `7.0.5`
  - `uuid` до `14.0.0`
- Усилен security pipeline:
  - `govulncheck` теперь завершает job с ошибкой при найденных уязвимостях
  - `npm audit` проверяет `moderate+` и завершает job с ошибкой при проблемах
  - `trivy` запускается с `MEDIUM,HIGH,CRITICAL` и `exit-code: 1`
