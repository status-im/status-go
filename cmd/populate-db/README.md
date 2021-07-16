### How to build

You must have go installed.
Then you can run, from `cmd/populate-db`

```
go build
```

which should create a `populate-db` executable

### How to run
```
./populate-db --added-contacts 100 --contacts 200 --public-chats 100 --one-to-one-chats 40 --number-of-messages 2  --seed-phrase "your seed phrase"
```


The parameters are:

`added-contacts`: contacts you have added
`contacts`: number of "contacts" in the database, these are not added by you
`one-to-one-chats`: the number of one to one chats open
`public-chats`: the number of public chats
`number-of-messages`: the number of messages in each chat
`seed-phrase`: the seed phrase of the account to be created

The db will be created in the `./tmp` directory

### How to import the db

1) Create an account in status-react
2) Login, copy the seed phrase
3) Create a db using this script using the seed phrase
4) Copy the db to the import directory
5) Import the database
6) Login


Note that the db is not complete, so the app might not be fully functioning, but it 
should be good enough to test performance and probably migrations
