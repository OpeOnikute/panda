package panda

import (
	"fmt"
	"reflect"
)

// Settings exposes all package settings
type Settings struct {
	MgKey          string
	MgDomain       string
	MailRecipients string
	EmailSender    string
	CdCloudName    string
	CdUploadPreset string
	CdAPIKey       string
	CdAPISecret    string
	MongoURL       string
	MongoDB        string
}

func (s *Settings) get(name string) (interface{}, error) {

	rv := reflect.ValueOf(s)

	fv := reflect.Indirect(rv).FieldByName(name)
	if !fv.IsValid() {
		return nil, fmt.Errorf("not a field name: %s", name)
	}
	return fv.Interface(), nil
}
