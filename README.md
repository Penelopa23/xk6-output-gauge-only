# xk6-output-penelopa

## Описание

`xk6-output-penelopa` — это кастомный output-модуль для [k6](https://k6.io/), отправляющий метрики в Prometheus через remote write. Поддерживает сбор и агрегацию метрик, автоматическую очистку старых серий, а также расширяемую архитектуру.

---

## Структура проекта

```
xk6-output-penelopa/
├── register.go                    # Регистрация расширения (точка входа)
├── pkg/
│   ├── penelopa/                  # Основная логика output-модуля
│   │   ├── config.go             # Конфигурация и парсинг
│   │   ├── output.go             # Основная логика output
│   │   ├── metrics.go            # Работа с метриками
│   │   └── http.go               # HTTP утилиты
│   └── remote/                   # Prometheus remote write клиент
│       └── client.go             # HTTP клиент для remote write
├── test-script.js                # Пример скрипта для k6
├── go.mod
├── go.sum
├── README.md
└── build.sh                      # Скрипт сборки
```

---

## Сборка

```sh
# Собрать бинарник для использования с k6
$ go build -o k6-penelopa .

# Собрать через xk6
$ GOOS=linux GOARCH=amd64 ~/go/bin/xk6 build v0.48.0 \
  --output k6-penelopa \
  --with xk6-output-penelopa=/path/to/your/project
```

---

## Использование с k6

```sh
# Пример запуска теста с кастомным output
$ ./k6-penelopa run --out penelopa=http://localhost:8428/api/v1/write test-script.js
```

---

## Архитектура

Проект следует паттерну официальных xk6 расширений:

- **register.go** — точка входа, регистрация расширения для k6
- **pkg/penelopa/** — основная логика output-модуля (конфигурация, метрики, HTTP)
- **pkg/remote/** — клиент для Prometheus remote write протокола
- **ООП-структура** — код разделён по смысловым пакетам
- **Расширяемость** — легко добавлять новые типы метрик и источники конфигурации
- **Безопасность памяти** — автоматическая очистка старых серий, нет утечек памяти

---

## Основные возможности

- ✅ Отправка метрик в Prometheus через remote write
- ✅ Поддержка gauge и counter метрик
- ✅ Автоматическая очистка старых серий
- ✅ Метрики использования памяти Go runtime
- ✅ Логирование HTTP запросов
- ✅ Конфигурация через переменные окружения и JSON

---

## Пример test-script.js

```js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 5 },
    { duration: '1m', target: 5 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.1'],
  },
};

export default function () {
  const response = http.get('https://httpbin.org/get');
  check(response, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
  sleep(1);
}
```

---

## Контакты и поддержка

Если возникли вопросы или нужны доработки — пишите! 