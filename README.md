# Go Panda

Go package that scrapes the interwebs for images of pandas and emails them and stores in a database for future fetching. If you're here to see today's Panda, the website is https://daily-panda.herokuapp.com.

## How to use

- Run `go get github.com/opeonikute/panda`

- To fetch a panda image, save it to your cloudinary and mongo database:
```
    import "github.com/opeonikute/panda"

	goPanda := panda.GoPanda{
		Config: panda.Settings{
			MgDomain:       os.Getenv("MG_DOMAIN"),
			MgKey:          os.Getenv("MG_API_KEY"),
			MailRecipients: os.Getenv("MAIL_RECIPIENT"),
			CdCloudName:    os.Getenv("CD_CLOUD_NAME"),
			CdUploadPreset: os.Getenv("CD_UPLOAD_PRESET"),
			MongoURL:       os.Getenv("MONGO_URL"),
			MongoDB:        os.Getenv("MONGO_DATABASE"),
		},
	}
	res, err = goPanda.Run(0)

    if err != nil {
		log.Fatal(err)
	}

    fmt.Println(res)
```

- To get today's panda (from your Mongo Database0)
```
    import "github.com/opeonikute/panda"
    
    goPanda := panda.GoPanda{
		Config: panda.Settings{
			MongoURL: os.Getenv("MONGO_URL"),
			MongoDB:  os.Getenv("MONGO_DATABASE"),
		},
	}

	tm := time.Now()
	res, err := goPanda.GetPOD(tm)

	if err != nil {
		log.Fatal(err)
	}

    fmt.Println(res)
```

### Using Docker Compose
The compose file is used to start a local Mongo container.
- Run `docker-compose up -d`

### Environment Variables

- MG_DOMAIN - Your Mailgun domain.
- MG_API_KEY - Your Mailgun private API key. **Do not commit this to source control.**
- MAIL_RECIPIENT - The email you want the pictures sent to.
- CD_CLOUD_NAME - Cloudflare cloud name
- CD_UPLOAD_PRESET - Cloudflare upload preset (unsigned uploads)
- MONGO_URL - Mongo connection string
- MONGO_DATABASE - Mongo database name

# TODO
- Add features to store the image, and retrieve the image of the day.
- Create IAM Role, lambda function, event service rule with terraform.