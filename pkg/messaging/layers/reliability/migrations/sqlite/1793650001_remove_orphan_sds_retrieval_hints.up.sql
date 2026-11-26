DELETE FROM sds_retrieval_hints
WHERE NOT EXISTS (
  SELECT 1 FROM sds_channel_history
  WHERE sds_channel_history.channel_id = sds_retrieval_hints.channel_id
    AND sds_channel_history.message_id = sds_retrieval_hints.message_id
);
