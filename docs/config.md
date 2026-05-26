# Configuration

`pswitch` persists runtime config in `settings.json`, which is written by the dashboard in the current working directory.

The TOML example below is a field reference for the runtime config shape. `pswitch` does not load this TOML file on startup.

```toml
# Listen address for the local proxy and dashboard.
listen = "127.0.0.1:8080"

# Provider selection strategy:
# - round_robin: rotate across healthy providers
# - sequential: always try providers in list order
# - least_failures: prefer healthy providers with fewer recent failures
mode = "round_robin"

# Circuit-break a provider after this many consecutive upstream failures.
failure_threshold = 1

# Wait this long before probing a failed provider again.
cooldown = "20s"

# How often the background health loop checks whether a provider should be probed.
health_check_interval = "15s"

# Timeout for each health probe request.
health_check_timeout = "3s"

[[routes]]
# Public route prefix exposed by pswitch.
prefix = "/codex"

# Protocol adapter used by this route.
type = "openai"

[[routes]]
# Optional route for Anthropic-compatible clients.
prefix = "/claude"
type = "anthropic"

# Model name advertised back to Anthropic-style clients.
model = "claude-sonnet-4-20250514"

# Actual upstream model sent to your provider.
upstream_model = "gpt-5.4"

[[providers]]
# Unique provider name used in the dashboard and route filters.
name = "provider-a"

# Base URL of the upstream API. Most providers expect the /v1 suffix here.
base_url = "https://provider-a.example/v1"

# Secret key forwarded to the upstream provider.
api_key = "sk-your-provider-a-key"

[[providers]]
# Add more providers to enable failover and traffic spreading.
name = "provider-b"
base_url = "https://provider-b.example/v1"
api_key = "sk-your-provider-b-key"
```

Fields:

- `listen`: local bind address
- `mode`: `sequential` or `round_robin`
- `failure_threshold`: consecutive failures before a provider is circuit-broken
- `cooldown`: how long to wait before probing a broken provider again
- `health_check_interval`: probe interval
- `health_check_timeout`: probe timeout

Admin console:

- The embedded admin UI is served at `/dashboard/`
- `PSWITCH_ADMIN_TOKEN` is optional; if it is set, the admin UI and admin API require it
- The admin API accepts the token in `X-PSwitch-Admin-Token`
- Changes to `mode`, health-check settings, `routes`, and `providers` are hot reloaded after save
- Changes saved from the admin UI are written to `settings.json` in the current working directory
- Changes to `listen` are persisted to `settings.json` but still require a process restart to take effect

Route fields:

- `prefix`: URL prefix, such as `/codex` or `/claude`
- `type`: `openai` or `anthropic`
- `model`: advertised model name for Anthropic clients
- `upstream_model`: actual upstream model to call for Anthropic routes

Each provider needs:

- `name`
- `base_url`
- `api_key`

If no saved runtime config exists yet, `pswitch` starts with a single OpenAI-compatible route on `/codex`.
