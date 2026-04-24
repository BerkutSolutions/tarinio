# Enterprise-проверка Sentinel

Последний полный smoke-прогон: 2026-04-24 17:31:33 +03:00.

Проверка покрывает два профиля: `default` и `high-availability-lab`.

Покрываемые сценарии:

- нормальный трафик;
- scanner paths;
- brute-force;
- XSS;
- SQL injection;
- command injection;
- single-source flood;
- distributed burst;
- high-cardinality noise.

Критерии приёмки:

1. Нормальный источник не попадает в adaptive-вывод.
2. Вредоносные паттерны дают события и объяснимые причины.
3. Есть L7-подсказки для операторской корректировки.
4. Ложные срабатывания для нормального потока равны нулю.

Артефакты прогона находятся в `.work/sentinel-smoke/`.
