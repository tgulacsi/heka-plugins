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
	"errors"
	"fmt"
	"github.com/mozilla-services/heka/pipeline"
	"github.com/sfreiberg/gotwilio"
)

type TwilioOutput struct {
	client   *gotwilio.TwilioClient
	From, To string
}

// Extract from, to, sid and token value from config and store it on the plugin instance.
func (o *TwilioOutput) Init(config interface{}) error {
	var (
		sid, token string
		err        error
	)
	conf := config.(pipeline.PluginConfig)
	if sid, err = confGetStr(conf, "sid"); err != nil {
		return err
	}
	if token, err = confGetStr(conf, "token"); err != nil {
		return err
	}
	o.client = gotwilio.NewTwilioClient(sid, token)
	if o.From, err = confGetStr(conf, "from"); err != nil {
		return err
	}
	if o.To, err = confGetStr(conf, "to"); err != nil {
		return err
	}
	return nil
}

func confGetStr(conf pipeline.PluginConfig, key string) (string, err) {
	intf, ok := conf[key]
	if !ok {
		return nil, errors.New("No '" + key + "' setting specified.")
	}
	var str string
	if str, ok = intf.(string); !ok {
		return nil, errors.New("'" + key + "' setting not a sequence.")
	}
	return str, nil
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
		hostname string
		exc      gotwilio.Exception
		err      error
	)

	for pack := range runner.InChan() {
		_, exc, err = o.client.SendSMS(o.From, o.To, pack.Message.GetPayload(), "", "")
		if err == nil && exc != nil {
			return fmt.Errorf("%s: %d\n%s", exc.Message, exc.Code, exc.MoreInfo)
		}

		pack.Recycle()
	}
	return
}

func init() {
	pipeline.RegisterPlugin("TwilioOutput", func() interface{} {
		return new(TwilioOutput)
	})
}
