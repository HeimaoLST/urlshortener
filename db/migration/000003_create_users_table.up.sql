

CREATE TABLE "users" (
  "id" bigserial PRIMARY KEY,
  "username" varchar(50) NOT NULL UNIQUE,
  "email" varchar(255) NOT NULL UNIQUE,
  "password_hash" varchar(255) NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);


CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_timestamp
BEFORE UPDATE ON "users"
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

-- 修改 urls 表
ALTER TABLE "urls" ADD COLUMN "user_id" bigint;

-- 添加外键约束
ALTER TABLE "urls" ADD CONSTRAINT "fk_user"
FOREIGN KEY ("user_id") REFERENCES "users"("id")
ON DELETE SET NULL; -- 或者 ON DELETE CASCADE，取决于您的业务需求