## [3.0.1] - 24.04.2026

### Anti-Bot: Проверка реальных профилей трафика
- Добавлен e2e-тест `ui/tests/e2e_antibot_profiles_test.go` с 5 профилями:
  - 2 human-профиля (`human-regular`, `human-hacker`);
  - 3 bot-профиля (`bot-curl`, `bot-python-requests`, `bot-scanner-direct-verify-no-jar`).
- Тест проверяет:
  - обязательный challenge-redirect для новых клиентов без anti-bot cookie;
  - выдачу/проверку challenge cookie через `/challenge/verify`;
  - сохранение cookie в cookie jar для human-профилей;
  - отсутствие стабильного bypass у bot-профилей без persistence cookie.

### Anti-Bot: Покрытие всех режимов
- Добавлен unit-тест `compiler/internal/compiler/easy_antibot_modes_test.go`.
- Покрыты режимы `no`, `cookie`, `javascript`, `captcha`, `recaptcha`, `hcaptcha`, `turnstile`, `mcaptcha`.
- Проверяется генерация:
  - антибот-guard в `nginx/easy/<site>.conf`;
  - challenge/verify locations в `nginx/easy-locations/<site>.conf`;
  - interstitial страницы `errors/<site>/antibot.html` для interstitial-режимов.

### Anti-Bot: Поведение redirect на сервисных URL
- Для обычных сервисов отключен admin-path bypass антибота.
- Admin-path bypass оставлен только для management-сайтов (`control-plane-access`, `control-plane`, `ui`).
- Обновлены тесты `compiler/internal/compiler/easy_test.go`.

### Security Scan
- Добавлен `.gitleaks.toml` с исключением `deploy/compose/*/.env.example` из secret-scan как шаблонных example-файлов.
- Example `.env` шаблоны оставлены для документационного использования без блокировки CI secret-scan.

### Sentinel Enterprise Docs
- Расширены страницы enterprise-валидации Sentinel:
  - `docs/ru/high-availability-docs/sentinel-enterprise-validation.md`
  - `docs/eng/high-availability-docs/sentinel-enterprise-validation.md`
- Добавлены отдельные документы по anti-bot enterprise-валидации:
  - `docs/ru/high-availability-docs/antibot-enterprise-validation.md`
  - `docs/eng/high-availability-docs/antibot-enterprise-validation.md`
- Добавлены:
  - описание полного decision loop Sentinel для enterprise-review;
  - отдельный anti-bot validation документ с 5 профилями (`2 human + 3 bot`);
  - явный итоговый enterprise-вывод по тесту (готовность продукта к production-like эксплуатации).

### Версия и релизная синхронизация
- Версия продукта синхронизирована до `3.0.1` в control-plane, UI, docs, README и release-метаданных.