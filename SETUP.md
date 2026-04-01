# Standup Bot — Setup

## 1. Create the Slack app

1. Go to [api.slack.com/apps](https://api.slack.com/apps) → **Create New App** → **From a manifest**
2. Select your workspace, then paste the contents of `slack-app-manifest.yml` and click **Next → Create**
3. You'll land on the app's **Basic Information** page — keep this tab open for the next steps

## 2. Generate tokens

### Bot token
1. In the sidebar go to **OAuth & Permissions**
2. Click **Install to Workspace** (or **Reinstall** if already installed) and approve
3. Copy the **Bot User OAuth Token** (starts with `xoxb-`) — this is `SLACK_BOT_TOKEN`

### App-Level token (Socket Mode)
1. In the sidebar go to **Basic Information** → scroll to **App-Level Tokens**
2. Click **Generate Token and Scopes**
3. Give it a name (e.g. `socket`), add the `connections:write` scope, click **Generate**
4. Copy the token (starts with `xapp-`) — this is `SLACK_APP_TOKEN`

## 3. Configure environment

```sh
cp .env.example .env
```

Open `.env` and fill in the two tokens from above:

```
SLACK_BOT_TOKEN=xoxb-...
SLACK_APP_TOKEN=xapp-...
```

No public URL or server required — Socket Mode connects outbound to Slack.

## 4. Run

```sh
npm install
npm start        # production
npm run dev      # auto-reload on file changes (Node 18+)
```

You should see `⚡ Standup bot running (Socket Mode)` in the console. The bot is now live.

<details>
<summary>HTTP Mode (alternative — if you have a public server)</summary>

1. In `slack-app-manifest.yml` set `socket_mode_enabled: false` and add your URL to the `slash_commands` and `interactivity` entries.
2. In `.env`, remove `SLACK_APP_TOKEN` and set `SLACK_SIGNING_SECRET` (from **Basic Information** → *Signing Secret*) and optionally `PORT` (default `3000`).

</details>

---

## Usage

| Command | Effect |
|---|---|
| `/standup` | Start standup with active channel members |
| `/standup @alice @bob @carol` | Start with specific people (random order) |
| `/standup next` | Advance to next speaker |
| `/standup add @dave` | Add someone to the remaining order |
| `/standup remove @bob` | Remove someone from the remaining order |
| `/standup status` | Re-post the current order |
| `/standup end` | End the standup early |

You can also use the **Next ▶** and **End standup** buttons that appear in the message.
