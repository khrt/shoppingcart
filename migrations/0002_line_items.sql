-- +goose Up
CREATE TABLE IF NOT EXISTS "line_items" (
  "id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  "cart_id" integer,
  "product_id" integer,
  "quantity" integer DEFAULT 1,
  "created_at" datetime NOT NULL,
  "updated_at" datetime NOT NULL,
  CONSTRAINT "fk_carts_id" FOREIGN KEY ("cart_id") REFERENCES "carts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "uniq_cart_id_product_id" UNIQUE ("cart_id", "product_id")
);

-- +goose Down
DROP TABLE line_items;
