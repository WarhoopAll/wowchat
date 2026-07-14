# WoWChat

[![Build](https://github.com/WarhoopAll/wowchat/actions/workflows/build.yml/badge.svg)](https://github.com/WarhoopAll/wowchat/actions/workflows/build.yml)
[![Version](https://img.shields.io/github/v/release/WarhoopAll/wowchat?include_prereleases&label=version)](https://github.com/WarhoopAll/wowchat/releases)
[![Docker](https://img.shields.io/badge/docker-re3os%2Fwowchat-blue)](https://hub.docker.com/r/re3os/wowchat)

A relay that connects a **World of Warcraft** (3.3.5 / WotLK) game client chat
to a **Discord** server. PROXY SUPPORT!

It logs into a realm as a real game character, enters the world, joins
configured chat channels, and mirrors chat traffic in both directions —
WoW ↔ Discord.

---
## Features

- **Two-way relay** — WoW → Discord and Discord → WoW, over any configured chat
  type (say, guild, officer, yell, whisper, emote, system, custom channels).
- **Cyrillic channel support** — channels like `Поиск спутников` work out of the box.
- **Message formatting** — templates with `%time`, `%user`, `%message`, `%target`
  placeholders, independently per direction.
- **Item link resolution** — WoW item links (`|Hitem:...|h[name]|h`) are turned
  into clickable DB links for Discord.
- **Raid icon conversion** — WoW raid icons (star, skull, cross, …) are mapped to
  Discord emoji on the way out.
- **In-game dot-commands** — run WoW commands (`.server info`, `.tele …`) from
  Discord, with an allowlist and channel restriction.
- **Message filters** — drop messages matching regex patterns, in either direction.
- **Proxy support** — route traffic through **http / https / socks5** proxies,
  including VLESS via a local Xray/V2Ray SOCKS5 inbound. WoW and Discord traffic
  can be proxied independently.
- **Bot command isolation** — ignore Discord messages starting with configurable
  prefixes (e.g. `!`) so other bots' commands are not relayed into the game.
- **Emoji stripping** — remove Discord emoji before sending to WoW, which cannot
  render them (avoids `?` glyphs).
- **Auto-reconnect** — recovers automatically from game connection drops.
---
## Quick Start

You need a `.env` file with your settings first. Copy the
[`.env.example`](https://github.com/WarhoopAll/wowchat/blob/master/.env.example)
template and edit it:

```bash
cp .env.example .env
# then edit .env with your editor
```

Then pick one of the run options below.

### Option A — Docker Compose (recommended)

Just run this (the [`compose.yml`](https://github.com/WarhoopAll/wowchat/blob/master/compose.yml)
is already in the repo):

```bash
docker compose up -d
```

That's it — the container starts, restarts on crash, and reads `.env` from the
same folder.

### Option B — Docker run (no build needed)

```bash
docker run -d --name wowchat --restart unless-stopped \
  -v "$PWD/.env:/app/.env:ro" \
  re3os/wowchat:latest
```

The prebuilt image is pulled automatically — you never need to build anything.

### Option C — Download a binary

Grab a ready-made binary for your OS/arch from the
[Releases](https://github.com/WarhoopAll/wowchat/releases) page, then run it
from the folder that contains your `.env`:

```bash
./wowchat            # linux / mac
wowchat.exe          # windows
```

To validate your config before connecting:

```bash
./wowchat -check-config
```

Print the application version (set from the git tag at build time):

```bash
./wowchat -version
```

The version is injected at build time: release binaries and the Docker image
pick it up from the release git tag (e.g. `v1.2.3`); local/dev builds report
`dev`.

---

## Build from Source (developers)

Requires **Go 1.25+**.

```bash
# build the binary
go build -o wowchat ./cmd/wowchat

# run the tests
go test ./...
```

Stop with `Ctrl+C` (handles SIGINT/SIGTERM). On a game connection drop the bot
reconnects automatically.

---

## Configuration

All configuration is done through a `.env` file (or environment variables;
environment variables override `.env` values). See
[`.env.example`](https://github.com/WarhoopAll/wowchat/blob/master/.env.example)
for a complete template. The Docker image is built by the
[`Dockerfile`](https://github.com/WarhoopAll/wowchat/blob/master/Dockerfile).

Required fields: `WOW_ACCOUNT`, `WOW_PASSWORD`, `WOW_CHARACTER`,
`WOW_REALMLIST_HOST`, `WOW_REALM`, `WOW_VERSION`, `DISCORD_TOKEN`.
If a required value is empty, startup prints the list of missing keys and exits
with an error.

`CHAT_ROUTE_COUNT` is not itself required, but the bot has nothing to relay
until you define at least one `CHAT_ROUTE_N_*` block (see
[Chat Routes](#chat-routes-chat_route_n)).

### Discord

| Variable | Default | Purpose |
|---|---|---|
| `DISCORD_TOKEN` | _(required)_ | Discord bot token. Provide only the token itself (the `Bot ` prefix is added automatically). |
| `DISCORD_ENABLE_DOT_COMMANDS` | `true` | Allow dot-commands (`.server info`, `.tele …`) from Discord → WoW. |
| `DISCORD_DOT_COMMANDS_WHITELIST` | _—_ | Comma-separated list of allowed commands. Multi-word commands (`.server info`) are written whole — the comma does not split them. An entry ending in `*` (e.g. `tele *`) allows any command with that prefix. Empty = all dot-commands allowed. |
| `DISCORD_ENABLE_COMMANDS_CHANNELS` | _—_ | Space-separated list of Discord channel names where dot-commands are accepted. Empty = allowed in every configured channel. |
| `DISCORD_ENABLE_TAG_FAILED_NOTIFICATIONS` | `true` | Notify in Discord if a send to WoW failed. |
| `DISCORD_IGNORE_PREFIXES` | `!` | Discord message prefixes that are **not** relayed to WoW (so other bots' commands are not duplicated). Multiple prefixes comma-separated: `!,?`. |
| `DISCORD_STRIP_EMOJI` | `true` | Strip Discord emoji from Discord → WoW messages (the WoW client cannot render them and shows `?`). |
| `DISCORD_ITEM_DATABASE` | _—_ | Item database base URL (fallback). See `ITEM_DATABASE`. |

### World of Warcraft

| Variable | Default | Purpose |
|---|---|---|
| `WOW_PLATFORM` | `Mac` | Platform reported to the server: `windows` or anything else (treated as `Mac`). |
| `WOW_VERSION` | _(required)_ | Client version, e.g. `3.3.5`. Determines the build number (3.3.5 → `12340`). |
| `WOW_REALMLIST_HOST` | _(required)_ | Realm server (authserver) address, e.g. `logon.warhoop.su` or `127.0.0.1`. |
| `WOW_REALMLIST_PORT` | `3724` | Realm server port. |
| `WOW_REALM` | _(required)_ | Realm name the character logs into. |
| `WOW_ACCOUNT` | _(required)_ | Account login (only ASCII letters are upper-cased, for SRP hash stability). |
| `WOW_PASSWORD` | _(required)_ | Account password. |
| `WOW_CHARACTER` | _(required)_ | Character name used for the relay. |
| `WOW_ENABLE_SERVER_MOTD` | `false` | _Currently unused_ (kept for compatibility). |

### Chat Routes (`CHAT_ROUTE_N_*`)

Route blocks are numbered from 1 to `CHAT_ROUTE_COUNT`. Each block describes a
single relay rule.

| Variable | Purpose |
|---|---|
| `CHAT_ROUTE_COUNT` | Number of routes. |
| `CHAT_ROUTE_N_DIRECTION` | Direction: `wow_to_discord`, `discord_to_wow`, or `both`. |
| `CHAT_ROUTE_N_WOW_TYPE` | WoW chat type: `system`, `say`, `guild`, `officer`, `yell`, `emote`, `whisper`, `channel`/`custom`. |
| `CHAT_ROUTE_N_WOW_CHANNEL` | WoW channel name (for `channel`). Empty for other types. |
| `CHAT_ROUTE_N_WOW_FORMAT` | Message format when sending to WoW (placeholders `%user`, `%message`). |
| `CHAT_ROUTE_N_DISCORD_CHANNEL` | Discord channel: **name** (without `#`) or **snowflake-ID**. |
| `CHAT_ROUTE_N_DISCORD_FORMAT` | Message format when sending to Discord (placeholders `%time`, `%user`, `%message`, `%target`). |

> `channel`-type channels from all routes are auto-joined by the bot after
> entering the world. Discord channel matching works by both name and ID.

### Item Database

| Variable | Default | Purpose |
|---|---|---|
| `ITEM_DATABASE` | `https://db.warhoop.su` | Base URL for item links (takes priority), e.g. `https://db.warhoop.su/?item=12345`. |
| `DISCORD_ITEM_DATABASE` | _—_ | Fallback when `ITEM_DATABASE` is not set. |

WoW item links of the form `|Hitem:12345|h[name]|h` are automatically turned
into `[[name]](<URL>?item=12345)` for Discord.

### Filters

| Variable | Default | Purpose |
|---|---|---|
| `FILTERS_ENABLED` | `false` | Enable regex-based filtering. |
| `FILTER_PATTERNS` | _—_ | Space-separated list of patterns. Each is matched against the **entire** formatted message (`^(?:pattern)$`); a match drops the message in both directions. |

### Proxy

Supported schemes: `http://`, `https://`, `socks5://` (socks5 default port
`1080`). WoW traffic (realm + game TCP) and Discord traffic (REST + gateway)
are routed **independently** via two separate toggles.

| Variable | Default | Purpose |
|---|---|---|
| `PROXY_URL` | _—_ | Proxy address: `http(s)://[user:pass@]host:port` or `socks5://[user:pass@]host:port`. Empty = proxy disabled. |
| `PROXY_DISCORD_CONNECT` | `false` | Route all Discord traffic (REST + websocket gateway) through the proxy. |
| `PROXY_REALM_CONNECT` | `false` | Route the WoW realm server and game connection through the proxy. |
| `PROXY_INSECURE_SKIP_VERIFY` | `false` | Disable TLS certificate verification for the proxy (needed for self-signed / private-CA `https` proxies). |

> **VLESS / Xray / V2Ray.** VLESS itself is not a forward proxy, so run
> Xray/V2Ray with a local `socks` inbound and point `PROXY_URL` at it:
> ```json
> "inbounds": [{ "protocol": "socks", "listen": "127.0.0.1", "port": 10808 }]
> ```
> ```bash
> PROXY_URL=socks5://127.0.0.1:10808
> PROXY_DISCORD_CONNECT=true
> PROXY_REALM_CONNECT=true
> ```

---

## Message Formatting

Available placeholders (in `WOW_FORMAT` and `DISCORD_FORMAT`):

| Placeholder | Meaning |
|---|---|
| `%time` | Send time (`15:04:05`, Discord format only). |
| `%user` | Sender name. |
| `%message` | Message text. |
| `%target` | Channel/target name (for `channel` — the WoW channel name). |

Example: `DISCORD_FORMAT=[%target] [%user]: %message` →
`[LookingForGroup] [Herlo]: hi`.

---

## Security Notice

This application logs into a WoW realm as a real character.

Use a dedicated account. Do not use your main account unless you trust
the server and understand the risks involved.

The authors are not responsible for account bans, data loss,
or violations of server rules.