ALTER TABLE "transaction"
    ADD COLUMN import_key TEXT;

-- Preserve historical duplicate rows while reserving every external ID that has
-- already been imported. New imports always populate import_key.
WITH imported_external_ids AS (
    SELECT MIN(id) AS transaction_id, external_id
    FROM "transaction"
    GROUP BY external_id
)
UPDATE "transaction" AS transaction_to_reserve
SET import_key = imported_external_ids.external_id
FROM imported_external_ids
WHERE transaction_to_reserve.id = imported_external_ids.transaction_id;

ALTER TABLE "transaction"
    ADD CONSTRAINT transaction_import_key_unique UNIQUE (import_key);
