ALTER TABLE zones
  DROP COLUMN server;
ALTER TABLE zones
  DROP COLUMN key_name;
ALTER TABLE zones
  DROP COLUMN key_algo;
ALTER TABLE zones
  DROP COLUMN key_blob;
ALTER TABLE zones
  DROP COLUMN storage_facility;

RENAME TABLE zones TO domains;
