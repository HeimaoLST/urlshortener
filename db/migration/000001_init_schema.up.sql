CREATE TABLE "urls" (
  "id" bigserial PRIMARY KEY,
  "original_url" text NOT NULL,
  "short_code" text NOT NULL,
  "is_custom" bool NOT NULL DEFAULT false,
  "expired_at" timestamp NOT NULL,
  "created_at" timestamp NOT NULL DEFAULT (now())
);

CREATE INDEX ON "urls" ("short_code");

CREATE INDEX ON "urls" ("expired_at");

CREATE INDEX ON "urls" ("is_custom");
