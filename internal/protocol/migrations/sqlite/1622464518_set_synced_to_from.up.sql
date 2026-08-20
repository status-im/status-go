UPDATE chats SET synced_to = strftime('%s', 'now') - 86400, synced_from = strftime('%s', 'now') - 86400;
