package panda_test

import (
	"os"
	"testing"

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
		},
	}
	_ = goPanda.Run(0)
}
