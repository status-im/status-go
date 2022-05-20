CREATE TABLE IF NOT EXISTS notifications_settings (
  id TEXT PRIMARY KEY ON CONFLICT REPLACE,
  exemption BOOLEAN,
  text_value TEXT,
  int_value INT,
  bool_value BOOLEAN,
  ex_mute_all_messages BOOLEAN,
  ex_personal_mentions TEXT,
  ex_global_mentions TEXT,
  ex_other_messages TEXT
);

INSERT INTO notifications_settings (
  id, 
  exemption, 
  text_value, 
  int_value, 
  bool_value, 
  ex_mute_all_messages, 
  ex_personal_mentions,
  ex_global_mentions,
  ex_other_messages
) 
VALUES 
  ("AllowNotifications", 0, NULL, NULL, 1, NULL, NULL, NULL, NULL),
  ("OneToOneChats", 0, "SendAlerts", NULL, NULL, NULL, NULL, NULL, NULL),
  ("GroupChats", 0, "SendAlerts", NULL, NULL, NULL, NULL, NULL, NULL),
  ("PersonalMentions", 0, "SendAlerts", NULL, NULL, NULL, NULL, NULL, NULL),
  ("GlobalMentions", 0, "SendAlerts", NULL, NULL, NULL, NULL, NULL, NULL),
  ("AllMessages", 0, "TurnOff", NULL, NULL, NULL, NULL, NULL, NULL),
  ("ContactRequests", 0, "SendAlerts", NULL, NULL, NULL, NULL, NULL, NULL),
  ("IdentityVerificationRequests", 0, "SendAlerts", NULL, NULL, NULL, NULL, NULL, NULL),
  ("SoundEnabled", 0, NULL, NULL, 1, NULL, NULL, NULL, NULL),
  ("Volume", 0, NULL, 50, NULL, NULL, NULL, NULL, NULL),
  ("MessagePreview", 0, NULL, 2, NULL, NULL, NULL, NULL, NULL);