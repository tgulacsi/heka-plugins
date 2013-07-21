# TwilioOutput
Give Twilio's sid and token, a from and some to, and don't forget to set the
message_matcher!

# HttpSimpleInput
Simple HTTP endpoint for accepting messages - with simple clients (vanilla Python, curl, or even bash).
The fields of message are read from the query string - unknown fields will
go into the extra field store. The payload is read from the "payload" field,
if present.
If not present, than the POST's body is read as the payload.
