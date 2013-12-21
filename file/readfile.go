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

package file

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"text/template"

	"github.com/mozilla-services/heka/message"
	"github.com/mozilla-services/heka/pipeline"

	"code.google.com/p/go.text/transform"
	"github.com/tgulacsi/go/text"
)

// FileReadFilter plugin for expanding messages from external files based on template
// in the template you can use the Message's fields
type FileReadFilter struct {
	tmpl    *template.Template
	newType string
	decoder transform.Transformer
}

// Extract hosts value from config and store it on the plugin instance.
func (fr *FileReadFilter) Init(config interface{}) (err error) {
	var (
		value interface{}
		str   string
		ok    bool
	)
	conf := config.(pipeline.PluginConfig)
	if value, ok = conf["newType"]; !ok {
		return errors.New("FileReadFilter: No 'newType' setting specified.")
	}
	if fr.newType, ok = value.(string); !ok {
		return errors.New("FileReadFilter: 'newType' setting not a string value.")
	}
	if value, ok = conf["sourceEncoding"]; !ok {
		log.Printf("FileReadFilter: no source encoding specified - source should be UTF-8!")
	} else {
		if str, ok = value.(string); !ok {
			return errors.New("FileReadFilter: 'sourceEncoding' setting not a string value.")
		}
		e := text.GetEncoding(str)
		if e == nil {
			return fmt.Errorf("FileReadFilter: 'sourceEncoding' with value %q is unknown.", str)
		}
		fr.decoder = e.NewDecoder()
	}

	if value, ok = conf["pattern"]; !ok {
		return errors.New("FileReadFilter: No 'pattern' setting specified.")
	}
	if str, ok = value.(string); !ok {
		return errors.New("FileReadFilter: 'pattern' setting not a string value.")
	}

	if fr.tmpl, err = template.New("filereader").Parse(str); err == nil && fr.tmpl == nil {
		return fmt.Errorf("FileReadFilter: empty template (%q)", str)
	}
	return err
}

type extendedMessage struct {
	*message.Message
}

func (m extendedMessage) GetField(name string) string {
	f := m.Message.FindFirstField(name)
	if f != nil {
		return f.ValueString[0]
	}
	return ""
}

// Run runs the FileReadFilter filter, which inspects each message, and appends
// the content of the file named as the executed template to the existing payload.
// The resulting message will be injected back, and have newType type.
func (fr FileReadFilter) Run(r pipeline.FilterRunner, h pipeline.PluginHelper) (err error) {
	if fr.tmpl == nil {
		return errors.New("FileReadFilter: empty template")
	}
	var (
		fh           *os.File
		inp          io.Reader
		npack, opack *pipeline.PipelinePack
	)
	out := bytes.NewBuffer(make([]byte, 0, 4096))
	log.Printf("FileReadFilter: Starting with template %s", fr.tmpl)
	for opack = range r.InChan() {
		//log.Printf("opack=%v", opack)
		//if opack.Decoded {
		out.Reset()
		if err = fr.tmpl.Execute(out, extendedMessage{opack.Message}); err != nil {
			opack.Recycle()
			return fmt.Errorf("FileReadFilter: error executing template %v with message %v: %v",
				fr.tmpl, opack.Message, err)
		}
		//log.Printf("out=%q", out)
		if fh, err = os.Open(out.String()); err != nil {
			log.Printf("FileReadFilter: cannot read %q: %v", out, err)
			opack.Recycle()
			continue
		}
		out.Reset()
		//if _, err = io.Copy(out, io.LimitedReader{R: fh, N: 65000}); err != nil && err != io.EOF {
		inp = fh
		if fr.decoder != nil {
			inp = transform.NewReader(fh, fr.decoder)
		}
		if _, err = io.Copy(out, inp); err != nil && err != io.EOF {
			log.Printf("FileReadFilter: error reading %q: %v", fh.Name(), err)
			opack.Recycle()
			fh.Close()
			continue
		}
		fh.Close()

		npack = h.PipelinePack(opack.MsgLoopCount)
		if npack == nil {
			opack.Recycle()
			return errors.New("FileReadFilter: no output pack - infinite loop?")
		}
		npack.Decoded = true
		npack.Message = message.CopyMessage(opack.Message)
		npack.Message.SetType(fr.newType)
		npack.Message.SetPayload(npack.Message.GetPayload() + "\n" + out.String())
		if !r.Inject(npack) {
			log.Printf("FileReadFilter: cannot inject new pack %v", npack)
		}
		//}
		opack.Recycle()
	}
	return nil
}

func init() {
	pipeline.RegisterPlugin("FileReadFilter", func() interface{} { return new(FileReadFilter) })
}
