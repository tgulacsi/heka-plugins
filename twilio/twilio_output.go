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

package twilio

import (
	"github.com/mozilla-services/heka/pipeline"
	"github.com/sfreiberg/gotwilio"
	plugins "github.com/tgulacsi/heka-plugins"

	"fmt"
	"time"
)

// TwilioOutput holds the config values for the Twilio Output plugin
type TwilioOutput struct {
	From   string
	To     []string
	client *gotwilio.Twilio
}

// TwilioOutputConfig is for reading the configuration file
type TwilioOutputConfig struct {
	Sid   string   `toml:"sid"`
	Token string   `toml:"token"`
	From  string   `toml:"from"`
	To    []string `toml:"to"`
}

// ConfigStruct returns the struct for reading the configuration file
func (o *TwilioOutput) ConfigStruct() interface{} {
	return new(TwilioOutputConfig)
}

// Init initializes the givegn TwilioOutput instance by
//extracting from, to, sid and token value from config
//and store it on the plugin instance.
func (o *TwilioOutput) Init(config interface{}) error {
	conf := config.(*TwilioOutputConfig)
	o.From, o.To = conf.From, conf.To
	o.client = gotwilio.NewTwilioClient(conf.Sid, conf.Token)
	return nil
}

//type Output interface {
//       Run(or OutputRunner, h PluginHelper) (err error)
//    }
//

// Run is the plugin's main loop
//iterates over received messages, checking against
//message hostname and delivering to the output if hostname is in our config.
func (o *TwilioOutput) Run(runner pipeline.OutputRunner, helper pipeline.PluginHelper) (
	err error) {

	var (
		to, sms string
		exc     *gotwilio.Exception
	)

	for pack := range runner.InChan() {
		sms = fmt.Sprintf("%s [%d] %s@%s: %s",
			plugins.TsTime(pack.Message.GetTimestamp()).Format(time.RFC3339),
			pack.Message.GetSeverity(), pack.Message.GetLogger(),
			pack.Message.GetHostname(), pack.Message.GetPayload())
		pack.Recycle()
		for _, to = range o.To {
			_, exc, err = o.client.SendSMS(o.From, to, sms, "", "")
			if err == nil && exc != nil {
				return fmt.Errorf("%s: %d\n%s", exc.Message, exc.Code, exc.MoreInfo)
			}
		}

	}
	return
}

func init() {
	pipeline.RegisterPlugin("TwilioOutput", func() interface{} {
		return new(TwilioOutput)
	})
}
