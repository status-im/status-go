adapters
========

Currently, only Whisper adapter is implemented but it is available as a Whisper service (useful if the node is embedded in your program) or Whisper client in a case when your program connects to a node running as a separate process (it might not even run on the same server).

## Tips

* In order to send a request to a MailServer you need a symmetric key created from a password. The password is stored in `mailserver.go`,
* In `whisper_topic.go` you can learn how to create topics for public and private chats,
* Adapters handle MailServer pagination (using cursor) and requesting for historic messages is synchronous.
