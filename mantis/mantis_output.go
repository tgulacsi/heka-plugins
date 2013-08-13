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

package mantis

import (
	"github.com/mozilla-services/heka/pipeline"
	"github.com/tgulacsi/heka-plugins/utils"

	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/tgulacsi/go-xmlrpc"
	"io"
	"log"
	"net/http"
	"time"
)

// MantisOutput holds the config values for the Mantis Output plugin
type MantisOutput struct {
	sender *mantisSender
}

// MantisOutputConfig is for reading the configuration file
type MantisOutputConfig struct {
	URL         string `toml:"url"` //URL of the xmlrpc.php
	Project     string `toml:"project"`
	Category    string `toml:"category"`
	Method      string `toml:"method"`
	Username    string `toml:"username"`
	Password    string `toml:"password"`
	NoCertCheck bool   `toml:"no_cert_check"`
}

// ConfigStruct returns the struct for reading the configuration file
func (o *MantisOutput) ConfigStruct() interface{} {
	return new(MantisOutputConfig)
}

// Init initializes the givegn MantisOutput instance by
//extracting from, to, sid and token value from config
//and store it on the plugin instance.
func (o *MantisOutput) Init(config interface{}) error {
	conf := config.(*MantisOutputConfig)
	o.sender = NewMantisSender(conf.URL, conf.Project, conf.Category, conf.Method,
		conf.Username, conf.Password, conf.NoCertCheck)
	return nil
}

//type Output interface {
//       Run(or OutputRunner, h PluginHelper) (err error)
//    }
//

// Run is the plugin's main loop
//iterates over received messages, checking against
//message hostname and delivering to the output if hostname is in our config.
func (o *MantisOutput) Run(runner pipeline.OutputRunner, helper pipeline.PluginHelper) (
	err error) {

	var (
		short, long string
		//issue       int
	)

	for pack := range runner.InChan() {
		long = pack.Message.GetPayload()
		short = fmt.Sprintf("%s [%d] %s@%s: %s",
			utils.TsTime(pack.Message.GetTimestamp()).Format(time.RFC3339),
			pack.Message.GetSeverity(), pack.Message.GetLogger(),
			pack.Message.GetHostname(), long)
		pack.Recycle()
		if _, err = o.sender.Send(short, long); err != nil {
			return fmt.Errorf("error sending to %s: %s", o.sender.URL, err)
		}

	}
	return
}

type callFunc func(subject, body string) (int, error)

type mantisSender struct {
	URL, project, category, username, password, method string
	callers                                            map[string]callFunc
	client                                             *http.Client
}

// NewMantisSender returns a new Mantis sender
func NewMantisSender(url, project, category, method, username, password string, noCertCheck bool) *mantisSender {
	ms := &mantisSender{callers: make(map[string]callFunc, 4)}
	if method == "" {
		method = "new_issue"
	}
	ms.URL, ms.project, ms.category, ms.method = url, project, category, method
	ms.username, ms.password = username, password
	if ms.method == "" {
		ms.method = "new_issue"
	}
	tr := &http.Transport{
		DisableKeepAlives:     false,
		DisableCompression:    false,
		ResponseHeaderTimeout: 30,
	}
	if noCertCheck {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	ms.client = &http.Client{Transport: tr}
	return ms
}

// Send creates a new Mantis issue on the given uri
func (ms *mantisSender) Send(subject, body string) (int, error) {
	if ms.callers == nil {
		ms.callers = make(map[string]callFunc, 1)
	}
	call, ok := ms.callers[ms.URL]
	if !ok {
		call = func(subject, body string) (int, error) {
			args := map[string]string{"project_name": ms.project,
				"summary":     subject,
				"description": body,
				"category":    ms.category}
			log.Printf("calling %s new_issue(%v)", ms.URL, args)
			resp, fault, err := Call(ms.client, ms.URL, ms.username, ms.password, ms.method, args)
			log.Printf("got %v, %v, %s", resp, fault, err)
			if err == nil {
				log.Printf("response: %v", resp)
				return -1, fault
			}
			return 0, err
		}
	}
	return call(subject, body)
}

// Call is an xmlrpc.Call, but without gzip and Basic Auth and strips non-xml
func Call(client *http.Client, uri, username, password, name string, args ...interface{}) (
	interface{}, *xmlrpc.Fault, error) {

	buf := bytes.NewBuffer(nil)
	e := xmlrpc.Marshal(buf, name, args...)
	if e != nil {
		return nil, nil, e
	}
	req, e := http.NewRequest("POST", uri, buf)
	if e != nil {
		return nil, nil, e
	}
	req.SetBasicAuth(username, password)
	r, e := client.Do(req)
	//r, e := client.Post(uri, "text/xml", buf)
	if e != nil {
		return nil, nil, e
	}

	buf.Reset()
	n, e := io.Copy(buf, r.Body)
	r.Body.Close()
	if e != nil {
		return nil, nil, e
	}
	b := buf.Bytes()[:n]
	log.Printf("got\n%s", b)
	i := bytes.Index(b, []byte("<?"))
	if i < 0 {
		return nil, nil, io.EOF
	}
	_, v, f, e := xmlrpc.Unmarshal(bytes.NewReader(b[i:]))
	return v, f, e
}

func init() {
	pipeline.RegisterPlugin("MantisOutput", func() interface{} {
		return new(MantisOutput)
	})
}
