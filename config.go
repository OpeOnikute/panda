package panda

import (
	"fmt"
	"reflect"
)

// Config exposes all package settings
type Config struct {
	MgKey          string
	MgDomain       string
	MailRecipients string
	EmailSender    string
}

func (c *Config) get(name string) (interface{}, error) {

	rv := reflect.ValueOf(c)

	fv := reflect.Indirect(rv).FieldByName(name)
	if !fv.IsValid() {
		return nil, fmt.Errorf("not a field name: %s", name)
	}
	return fv.Interface(), nil
}
