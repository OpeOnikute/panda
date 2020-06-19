# Go Panda

Go routine that scrapes the interwebs for images of pandas and emails them to me.

## How to use

- Clone the repo using `git clone https://github.com/OpeOnikute/go-panda.git`
- Add the MG_DOMAIN, MG_API_KEY and MAIL_RECIPIENT values in the .env file.

### Using Local Docker

- Run `docker build -t go-panda .`
- Then `docker container run -d go-panda`
- The default config in the dockerfile is for the cron to run at 10am everyday. You can modify this however you want, just remember to run `docker build` again.

### Using Docker Compose

- Run `docker-compose up -d`

### Using Your Machine (Linux)

- Run `go get -d`
- Then `echo "0 10 * * * <host-directory>/go-panda/cronjob" > /etc/crontabs/root`
- Then `crond -l 2 -f`

### Environment Variables

- MG_DOMAIN - Your Mailgun domain.
- MG_API_KEY - Your Mailgun private API key. **Do not commit this to source control.**
- MAIL_RECIPIENT - The email you want the pictures sent to.

# TODO
- Split the lambda into a seperate repo and export this as a package