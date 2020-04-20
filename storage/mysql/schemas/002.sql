ALTER TABLE user_sessions
  DROP FOREIGN KEY user_sessions_ibfk_1;

ALTER TABLE zones
  DROP FOREIGN KEY zones_ibfk_1;

ALTER TABLE users
  CHANGE id_user id_user BIGINT NOT NULL AUTO_INCREMENT;

ALTER TABLE user_sessions
  CHANGE id_user id_user BIGINT NOT NULL;

ALTER TABLE zones
  CHANGE id_user id_user BIGINT NOT NULL;
