# PCI/ASV baseline для внешнего WAF

Эта страница фиксирует baseline hardening для standalone-развёртывания TARINIO как внешнего WAF перед production-сайтом.

## Область действия

В scope:

- внешний интернет-доступный runtime edge TARINIO;
- management-граница control-plane;
- production upstream-сайт, доступный через маршрутизацию WAF.

Вне scope:

- внутренняя топология Kubernetes ingress;
- сканы внутренних сервисов, не проходящих через внешний WAF-контур.

## Матрица находок

| Класс находки сканера | Пример | Владелец | Обязательный контроль | Проверка |
| --- | --- | --- | --- | --- |
| Раскрытие сетевого стека | `TCP timestamps enabled` | Платформа / ОС | Отключить timestamps или оформить согласованное исключение с рисками | Проверка sysctl/netstack + внешний скан |
| Внешняя поверхность edge | Лишние публичные порты | Владелец deployment-профиля | Публиковать только необходимые edge-порты (`80/443`), management не выставлять наружу | `docker compose ps`, host firewall, port scan |
| TLS-протоколы и шифры | Отчёты по TLS/cipher | Владелец TLS-политики runtime | Использовать утверждённый TLS baseline и убрать слабые/устаревшие опции | TLS scan + проверка revision artifacts |
| HSTS-политика | `preload` отсутствует | Владелец runtime-политики | Поддерживать preload-ready режим там, где он допустим политикой домена | Проверка заголовков на типовых ответах |
| Устаревшие legacy-контроли | HPKP missing | Владелец compliance | Явно фиксировать статус `deprecated/not applicable` | Пакет compliance-доказательств |
| Информационные проверки | Host alive / service detection / ALPN | Security operations | Держать только ожидаемую и контролируемую экспозицию | Периодические внешние сканы + инвентаризация |

## Критерии приёмки (Definition of Done)

Релиз проходит baseline, когда одновременно выполняется:

1. Внешний edge публикует только целевые production-порты и сервисы.
2. Management UI/API не экспонируются публично в профиле по умолчанию.
3. TLS и security-headers стабильны между ревизиями.
4. Устаревшие scanner-checks закрыты явной compliance-политикой, а не «молчаливым» игнорированием.
5. Перед официальным ASV-сканом запускается повторяемый preflight.
6. В актуальном ASV-цикле нет проблемных уязвимостей, требующих remediation.

## Требования к evidence-пакету

На каждый цикл сканирования архивируются:

- полный комплект внешнего отчёта (executive/detailed/attestation при наличии);
- фактический deploy-профиль и env overrides для production;
- результаты preflight-проверок (ports/TLS/headers);
- id ревизии и apply-статус как доказательство активной политики;
- согласованные исключения с владельцем и датой пересмотра.

## Порядок preflight перед ASV-окном

Запускайте preflight с внешнего edge-хоста WAF (или с контролируемого probe-хоста с эквивалентной видимостью):

```bash
TARGET_HOST=<public-waf-hostname> \
TARGET_HTTP_PORT=80 \
TARGET_HTTPS_PORT=443 \
EXPECTED_OPEN_PORTS=80,443 \
PORT_PROBE_SET=22,80,443,8080,8443,9200 \
COMPLIANCE_POLICY_FILE=security/compliance/deprecated-controls-policy.json \
./scripts/pci-preflight-perimeter.sh
```

В evidence-пакет обязательно сохраняйте `summary.json`, `summary.txt` и все generated check-артефакты.

## Правила интерпретации ASV-результатов

Используйте следующий порядок при разборе:

1. Провалы, влияющие на экспозицию, протокольный профиль или эксплуатабельность, являются блокирующими и требуют remediation + повторного сканирования.
2. Deprecated legacy checks (например HPKP / Expect-CT) фиксируются через `security/compliance/deprecated-controls-policy.json` как policy-backed исключения, а не скрытые неисправности.
3. Информационные находки ведутся в инвентаре, но не блокируют релиз, если не показывают непредусмотренную экспозицию.

## Правила безопасного rollback при scan-регрессии

Если после изменения preflight или ASV показал регрессию:

1. Откатиться на последнюю стабильную ревизию в разделе `Ревизии`.
2. Повторно запустить preflight и подтвердить `"overall": "pass"`.
3. Зафиксировать причину отката и приложить артефакты к инциденту/изменению.
4. Возвращать исправление отдельной суженной ревизией.
