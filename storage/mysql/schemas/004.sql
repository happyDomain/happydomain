ALTER TABLE users
  DROP COLUMN salt;
ALTER TABLE users
  MODIFY COLUMN password VARBINARY(255);