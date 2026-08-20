ALTER TABLE chats ADD COLUMN description TEXT DEFAULT "";
UPDATE chats SET description = "";
