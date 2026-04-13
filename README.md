# Slack Standup Bot

A Slack bot that randomizes the speaking order for standups. Use the `/standup` slash command to start a standup, advance through speakers, and manage the order — all within Slack.

## Features

- Randomly shuffles speaking order from active channel members or a specified list
- Interactive buttons to advance or end the standup
- Add or remove participants on the fly
- Works via Socket Mode (no public URL needed) or HTTP

## Quick Start

```sh
cp .env.example .env   # then fill in your Slack tokens
go build -o standup 
./standup
```

See [SETUP.md](SETUP.md) for full Slack app creation and configuration instructions.

## Development

Install dependencies:

```sh
go mod download
```

Run tests:

```sh
go test ./...
```

## Usage

| Command                              | Effect                                      |
| ------------------------------------ | ------------------------------------------- |
| `/standup`                           | Show available commands                     |
| `/standup start`                     | Start standup with active channel members   |
| `/standup start @alice @bob @carol`  | Start with specific people (random order)   |
| `/standup next`                      | Advance to next speaker                     |
| `/standup add @dave`                 | Add someone to the remaining order          |
| `/standup remove @bob`               | Remove someone from the remaining order     |
| `/standup status`                    | Re-post the current order                   |
| `/standup end`                       | End the standup early                       |

You can also use the **Next** and **End standup** buttons that appear in the message.

## Requirements

- Go 1.22+
- A Slack workspace with permission to install apps
