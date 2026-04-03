# Speak Small

Five-minute English free-talk prompts with a Go API and a TypeScript frontend.

## Requirements

- Go 1.19+
- Node.js 18+
- npm
- An existing web server for static hosting and reverse proxying `/api`

## Environment

Create a `.env` file in the project root.

```bash
cp .env.example .env
```

Set these values:

```bash
GEMINI_API_KEY=your_key_here
GEMINI_MODEL=gemini-2.5-flash-lite
```

## Build

Build the API binary:

```bash
mkdir -p bin
(cd api && go build -o ../bin/theme-server ./cmd/server)
```

Build the frontend:

```bash
cd web
npm ci
npm run build
```

The static files will be generated in `web/dist/`.

## Run

Start the API with the project `.env` loaded:

```bash
set -a
. ./.env
set +a
ADDR=127.0.0.1:18080 ./bin/theme-server
```

Run that command from the project root.

Then serve `web/dist/` as static files and reverse proxy `/api/` to `127.0.0.1:18080`.

## systemd

An example unit file is included at `systemd/speak-api.service`.

## Endpoints

- `GET /api/health`
- `GET /api/theme?category=any|daily-life|work-and-study|travel-and-places|opinions-and-ideas|relationships|culture-and-media|future-and-goals|food-and-home&energy=any|gentle|playful|stretch`
