-- +goose up
ALTER TABLE users
ADD CONSTRAINT users_email_unique UNIQUE (email);

-- +goose down
ALTER TABLE users
DROP CONSTRAINT email;