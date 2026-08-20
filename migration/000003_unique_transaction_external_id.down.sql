ALTER TABLE "transaction"
    DROP CONSTRAINT transaction_import_key_unique,
    DROP COLUMN import_key;
