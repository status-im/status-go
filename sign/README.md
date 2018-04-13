# sign 
`sign` package represents the API and signals for sending and receiving
signature request to and from our API user.

When a method is called that requires an additional signature confirmation from
a user (like, a transaction), it gets it's sign request.

Client of the API is then nofified of the sign request.

Client has a chance to approve the sign request (by providing a valid password)
or to discard it. When the request is approved, the locked functinality is
executed.

