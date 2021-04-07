# App Metrics

`appmetrics` is a way to capture and transfer app usage data, anonymously over Waku to a Status managed key.


## History
To learn more about the history, background and debates around this feature, refer to this detailed note: https://notes.status.im/anonymous-metrics


## Implementation
On the Go side, the metrics system is just a table, which can store predefined events and the their values, along with metadata like create time, app version and operating system. The collected data will never contain any personally identifiable information like three letter public name or the public wallet address. There is a validation layer that ensures (to some extent) that developers don't capture sensitive data by mistake.

### Opt-in system
These data points are saved in the local sqlite database, and are not transferred to our servers until the user opts-in to share metrics. If the user opts-out, we stop capturing data.

### Validation and audit
The interesting bit is the validation process, which serves two purposes:
1. validates the metric before saving it to local db: ensures that we don't capture any sensitive information
2. acts as an audit layer: anyone who wishes to check the kind of data being captured, can do so by auditing just one file: https://github.com/status-im/status-go/blob/develop/appmetrics/validators.go

### Transmission and deletion
Transmission happens over Waku, and as of now, all data will be deleted locally after transmission, however (in future) we might want to keep a copy of the data locally.


## Events
| Event                 | Value                                | Description                                                                                                         |
|-----------------------|--------------------------------------|---------------------------------------------------------------------------------------------------------------------|
| navigate-to           | {:view_id: "", :params {:screen ""}} | The user navigated to one of the screens. If the `view_id` has a `_stack` suffix, it could signify a top level tab. |
| screens/on-will-focus | {:view_id: "", :params {:screen ""}} | The user navigated to a top level tab.                                                                              |

