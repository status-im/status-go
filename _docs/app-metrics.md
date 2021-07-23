
# Status usage data
_______
:warning: **We cannot guarantee that there are no ways to de-anonymous usage data you share. If you find yourself at risk if your identity were to be uncovered: don't enable sharing data!**

_______

Starting release 1.14, the Status mobile app asks to share data about how you use Status. Data is only ever shared if you opt in to doing so, you can review all data before it is sent and it is shared over Status' peer-to-peer network (Waku), just like a 1:1 message.

Sharing data is strictly opt-in, and can be changed at any times in Settings in the app.

### What is shared (Opt-in)
Status is an open-source platform. To verify that the app only collects and shares the events and metadata listed below, you can view the rules set to store metrics data in the [source code](https://github.com/status-im/status-go/blob/develop/appmetrics/validators.go).

- Your interactions with the app like clicks and screenviews
- Background activity and internal processes
- Settings and preferences

In detail this means the app can collect and share:
- Navigation events
- Screens that are navigated to
- Action events (for example button clicks, list swipes, switch toggles, etc)
- The time events are created
- Operating System
- App version
- Batch ID
- Time since last batch sent

For any data to be collected and shared, it needs to meet the rules set in the validator script. By policy of Status' Core Contributors, rules set in the validator script need to be assessed as posing a non-existent to low threat by being in:
- local storage over time
- aggregated storage from Status over time

### What will never be shared
No data will be shared unless you opt-in. Furthermore, Status commits to never collect identifiable data in its broadest sense. This means that we will never collect anything that we believe can be linked back to you. Including but not limited to:
- IP addresses
- Random name
- Chat key
- Public Ethereum addresses
- Account balance(s)
- Account history
- ENS name
- Input field entries (browser address bar, chat key/ENS entry)
- Any content you generate (images, messages, profile picture)
- Contacts or other (favorite) lists (wallet and browser favorites)
- Chat, group, community memberships

The above data can technically be logged, stored and shared. In order to verify that this does not happen, view the rules defined to store metrics data in the [source code](https://github.com/status-im/status-go/blob/develop/appmetrics/validators.go).

Moreover, while we made a best effort to make data collection as privacy-preserving as possible, we cannot guarantee that there are no ways to de-anonymous collected data. **If you find yourself at risk if your identity were to be uncovered: don't enable sharing data.**

### Purpose of the data
Aggregated data is used to inform product development. Given our principles, the type of data that is collected and its rudimentary nature, there is no incentive for Status or other parties to use the data for any other purpose. Status will never use these data for profit.


### Viewing the data
As of writing, we plan to make a Dashboard that shows the aggregated data public; Similar to metrics.status.im. As soon as this Dashboard is available we will provide a link here. The data that is stored on your device and shared can be viewed through the interface of the app.

_________

# How it works
`appmetrics` is a way to capture and transfer app usage data, end-to-end encrypted over Waku to a Status managed key.


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
Transmission happens over Waku, and as of now, all data is deleted locally after transmission, however (in the future) we might want to keep a copy of the data locally. If a batch is not transmitted within 7 days, data is deleted locally.


## Events
| Event                 | Value                                | Description                                                                                                         |
|-----------------------|--------------------------------------|---------------------------------------------------------------------------------------------------------------------|
| navigate-to           | {:view_id: "", :params {:screen ""}} | The user navigated to one of the screens. If the `view_id` has a `_stack` suffix, it could signify a top level tab. |
| screens/on-will-focus | {:view_id: "", :params {:screen ""}} | The user navigated to a top level tab.                                                                              |
