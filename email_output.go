/***** BEGIN LICENSE BLOCK *****
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this file,
# You can obtain one at http://mozilla.org/MPL/2.0/.
#
# The Initial Developer of the Original Code is Tamás Gulácsi.
# Portions created by the Initial Developer are Copyright (C) 2013
# the Initial Developer. All Rights Reserved.
#
# ***** END LICENSE BLOCK *****/

package plugins

import (
	"github.com/mozilla-services/heka/pipeline"

	"bytes"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// EmailOutput holds the config values for the Email Output plugin
type EmailOutput struct {
	From     string
	To       []string
	hostport string
	auth     smtp.Auth
}

// EmailOutputConfig is for reading the configuration file
type EmailOutputConfig struct {
	Address  string   `toml:"address"`
	Username string   `toml:"username"`
	Password string   `toml:"password"`
	From     string   `toml:"from"`
	To       []string `toml:"to"`
}

// ConfigStruct returns the struct for reading the configuration file
func (o *EmailOutput) ConfigStruct() interface{} {
	return new(EmailOutputConfig)
}

// Init initializes the givegn EmailOutput instance by
//extracting from, to, sid and token value from config
//and store it on the plugin instance.
func (o *EmailOutput) Init(config interface{}) error {
	conf := config.(*EmailOutputConfig)
	o.hostport = conf.Address
	host := o.hostport
	if i := strings.Index(host, ":"); i >= 0 {
		host = host[:i]
	} else {
		o.hostport = host + ":25"
	}
	if conf.Username != "" {
		o.auth = smtp.PlainAuth("", conf.Username, conf.Password, host)
	}
	o.From, o.To = conf.From, conf.To
	return nil
}

//type Output interface {
//       Run(or OutputRunner, h PluginHelper) (err error)
//    }
//

// Run is the plugin's main loop
//iterates over received messages, checking against
//message hostname and delivering to the output if hostname is in our config.
func (o *EmailOutput) Run(runner pipeline.OutputRunner, helper pipeline.PluginHelper) (
	err error) {

	var payload string
	body := bytes.NewBuffer(nil)

	for pack := range runner.InChan() {
		payload = pack.Message.GetPayload()
		if len(payload) > 100 {
			payload = payload[:100]
		}
		body.WriteString(fmt.Sprintf("Subject: %s [%d] %s@%s: ",
			time.Unix(pack.Message.GetTimestamp(), 0).Format(time.RFC3339),
			pack.Message.GetSeverity(), pack.Message.GetLogger(),
			pack.Message.GetHostname()))
		body.WriteString(payload)
		body.WriteString("\r\n\r\n")
		body.WriteString(pack.Message.GetPayload())
		pack.Recycle()
		err = smtp.SendMail(o.hostport, o.auth, o.From, o.To, body.Bytes())
		body.Reset()
		if err != nil {
			return fmt.Errorf("error sending email: %s", err)
		}

	}
	return
}

func init() {
	pipeline.RegisterPlugin("EmailOutput", func() interface{} {
		return new(EmailOutput)
	})
}
