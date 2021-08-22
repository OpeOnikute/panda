package panda_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/opeonikute/panda"
)

func TestPanda(t *testing.T) {
	goPanda := panda.GoPanda{
		Config: panda.Settings{
			MgDomain:       os.Getenv("MG_DOMAIN"),
			MgKey:          os.Getenv("MG_API_KEY"),
			MailRecipients: os.Getenv("MAIL_RECIPIENT"),
			CdCloudName:    os.Getenv("CD_CLOUD_NAME"),
			CdUploadPreset: os.Getenv("CD_UPLOAD_PRESET"),
			MongoURL:       os.Getenv("MONGO_URL"),
			MongoDB:        os.Getenv("MONGO_DATABASE"),
			SendMail:       os.Getenv("SEND_EMAIL") == "true",
		},
	}
	_ = goPanda.Run(0)
}

func TestGetPOD(t *testing.T) {
	// TODO: Create a POD first
	goPanda := panda.GoPanda{
		Config: panda.Settings{
			MongoURL: os.Getenv("MONGO_URL"),
			MongoDB:  os.Getenv("MONGO_DATABASE"),
		},
	}

	tm := time.Now()
	res, err := goPanda.GetPOD(tm)

	if err != nil {
		t.Error(err)
		return
	}

	fmt.Printf("Panda found: %s\n", res)
}

func TestCreateGif(t *testing.T) {
	goPanda := panda.GoPanda{
		Config: panda.Settings{
			CdCloudName: os.Getenv("CD_CLOUD_NAME"),
			CdAPIKey:    os.Getenv("CD_API_KEY"),
			CdAPISecret: os.Getenv("CD_API_SECRET"),
		},
	}

	resp, err := goPanda.CreateGif()
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println("Response Body:", resp)
}
