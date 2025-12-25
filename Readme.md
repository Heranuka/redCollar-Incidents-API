# redCollar — Incidents API

Сервис для управления инцидентами (admin CRUD), публичной проверки координат и health-check.

---

## Содержание

<ul>
  <li><a href="#features">Возможности</a></li>
  <li><a href="#stack">Стек</a></li>
  <li><a href="#quickstart">Быстрый старт</a></li>
  <li><a href="#env">Переменные окружения</a></li>
  <li><a href="#api">API</a></li>
  <li><a href="#examples">Примеры запросов (curl)</a></li>
  <li><a href="#tests">Тесты</a></li>
  <li><a href="#coverage">Покрытие</a></li>
  <li><a href="#debug">Отладка</a></li>
</ul>

---

## <span id="features">Возможности</span>

- **Admin API**: создание / просмотр / обновление / удаление (soft-delete) инцидентов (требуется API key).
- **Public API**: проверка координат пользователя.
- **System API**: health endpoint.

---

## <span id="stack">Стек</span>

- Go + chi
- Postgres + PostGIS
- Redis
- Docker Compose

---

## <span id="quickstart">Быстрый старт (Docker Compose)</span>

1) Создай файл `.env` рядом с `docker-compose.yml` (пример ниже).
2) Подними сервисы:

<pre><code>docker compose up --build --force-recreate</code></pre>

Остановить:

<pre><code>docker compose down</code></pre>

3) Проверь здоровье сервиса:

<pre><code>curl -i http://localhost:8080/api/v1/system/health</code></pre>

---

## <span id="env">Переменные окружения (.env)</span>

Пример `.env`:

<pre><code># --- APP ---
ENV=local
HTTP_PORT=8080

# --- SECURITY ---
API_KEY=super-secret-key

# --- POSTGRES ---
POSTGRES_HOST=pg-local
POSTGRES_PORT=5432
POSTGRES_DATABASE=postgres
POSTGRES_USER=postgres
POSTGRES_PASSWORD=1
POSTGRES_SSL_MODE=disable

# --- REDIS ---
REDIS_ADDR=redis:6379
REDIS_PASSWORD=
REDIS_DB=0

# --- WEBHOOK ---
WEBHOOK_URL=http://webhook-mock:80/webhook</code></pre>

> Рекомендация: не оставляй `API_KEY` пустым — иначе можно случайно “открыть” админские ручки.

---

## <span id="api">API</span>

<b>Base URL:</b> <code>http://localhost:8080/api/v1</code>

<h3>System</h3>

- <code>GET /system/health</code>

<h3>Admin (требует API key)</h3>

Все запросы к <code>/api/v1/admin/*</code> требуют заголовок:

<pre><code>X-API-Key: &lt;API_KEY&gt;</code></pre>

Доступные ручки:

- <code>POST /admin/incidents/</code> — создать инцидент
- <code>GET /admin/incidents/</code> — список (пагинация)
- <code>GET /admin/incidents/{id}/</code> — получить по id
- <code>PUT /admin/incidents/{id}/</code> — обновить
- <code>DELETE /admin/incidents/{id}/</code> — удалить (soft delete)
- <code>GET /admin/incidents/stats</code> — статистика

<h3>Public</h3>

- <code>POST /location/check</code> — проверить координаты

---

## <span id="examples">Примеры запросов (curl)</span>

<details>
  <summary><b>System health</b></summary>

<pre><code>curl -i http://localhost:8080/api/v1/system/health</code></pre>
</details>

<details>
  <summary><b>Admin: создать инцидент</b></summary>

<pre><code>curl -i -X POST http://localhost:8080/api/v1/admin/incidents/ \
  -H "Content-Type: application/json" \
  -H "X-API-Key: super-secret-key" \
  -d '{"lat":55.75,"lng":37.61,"radius_km":1}'</code></pre>
</details>

<details>
  <summary><b>Admin: список инцидентов</b></summary>

<pre><code>curl -i "http://localhost:8080/api/v1/admin/incidents/?page=1&limit=20" \
  -H "X-API-Key: super-secret-key"</code></pre>
</details>

<details>
  <summary><b>Admin: получить / обновить / удалить</b></summary>

<pre><code># GET
curl -i "http://localhost:8080/api/v1/admin/incidents/&lt;id&gt;/" \
  -H "X-API-Key: super-secret-key"

# PUT
curl -i -X PUT "http://localhost:8080/api/v1/admin/incidents/&lt;id&gt;/" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: super-secret-key" \
  -d '{"radius_km":2,"status":"active"}'

# DELETE
curl -i -X DELETE "http://localhost:8080/api/v1/admin/incidents/&lt;id&gt;/" \
  -H "X-API-Key: super-secret-key"</code></pre>
</details>

<details>
  <summary><b>Admin: stats</b></summary>

<pre><code>curl -i "http://localhost:8080/api/v1/admin/stats?minutes=60" \
  -H "X-API-Key: super-secret-key"
</code></pre>
</details>

<details>
  <summary><b>Public: location check</b></summary>

<pre><code>curl -i -X POST http://localhost:8080/api/v1/location/check \
  -H "Content-Type: application/json" \
  -d '{"lat":55.75,"lng":37.61,"user_id":"00000000-0000-0000-0000-000000000001"}'</code></pre>
</details>

<hr/>
<div> <h2>UI</h2> <p> В проекте есть <b>UI-интерфейс</b> (веб-страница), который работает поверх этого API и позволяет удобнее 
пользоваться сервисом. </p> <p> UI доступен после запуска Docker Compose на : <code>http://localhost:8080/</code> </p> </div>
<h2><span id="tests">Тесты</span></h2>

<p>В проекте есть:</p>
<ul>
  <li>Unit-тесты для service слоя (GoMock).</li>
  <li>Unit-тесты HTTP handlers (httptest + GoMock).</li>
  <li>Integration-тесты для слоя Postgres/PostGIS (реальная БД в Docker через build tag).</li>
</ul>

<h3>Unit-тесты (быстро, без Docker)</h3>

<p>Запуск всех unit-тестов:</p>
<pre><code>go test ./... -count=1</code></pre>

<p>Запуск с подробным выводом:</p>
<pre><code>go test ./... -count=1 -v</code></pre>

<p>Запуск конкретного теста / группы тестов:</p>
<pre><code>go test ./... -run TestAdminIncidentCreate -count=1 -v</code></pre>

<h3>Integration-тесты (нужен Docker)</h3>

<p>Integration-тесты помечены build tag <code>integration</code> и поднимают зависимости в Docker.</p>

<p>Запуск:</p>
<pre><code>go test -tags=integration ./... -count=1</code></pre>

<p>Подробный вывод:</p>
<pre><code>go test -tags=integration ./... -count=1 -v</code></pre>

<p>Важно:</p>
<ul>
  <li>Для integration-тестов нужен запущенный Docker.</li>
  <li>Интеграционные DB-тесты лучше запускать без параллельности, чтобы тесты не мешали друг другу (общая база/таблицы).</li>
</ul>

<hr/>

<h2><span id="coverage">Покрытие</span></h2>

<p>Сводка покрытия:</p>
<pre><code>go test ./... -count=1 -cover</code></pre>

<p>Покрытие с профилем:</p>
<pre><code>go test ./... -count=1 -coverprofile=coverage.out
go tool cover -func=coverage.out
go tool cover -html=coverage.out</code></pre>

---

## <span id="debug">Отладка</span>

<ul>
  <li>
    Если админские запросы возвращают <code>401 Unauthorized</code>, значит сервер не принял предоставленные учётные данные (в данном проекте — API key).
  </li>
  <li>
    Убедись, что отправляешь заголовок <code>X-API-Key</code> и он совпадает с <code>API_KEY</code> внутри контейнера.
    <pre><code>docker exec -it app sh -lc 'echo "API_KEY=$API_KEY"'</code></pre>
  </li>
  <li>
    Если код поменялся, а поведение “старое”, пересобери образ и пересоздай контейнеры:
    <pre><code>docker compose up --build --force-recreate</code></pre>
  </li>
  <li>
    Если нужно увидеть список запускаемых тестов — добавь <code>-v</code> (будут строки <code>=== RUN</code>/<code>--- PASS</code>).
    <pre><code>go test ./... -count=1 -v</code></pre>
  </li>
</ul>
