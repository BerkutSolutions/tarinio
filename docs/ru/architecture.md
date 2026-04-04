# РђСЂС…РёС‚РµРєС‚СѓСЂР° TARINIO (РѕР±Р·РѕСЂ)

Р‘Р°Р·РѕРІР°СЏ РІРµСЂСЃРёСЏ РґРѕРєСѓРјРµРЅС‚Р°С†РёРё: `1.0.2`

TARINIO вЂ” standalone self-hosted WAF РЅР° Р±Р°Р·Рµ NGINX + ModSecurity + OWASP CRS.

## РљР»СЋС‡РµРІС‹Рµ РїСЂРёРЅС†РёРїС‹

- Source of truth РґР»СЏ РЅР°РјРµСЂРµРЅРёР№ РѕРїРµСЂР°С‚РѕСЂР° вЂ” control-plane (С…СЂР°РЅРёР»РёС‰Рµ + СЂРµРІРёР·РёРё).
- Runtime РЅРµ СЂРµРґР°РєС‚РёСЂСѓРµС‚СЃСЏ РІСЂСѓС‡РЅСѓСЋ: РѕРЅ РїРѕС‚СЂРµР±Р»СЏРµС‚ С‚РѕР»СЊРєРѕ Р°РєС‚РёРІРЅС‹Р№ compiled bundle.
- Р›СЋР±С‹Рµ РёР·РјРµРЅРµРЅРёСЏ РїСЂРѕС…РѕРґСЏС‚ С‡РµСЂРµР· СЂРµРІРёР·РёРё: compile в†’ validate в†’ apply в†’ rollback.

## РСЃС‚РѕС‡РЅРёРє РёСЃС‚РёРЅС‹ (Stage 0)

РџРѕР»РЅС‹Р№ РЅР°Р±РѕСЂ Р°СЂС…РёС‚РµРєС‚СѓСЂРЅС‹С… РґРѕРєСѓРјРµРЅС‚РѕРІ (РѕР±СЏР·Р°С‚РµР»СЊРЅР°СЏ РѕСЃРЅРѕРІР°):
- `docs/architecture/adr-001-runtime-control-plane-split.md`
- `docs/architecture/adr-002-config-compilation-model.md`
- `docs/architecture/adr-003-config-rollout-and-rollback.md`
- `docs/architecture/core-domain-model.md`
- `docs/architecture/mvp-deployment-topology.md`
- `docs/architecture/mvp-ui-information-architecture.md`

