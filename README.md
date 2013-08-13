# Heka plugins
See [Building *hekad* with External Plugins](http://hekad.readthedocs.org/en/latest/installing.html#build-include-externals)
for compiling in plugins.

Basically, you'll need to edit the cmake/plugin_loader.cmake file and add

    add_external_plugin(git https://github.com/tgulacsi/heka-plugins master)

OR (if you don't need all the plugins)

    add_external_plugin(git https://github.com/tgulacsi/heka-plugins/email master)

for email.

Then add the needed extra packages' get into build.sh

    go get github.com/sfreiberg/gotwilio  # for twilio (SMS)
    go get github.com/tgulacsi/go-xmlrpc  # for mantis

right before `make`.

## TwilioOutput
Give Twilio's sid and token, a from and some to, and don't forget to set the
message_matcher!

    [sms]
    type = "TwilioOutput"
    message_matcher = "Severity <= 3"
    sid = "AC4d1e00928ee119e69a"
    token = "a9d323f90d8793f93d"
    from = "+36302740000"
    to = ["+1 858-500-3858"]

    [sms.retries]
    max_jitter = "1s"
    delay = "1s"
    max_retries = 3

## HttpSimpleInput
Simple HTTP endpoint for accepting messages - with simple clients (vanilla Python, curl, or even bash).
The fields of message are read from the query string - unknown fields will
go into the extra field store. The payload is read from the "payload" field,
if present.
If not present, than the POST's body is read as the payload.

    [HttpSimpleInput]
    address = ":5566"

## EmailOutput
Sends email with the given server OR directly (getting MX records) if no address is given.
Watch out: mail sending usually SLOW, thus send mail rarely or use a very fast mail server!

    [EmailOutput]
    message_matcher = "Severity <= 4"
    address = "mail.messagingengine.com:465"
    username = "test@example.eu"
    password = "passw"
    from = "hekad"
    to = ["test+heka@example.eu"]

## MantisOutput
Adds a new issue to the configured MantisBT instance.

    [MantisOutput]
    message_matcher = "Severity < 3"
    url = "https://www.example.com/mantis/xmlrpc.php"
    project = "Something"
    method = "new_issue"
    username = "user"
    password = "pwd"

