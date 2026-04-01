# Slack Standup Bot

A Slack bot that randomizes the speaking order for standups. Use the `/standup` slash command to start a standup, advance through speakers, and manage the order — all within Slack.

## Features

- Randomly shuffles speaking order from active channel members or a specified list
- Interactive buttons to advance or end the standup
- Add or remove participants on the fly
- Works via Socket Mode (no public URL needed) or HTTP

## Quick Start

```sh
npm install
cp .env.example .env   # then fill in your Slack tokens
npm start
```

See [SETUP.md](SETUP.md) for full Slack app creation and configuration instructions.

## Usage

| Command                        | Effect                                      |
| ------------------------------ | ------------------------------------------- |
| `/standup`                     | Start standup with active channel members   |
| `/standup @alice @bob @carol`  | Start with specific people (random order)   |
| `/standup next`                | Advance to next speaker                     |
| `/standup add @dave`           | Add someone to the remaining order          |
| `/standup remove @bob`         | Remove someone from the remaining order     |
| `/standup status`              | Re-post the current order                   |
| `/standup end`                 | End the standup early                       |

You can also use the **Next** and **End standup** buttons that appear in the message.

## Requirements

- Node.js >= 18
- A Slack workspace with permission to install apps
