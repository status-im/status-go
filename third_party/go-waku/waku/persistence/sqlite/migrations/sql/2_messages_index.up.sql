CREATE INDEX IF NOT EXISTS i_msg_1 ON message(contentTopic ASC, pubsubTopic ASC, senderTimestamp ASC, id ASC);
CREATE INDEX IF NOT EXISTS i_msg_2 ON message(contentTopic DESC, pubsubTopic DESC, senderTimestamp DESC, id DESC);
