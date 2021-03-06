CREATE TABLE schema_version (
    version SMALLINT UNSIGNED NOT NULL
);

CREATE TABLE users (
  id_user INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
  email VARCHAR(255) NOT NULL UNIQUE,
  password BINARY(92) NOT NULL,
  salt BINARY(64) NOT NULL,
  registration_time TIMESTAMP NOT NULL
) DEFAULT CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

CREATE TABLE user_sessions (
  id_session BLOB(255) NOT NULL,
  id_user INTEGER NOT NULL,
  time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY(id_user) REFERENCES users(id_user)
);

CREATE TABLE zones (
  id_zone INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
  id_user INTEGER NOT NULL,
  domain VARCHAR(255) NOT NULL,
  server VARCHAR(255),
  key_name VARCHAR(255) NOT NULL,
  key_algo ENUM("hmac-md5.sig-alg.reg.int.", "hmac-sha1.", "hmac-sha224.", "hmac-sha256.", "hmac-sha384.", "hmac-sha512.") NOT NULL DEFAULT "hmac-sha256.",
  key_blob BLOB NOT NULL,
  storage_facility ENUM("live", "history") NOT NULL DEFAULT "live",
  FOREIGN KEY(id_user) REFERENCES users(id_user)
) DEFAULT CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
