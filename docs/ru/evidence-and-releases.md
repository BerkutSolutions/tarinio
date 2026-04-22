# Evidence And Releases

Эта страница относится к текущей ветке документации.

Документ описывает evidence-grade механизмы в TARINIO `2.0.6`: цепочку аудита, подписи ревизий, signed support bundle и release artifacts.

## Что реализовано

В TARINIO `2.0.6` реализованы:

- tamper-evident цепочка аудита
- подпись manifest-полей ревизии
- signed support bundle
- release artifacts с `checksums`, `SBOM`, `provenance` и подписью

## Tamper-evident audit chain

Каждое audit-событие хранит:

- `prev_hash`
- `hash`

Хэш считается по нормализованному payload события и предыдущему хэшу. Поэтому изменение записи в середине цепочки ломает верификацию.

Цепочка используется не только внутри control-plane, но и выносится в support evidence.

## Подпись ревизий

Во время compile ревизии TARINIO подписывает поля, идентифицирующие ревизию:

- ID ревизии
- версию
- время создания
- checksum
- identity компилятора
- состояние approval policy

В ревизии сохраняются:

- `signature`
- `signature_key_id`

Эти данные возвращаются в каталоге ревизий и попадают в support evidence.

## Связь change -> revision -> rollout -> result

Enterprise-контур связывает:

- создание изменения
- compile ревизии
- approval
- apply
- rollback
- итоговый результат

Эта связь видна через:

- `GET /api/revisions`
- `GET /api/audit`
- `GET /api/events`

Каталог ревизий показывает:

- кто собрал ревизию
- approval state и approvals
- последний результат apply
- время последнего apply
- последнее rollout-событие

## Signed support bundle

Support evidence выгружается через:

- `GET /api/administration/support-bundle`

Архив содержит:

- `manifest.json`
- `signature.json`
- `evidence/public_key.pem`
- `audits.json`
- `events.json`
- `jobs.json`
- `revisions.json`

### Что есть в manifest

`manifest.json` включает:

- время генерации
- ID evidence-ключа
- список файлов
- `sha256` для каждого файла
- summary по audit chain
- количество подписанных ревизий

`signature.json` подписывает `manifest.json` ключом `Ed25519`, который хранится в enterprise settings store.

### Как проверять support bundle

1. Распакуйте архив.
2. Сверьте `sha256` файлов с `manifest.json`.
3. Проверьте подпись `Ed25519` из `signature.json` с помощью `evidence/public_key.pem`.
4. Пересчитайте audit chain по `audits.json`.
5. При необходимости отдельно проверьте подписи ревизий из `revisions.json`.

## Release artifacts

Артефакты релиза генерируются через:

- `scripts/generate-release-artifacts.ps1`
- `scripts/release.ps1`
- `go run ./cmd/release-artifacts`

Файлы складываются в:

- `build/release/<version>/`

Набор включает:

- `release-manifest.json`
- `signature.json`
- `checksums.txt`
- `sbom.cdx.json`
- `provenance.json`
- `release-public-key.pem`

### Release manifest

`release-manifest.json` хранит:

- версию продукта
- git tag
- commit SHA
- docker tags
- digests исходных release inputs
- digests сгенерированных файлов
- ID signing key

### SBOM

`sbom.cdx.json` — это генерируемый `CycloneDX 1.5` bill of materials, собираемый из:

- `go.mod`
- `package.json`
- `package-lock.json`

Он нужен как машинно-читаемый dependency inventory конкретного релиза.

### Provenance

`provenance.json` фиксирует:

- версию релиза
- tag и commit
- идентификатор builder-а
- source inputs
- контекст генерации артефактов

Это лёгкая release attestation для promotion pipeline и audit review.

## Ключи подписи

Для release artifacts используется отдельный persistent `Ed25519` key в:

- `.work/release-signing/`

Для support-bundle signing используется enterprise evidence key из enterprise settings state.

Это разные контуры:

- product evidence key — операторские support и audit доказательства
- release signing key — build и promotion доказательства

## Практический порядок работы

Для preprod/prod promotion:

1. Сгенерируйте release artifacts.
2. Архивируйте `release-manifest.json`, `signature.json`, `sbom.cdx.json` и `provenance.json` вместе с change ticket.
3. Промотируйте только те артефакты, у которых проверены checksum и подпись.
4. После rollout выгружайте signed support bundle для рискованных изменений.

## Связанные документы

- [Политика поддержки и жизненного цикла](support-lifecycle.md)
- [Безопасность](security.md)
- [Enterprise Identity](enterprise-identity.md)
- [API](api.md)
