### How to write migrations?

We only write `up` migrations, `down` migrations are not always possible in sqlite or 
too complex/too expensive. For example to remove a column you would have to duplicate
the table, copy over the data, delete and recreated.
This can be very expensive for some tables (user_messages for example), so should not be attempted.


### Notes

One issue we faced multiple times is that updates to `user_messages` can be very 
expensive, leading to slow upgrade times and interrupted migrations. So avoid
writes if not necessary.
