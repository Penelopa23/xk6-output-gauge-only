# xk6-output-penelopa

Кастомный output модуль для k6, который отправляет метрики в Prometheus-совместимые системы через remote write протокол с конвертацией всех метрик в gauge-тип.

## Архитектура

Проект разделен на несколько файлов, следуя принципам ООП:

### Основные компоненты

- **`main.go`** - Регистрация модуля в k6
- **`output.go`** - Основная логика output модуля (структура Output и её методы)
- **`metrics.go`** - Работа с метриками и сериями данных
- **`config.go`** - Конфигурация модуля
- **`http.go`** - HTTP клиент и утилиты
- **`remote/client.go`** - Реализация Prometheus remote write протокола

### Структуры данных

#### Output
Основная структура модуля, которая:
- Управляет жизненным циклом output
- Обрабатывает метрики из k6
- Отправляет данные в Prometheus
- Собирает метрики памяти Go runtime

#### seriesWithMeasure
Представляет временную серию с накопленными измерениями:
- Хранит последнее значение и время
- Определяет тип метрики (gauge/counter)
- Конвертирует в Prometheus формат

## Конфигурация

### Переменные окружения

- `PENELOPA_METRICS_URL` - URL для отправки метрик
- `PENELOPA_METRICS_PUSH_INTERVAL` - Интервал отправки (по умолчанию: 5s)
- `PENELOPA_TESTID` - Идентификатор теста
- `PENELOPA_POD` - Идентификатор пода
- `PENELOPA_BATCH_SIZE` - Размер батча
- `PENELOPA_INSECURE_SKIP_TLS_VERIFY` - Пропуск проверки TLS

### Значения по умолчанию

```go
defaultServerURL    = "http://vms-victoria-metrics-single-victoria-server.metricstest:8428/api/v1/write"
defaultTimeout      = 5 * time.Second
defaultPushInterval = 5 * time.Second
defaultBatchSize    = 1000
defaultPod          = "PenelopaPod"
defaultTestId       = "PenelopaTestId"
```

## Использование

### Регистрация в k6

```go
output.RegisterExtension("penelopa", func(p output.Params) (output.Output, error) {
    return New(p)
})
```

### Запуск

```bash
k6 run --out penelopa script.js
```

## Особенности

### Конвертация метрик

Модуль автоматически определяет тип метрики на основе официальной документации k6:

**Gauge метрики** (перезаписываются):
- `vus` - текущее количество виртуальных пользователей
- `vus_max` - максимальное количество виртуальных пользователей

**Counter/Trend/Rate метрики** (накапливаются):
- `iterations` - количество итераций
- `http_reqs` - количество HTTP запросов
- `http_req_failed` - количество неудачных запросов
- `checks` - проверки
- `data_sent` - отправленные данные
- `data_received` - полученные данные
- `http_req_duration` - длительность HTTP запросов
- `http_req_waiting` - время ожидания
- `http_req_connecting` - время подключения
- `http_req_tls_handshaking` - время TLS handshake
- `http_req_blocked` - время блокировки
- `http_req_receiving` - время получения
- `http_req_sending` - время отправки
- `iteration_duration` - длительность итераций
- И другие метрики k6

### Управление памятью

Модуль включает механизм автоматической очистки старых серий для предотвращения утечек памяти:

- **Интервал очистки**: каждую минуту
- **Время жизни серий**: 10 минут для неважных метрик
- **Сохранение важных метрик**: Counter, Trend и Rate метрики сохраняются независимо от времени последнего обновления

Это обеспечивает баланс между точностью данных и эффективным использованием памяти.

### Переименование метрик

Все метрики получают префикс `k6_`:

```go
renaming := map[string]string{
    "vus":                      "k6_vus",
    "vus_max":                  "k6_vus_max",
    "iterations":               "k6_iterations_total",
    "http_reqs":                "k6_http_reqs_total",
    // ...
}
```

### Метрики памяти

Модуль автоматически добавляет метрики использования памяти Go runtime:

- `k6_mem_alloc_mb` - Текущее использование памяти
- `k6_mem_heapalloc_mb` - Использование heap
- `k6_mem_heap_sys_mb` - Системная память heap
- `k6_mem_heap_idle_mb` - Свободная память heap
- `k6_mem_heap_inuse_mb` - Используемая память heap
- `k6_mem_stack_inuse_mb` - Использование стека
- `k6_mem_stack_sys_mb` - Системная память стека
- `k6_mem_gc_cpu_fraction` - Доля CPU для GC
- `k6_mem_gc_pause_ns` - Время паузы GC
- `k6_mem_gc_count` - Количество GC
- `k6_mem_objects` - Количество объектов

## Зависимости

- `go.k6.io/k6` - Фреймворк для нагрузочного тестирования
- `github.com/castai/promwrite` - Клиент для Prometheus remote write
- `github.com/prometheus/client_golang` - Prometheus клиент
- `github.com/sirupsen/logrus` - Логирование

## Разработка

### Сборка

```bash
go build .
```

### Тестирование

```bash
go test ./...
```

## Лицензия

MIT 