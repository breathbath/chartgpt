# Breathbath ChartGPT

Breathbath ChartGPT integrates ChartGPT with popular message bots. 

# List of supported bots:
- Telegram Bots

## Secrets
To manage secrets you need [Transcrypt](https://github.com/elasticdog/transcrypt) to be installed.

Use https://github.com/elasticdog/transcrypt#initialize-a-clone-of-a-configured-repository to decrypt secrets from a cloned repository. 

## How to generate a bcrypt encoded hash

- Just run `chartgpt bcrypt`. 

- You will be prompted for a password of your choice.

- Copy hash to the users config `AUTH_USERS` `password_hash` field.