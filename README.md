# Speak Small

Five-minute English free-talk prompts with a Go API, Gemini-powered topic generation, a TypeScript frontend, and nginx served through Docker.

## Run

Create a `.env` file with your Gemini settings and start the stack:

```bash
docker compose up --build
```

The included `.env` format is:

```bash
GEMINI_API_KEY=your_key_here
GEMINI_MODEL=gemini-2.5-flash-lite
```

Then open `http://localhost:8080`.

## Stack

- `api/`: Go API that generates prompts with Gemini and falls back to local prompts if needed
- `web/`: TypeScript frontend compiled to static files
- `nginx/`: reverse proxy and static hosting config

## Endpoints

- `GET /api/health`
- `GET /api/theme?category=any|daily-life|work-and-study|travel-and-places|opinions-and-ideas|relationships|culture-and-media|future-and-goals|food-and-home&energy=any|gentle|playful|stretch`
