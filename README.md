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

## Обработка метрик

### Типы метрик

Модуль правильно обрабатывает различные типы метрик k6:

#### **Gauge метрики** (перезаписываются):
- `vus`, `vus_max` - текущее количество пользователей
- `http_req_duration` - **текущее среднее** время запросов
- `http_req_waiting` - **текущее среднее** время ожидания
- `http_req_connecting` - **текущее среднее** время подключения
- `http_req_tls_handshaking` - **текущее среднее** время TLS handshake
- `http_req_blocked` - **текущее среднее** время блокировки
- `http_req_receiving` - **текущее среднее** время получения
- `http_req_sending` - **текущее среднее** время отправки
- `iteration_duration` - **текущее среднее** время итераций
- `group_duration` - **текущее среднее** время групп
- `ws_session_duration` - **текущее среднее** время WebSocket сессий
- `ws_connecting` - **текущее среднее** время подключения WebSocket
- `grpc_req_duration` - **текущее среднее** время gRPC запросов

#### **Counter метрики** (накапливаются):
- `http_reqs` - общее количество запросов
- `iterations` - общее количество итераций
- `checks` - общее количество проверок
- `data_sent` - общий объем отправленных данных
- `data_received` - общий объем полученных данных
- `ws_sessions` - общее количество WebSocket сессий
- `ws_msgs_sent` - общее количество отправленных WebSocket сообщений
- `ws_msgs_received` - общее количество полученных WebSocket сообщений

#### **Rate метрики** (накапливаются):
- `http_req_failed` - процент неудачных запросов
- `dropped_iterations` - процент пропущенных итераций

### Переименование метрик

Все метрики получают префикс `k6_`:
- `http_req_duration` → `k6_http_req_duration`
- `http_reqs` → `k6_http_reqs_total`
- `iterations` → `k6_iterations_total`

---

## Основные возможности

- ✅ **Правильная обработка метрик** - Gauge и Counter метрики обрабатываются корректно
- ✅ **Отправка метрик в Prometheus** через remote write протокол
- ✅ **Автоматическая очистка старых серий** - предотвращает утечки памяти
- ✅ **Метрики использования памяти** Go runtime
- ✅ **Логирование HTTP запросов** для отладки
- ✅ **Конфигурация** через переменные окружения и JSON
- ✅ **Метки testid и pod** для группировки метрик

---

## Примеры Prometheus запросов

### HTTP запросы:
```promql
# Среднее время запросов
avg(k6_http_req_duration{testid=~"$testid"})

# 95-й перцентиль времени запросов
histogram_quantile(0.95, rate(k6_http_req_duration_bucket{testid=~"$testid"}[5m]))

# Общее количество запросов
sum(k6_http_reqs_total{testid=~"$testid"})

# Процент неудачных запросов
rate(k6_http_req_failed{testid=~"$testid"}[5m])
```

### Пользователи:
```promql
# Текущее количество пользователей
k6_vus{testid=~"$testid"}

# Максимальное количество пользователей
k6_vus_max{testid=~"$testid"}
```

### Итерации:
```promql
# Общее количество итераций
sum(k6_iterations_total{testid=~"$testid"})

# Среднее время итераций
avg(k6_iteration_duration{testid=~"$testid"})
```

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
