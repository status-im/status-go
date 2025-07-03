# Scripts

# extract_logs.go

This script analyzes geth.log files in a specific format and extracts information related to "sent-message" actions or received messages. It then prints relevant details such as timestamp, recipients, message ID, message type, and hashes to the console.

## Usage

```bash
go run extract_logs.go -messages -received-message-count <filename>
```

It will output in tab separated values (TSV)
