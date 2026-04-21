# Модель угроз

Эта страница относится к текущей ветке документации.

## Назначение

Этот документ объясняет, что именно TARINIO помогает защищать, где проходят доверительные границы и какая ответственность остаётся у оператора.

## Защищаемые активы

TARINIO в первую очередь помогает защищать:

- application ingress;
- policy-controlled request handling;
- контролируемый rollout настроек защиты;
- операторскую видимость security и rollout events.

## Основные trust boundaries

- public client -> runtime ingress;
- runtime -> upstream applications;
- operator -> control-plane;
- control-plane -> PostgreSQL / Redis / runtime API.

## Какие угрозы TARINIO помогает закрывать

- типовой web attack traffic через WAF/CRS;
- abusive request patterns и brute-force-подобные сценарии;
- враждебные L4/L7 traffic patterns через Anti-DDoS;
- опасный configuration drift через revision-based change flow.

## Shared responsibility

TARINIO не заменяет:

- безопасный application code;
- сетевую сегментацию;
- secret management;
- backup и disaster recovery;
- hardening хоста и контейнеров.
