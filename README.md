# Breathbath ChatGPT

Breathbath ChatGPT integrates ChatGPT with popular message bots. 

# List of supported bots:
- Telegram Bots

## Secrets
To manage secrets you need [Transcrypt](https://github.com/elasticdog/transcrypt) to be installed.

Use https://github.com/elasticdog/transcrypt#initialize-a-clone-of-a-configured-repository to decrypt secrets from a cloned repository. 

## How to generate a bcrypt encoded hash

- Just run `chatgpt bcrypt`. 

- You will be prompted for a password of your choice.

- Copy hash to the users config `AUTH_USERS` `password_hash` field.

## How to add a new user to Telegram bot
- Copy the Telegram name of a person you want to add (it's the one with @ sign in front of it)
- As admin call `/adduser TelegramUserName telegram {password}`
- Check if it worked by calling `/users`
- Under the new user Telegram account just search for a user by name @breathbath_bot and add it
- On login prompt provide your {password}
- Enjoy