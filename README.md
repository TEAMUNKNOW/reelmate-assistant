# ReelMate Assistant

A lightweight Go web app for managing account *profiles*, collecting Reel URLs, generating contextual comment drafts, tracking duplicates, and sending optional Telegram notifications.

## Features

- Lightweight Go server and responsive web dashboard
- Multiple account profiles (metadata only; no passwords or session cookies)
- Context-aware fast draft generator with friendly, short, curious, neutral and Hinglish styles
- Reel URL duplicate detection and persistent local JSON storage
- 25-minute scheduler heartbeat with pause/resume
- Optional Telegram notifications for new drafts and scheduler checks
- Docker Compose deployment
- Copy draft, mark reviewed, archive

## Important boundaries

- This project does **not** log into Instagram, scrape public Reels, bypass rate limits, or automatically post comments.
- Paste the Reel URL and its caption/context manually.
- Instagram API access depends on Meta's current account eligibility, permissions, review and API capabilities. Do not assume arbitrary public Reel discovery or commenting is available.
- Do not put Instagram passwords, session cookies, access tokens, or Telegram bot tokens into source code or chat.
- Before exposing the dashboard to the public internet, put it behind authentication and HTTPS. The starter dashboard itself does not implement user authentication.

## Run locally

Requires Go 1.22+.

```bash
cp .env.example .env
go run .
```

Open http://localhost:8080.

## Run with Docker

```bash
cp .env.example .env
docker compose up --build -d
```

Open http://localhost:8080. Data is stored in the `reelmate-data` Docker volume.

## Telegram notifications

1. Create a bot through Telegram's official BotFather.
2. Get the destination chat ID using your own trusted Telegram setup.
3. Add `TELEGRAM_BOT_TOKEN` and `TELEGRAM_CHAT_ID` to `.env`.
4. Restart the app.

The bot only sends notifications; this starter does not accept Telegram commands.

## Project structure

```text
.
├── main.go
├── go.mod
├── web/index.html
├── Dockerfile
├── compose.yaml
├── .env.example
├── .gitignore
└── README.md
```

## Notes

- The 25-minute ticker is a scheduler heartbeat, not a promise to post a comment every 25 minutes.
- The built-in draft generator uses local rules, so it runs instantly and does not require a paid AI API key. For higher-quality AI drafts, add a provider behind a server-side interface and keep its API key in environment variables.
- Account profiles are organizational labels, not authenticated Instagram sessions.
