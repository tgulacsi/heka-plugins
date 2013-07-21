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
	"fmt"
	"github.com/mozilla-services/heka/pipeline"
	"github.com/sfreiberg/gotwilio"
	"time"
)

type TwilioOutput struct {
	client *gotwilio.Twilio
	From   string
	To     []string
}

type TwilioOutputConfig struct {
	Sid   string   `toml:"sid"`
	Token string   `toml:"token"`
	From  string   `toml:"from"`
	To    []string `toml:"to"`
}

func (o *TwilioOutput) ConfigStruct() interface{} {
	return new(TwilioOutputConfig)
}

// Extract from, to, sid and token value from config and store it on the plugin instance.
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

// Fetch correct output and iterate over received messages, checking against
// message hostname and delivering to the output if hostname is in our config.
func (o *TwilioOutput) Run(runner pipeline.OutputRunner, helper pipeline.PluginHelper) (
	err error) {

	var (
		to, sms string
		exc *gotwilio.Exception
	)

	for pack := range runner.InChan() {
		sms = fmt.Sprintf("%s [%d] %s@%s: %s",
			time.Unix(pack.Message.GetTimestamp(), 0).Format(time.RFC3339),
			pack.Message.GetSeverity(), pack.Message.GetLogger(),
			pack.Message.GetHostname(), pack.Message.GetPayload())
        for _, to = range o.To {
		_, exc, err = o.client.SendSMS(o.From, to, sms, "", "")
		if err == nil && exc != nil {
			return fmt.Errorf("%s: %d\n%s", exc.Message, exc.Code, exc.MoreInfo)
		}}

		pack.Recycle()
	}
	return
}

func init() {
	pipeline.RegisterPlugin("TwilioOutput", func() interface{} {
		return new(TwilioOutput)
	})
}
