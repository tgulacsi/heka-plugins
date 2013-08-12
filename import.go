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
	_ "github.com/tgulacsi/heka-plugins/email"
	_ "github.com/tgulacsi/heka-plugins/http"
	_ "github.com/tgulacsi/heka-plugins/mantis"
	_ "github.com/tgulacsi/heka-plugins/twilio"
)
