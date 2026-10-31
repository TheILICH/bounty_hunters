# Space Bounty Hunters API (midterm)

REST API for space bounty hunters: a database of wanted criminals and a journal of capture attempts (hunt logs).

Same stack as the course project: Go (chi router + pgx, no ORM) and PostgreSQL 17, started with Docker Compose.

## Run it

Clone this repo, open a terminal in its folder, then:

```bash
cp .env.example .env
docker compose up --build
```

The app is ready when the logs show `listening on :8080`. Check it: http://localhost:8080/health

## Database setup / import

Nothing to import by hand. The migration `migrations/00003_bounty_hunters.sql` creates the tables
and inserts the seed data (5 criminals, 10 hunt logs). It runs automatically when the backend starts.

Tables:

- `criminals`: id, name (nullable), nickname, reward, last_known_coordinates (nullable), difficulty (1-10), armed, has_crew, escape_master
- `hunt_logs`: id, criminal_id (FK to `criminals.id`, one-to-many), success, notes, created_at (set by the database with `now()`)

Check the data in the database (works in cmd, PowerShell and bash):

```bash
docker compose exec db sh
```

then inside the container:

```bash
psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "select * from criminals" -c "select * from hunt_logs"
exit
```

To start from zero (volume deleted, seed data comes back):

```bash
docker compose down -v
docker compose up -d
```

## Endpoints

Base URL: `http://localhost:8080`. Trailing slash is optional, `/api/criminals/` and `/api/criminals` both work.

| Method | Path | What it does |
|---|---|---|
| GET | `/api/criminals/` | List, filters: `min_reward`, `max_reward`, `armed`, `has_crew`, `escape_master`, `sort=reward` or `sort=-reward` (all can be combined) |
| GET | `/api/criminals/{id}/` | One criminal |
| GET | `/api/hunts/` | List hunt logs, newest first, filter: `?criminal_id=3` |
| GET | `/api/hunts/{id}/` | One hunt log |
| POST | `/api/hunts/` | Create, body: `{"criminal_id": 3, "success": false, "notes": "..."}` → `201` |
| PATCH | `/api/hunts/{id}/` | Change only `success` and/or `notes` → `200` |
| DELETE | `/api/hunts/{id}/` | Delete → `204` |

Example: `GET /api/criminals/?min_reward=5000&armed=true&sort=-reward`

Errors are `{"error": "..."}`:

- `404`: criminal or hunt log does not exist
- `400`: bad id, bad filter value, bad JSON, missing `criminal_id` or `success`, unknown `criminal_id`, or a field that is not allowed (for example `created_at`, it is always set by the server)

## Postman

1. In Postman press `Ctrl + O` (or **Import**) and choose `postman/space-bounty-hunters.postman_collection.json`
2. The collection "Space Bounty Hunters" appears with 3 folders
3. Send the requests from top to bottom, or click the collection → **Run collection**

**Create hunt log** saves the id of the new log into the collection variable `{{huntId}}`,
so get, update and delete work on that log, no need to change ids by hand.

| Folder | Request | Expected |
|---|---|---|
| 1. Criminals | List all criminals | `200`, 5 criminals |
| | Filter + sort (min_reward, armed, -reward) | `200`, armed criminals with reward >= 5000, biggest reward first |
| | Filter reward range + sort asc | `200`, reward 2000-13000, smallest first |
| | Filter flags (has_crew, escape_master) | `200`, only criminals with crew who are escape masters |
| | Get criminal by id | `200`, criminal 3 |
| 2. Hunt logs CRUD | List all hunt logs | `200`, 10 logs, newest first |
| | List hunt logs of one criminal | `200`, only logs of criminal 3 |
| | Create hunt log | `201`, `created_at` set by the server |
| | Get hunt log by id | `200`, the log just created |
| | Update hunt log (PATCH) | `200`, `success` and `notes` changed |
| | Delete hunt log | `204` |
| | Get deleted hunt log (404) | `404`, it is really gone |
| 3. Errors | Criminal not found (404) | `404` |
| | Hunt log not found (404) | `404` |
| | Bad id (400) | `400`, id is not a number |
| | Bad sort value (400) | `400`, `sort=name` is not allowed |
| | Bad boolean filter (400) | `400`, `armed=yes` is not true/false |
| | Create without criminal_id (400) | `400`, required field missing |
| | Create for unknown criminal (400) | `400`, `criminal_id` does not exist |
| | Create with client created_at (400) | `400`, `created_at` is server-generated |
| | PATCH not allowed field (400) | `400`, only `success` and `notes` can change |
| | Invalid JSON body (400) | `400`, broken JSON |

## Where the code is

```
migrations/00003_bounty_hunters.sql   tables + seed data
internal/store/bounty_store.go        SQL queries for criminals and hunt logs
internal/api/bounty_api.go            handlers, filters, validation
internal/api/api.go                   router, /api routes are registered here
postman/                              Postman collection
```
