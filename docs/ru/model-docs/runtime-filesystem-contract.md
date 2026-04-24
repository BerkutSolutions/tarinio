# Контракт Runtime Filesystem

Статус: Stage 1 runtime contract для single-node развертывания.

## Назначение

Документ фиксирует контракт между:
- compiled revision bundles;
- staging кандидатов;
- механизмом выбора active revision;
- runtime-visible путями `/etc/waf/...`.

Документ не добавляет новую runtime-логику. Он закрепляет уже принятые ownership boundaries.

## Единый источник истины

Единственный источник истины для выбора активной ревизии:
- `<runtime-root>/active/current.json`

В runtime image это обычно:
- `/var/lib/waf/active/current.json`

Runtime обязан определять через этот файл:
- какая ревизия активна;
- какой candidate bundle должен быть смонтирован как текущий.

Runtime не должен выводить активную ревизию из:
- `/etc/waf/current`;
- symlink targets;
- имён директорий “по догадке”.

`/etc/waf/current` — только производное представление.

## Канонические пути staging и activation

Основные пути:
- staged bundle root: `<runtime-root>/candidates/<revision-id>/`
- active pointer: `<runtime-root>/active/current.json`

Пример:
- `/var/lib/waf/candidates/rev-001/`
- `/var/lib/waf/active/current.json`

`current.json` хранит:
- `revision_id`
- `candidate_path`

## Пути внутри bundle

Внутри одного staged bundle должны использоваться стабильные относительные пути:
- `manifest.json`
- `nginx/nginx.conf`
- `nginx/conf.d/*.conf`
- `nginx/access/*.conf`
- `nginx/ratelimits/*.conf`
- `nginx/sites/*.conf`
- `modsecurity/modsecurity.conf`
- `modsecurity/sites/*.conf`
- `modsecurity/crs-setup.conf`
- `modsecurity/crs-overrides/*.conf`
- `tls/*.conf`
- `errors/...`
- `ddos-model/config.json`
- `l4guard/config.json`

Эти пути должны оставаться идентичными:
- в compiler output;
- в staged candidate;
- в bundle, на который указывает `current.json`.

## Runtime-visible канонические пути

Runtime публикует выбранный bundle через:
- `/etc/waf/current`

Это symlink или эквивалентная производная проекция на `candidate_path`.

Кроме того, runtime предоставляет стабильные пути:
- `/etc/waf/nginx/nginx.conf`
- `/etc/waf/nginx/conf.d/`
- `/etc/waf/nginx/access/`
- `/etc/waf/nginx/ratelimits/`
- `/etc/waf/nginx/sites/`
- `/etc/waf/modsecurity/modsecurity.conf`
- `/etc/waf/modsecurity/sites/`
- `/etc/waf/modsecurity/crs-setup.conf`
- `/etc/waf/modsecurity/crs-overrides/`
- `/etc/waf/tls/`
- `/etc/waf/errors/`
- `/etc/waf/l4guard/config.json`
- `/etc/waf/ddos-model/config.json`

Эти runtime-visible пути должны 1:1 отражать содержимое выбранного bundle.

## Правило для OWASP CRS

Установленный OWASP CRS не хранится внутри compiled bundle.

Runtime публикует его отдельно по пути:
- `/etc/waf/modsecurity/coreruleset/`

Этот каталог:
- предоставляется runtime;
- остаётся вне ownership bundle;
- может использоваться bundle, но не должен им перезаписываться.

## Ownership boundaries

- compiler владеет только bundle-relative artifact paths;
- staging владеет материализацией в `candidates/<revision-id>/`;
- activation владеет только `active/current.json`;
- runtime владеет только производным `/etc/waf/...` view.

Runtime не владеет:
- политикой выбора ревизии;
- историей ревизий;
- revision metadata за пределами active pointer;
- control-plane state.

## Итоговое правило

Для single-node MVP:
- `active/current.json` — authoritative;
- `/etc/waf/current` — derived;
- относительные пути артефактов не должны меняться между compile, stage и runtime;
- одинаковый artifact должен иметь одинаковый относительный путь на всех этапах.
