ALTER TABLE "urls" DROP CONSTRAINT IF EXISTS "fk_user";

ALTER TABLE "urls" DROP COLUMN IF EXISTS "user_id";


DROP TRIGGER IF EXISTS set_timestamp ON "users";


DROP FUNCTION IF EXISTS trigger_set_timestamp();

DROP TABLE IF EXISTS "users";
