-- +goose Up
CREATE TABLE IF NOT EXISTS "carts" (
  "id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  "user_id" integer,
  "created_at" datetime NOT NULL,
  "updated_at" datetime NOT NULL
);

-- +goose Down
DROP TABLE carts;
