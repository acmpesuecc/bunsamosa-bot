# BunSamosa Bot

BunSamosa-Bot is the official Github bot for ACM PESUECC's Hacknight.

## Developer Guide

1. Install the bot in the Github Org

2. Create a `secrets-dev.yaml` file with the following fields

```yaml
# Private Key to authenticate with the Github bot
certPath:
# Webhook secret to receive Github events
webhookSecret:
# App ID of the bot installed in the Github organisation
appID:
# Github Org ID
orgID:
# Port on which the web server listens
webServerPort:
# Sqlite3 DB Path
dbPath:
# Timer Service Webhook URL
timerDaemonURL:
# Maintainer Leads with @ for mentioning
maintainer-leads:
# Tech Leads with @ for mentioning
tech-leads:
```

3. Update `dev/init.sql` with the list of maintainers and contributors

4. Run `make setup-schema`, stop the service and run `make populate-db` to add the maintainers and contributors

5. Run `make` to start the bot in development mode
