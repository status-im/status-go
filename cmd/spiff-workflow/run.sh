
#/bin/bash
go build -mod=vendor
rm ./tmp -rf
./populate-db --added-contacts 1 --contacts 2 --public-chats 1 --one-to-one-chats 4 --number-of-messages 2  --seed-phrase "wolf uncover ancient kiss deer blossom blind expose estate average cancel kiss"
