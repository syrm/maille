INSERT INTO account (type, name, alias, icon, color)
VALUES ('Assets'::account_type, 'Bank:Checking', 'Checking', '💰', '#a8c8f0'),
       ('Assets'::account_type, 'Bank:Savings',  'Savings', '🏦', '#a8d5b5'),
       ('Assets'::account_type, 'Bank:Investment', 'Investment', '📈', '#f5d08a'),
       ('Expenses'::account_type, 'Other', 'Other', '🗂️', '#c4c9d4'),
       ('Expenses'::account_type, 'Shopping:Other', 'Shopping', '🛍️', '#f5a8b0'),
       ('Expenses'::account_type, 'Health:Pharmacy', 'Pharmacy', '💊', '#a8dce8'),
       ('Expenses'::account_type, 'Transport:PublicTransit', 'Public transit', '🚇', '#b8d4f0'),
       ('Expenses'::account_type, 'Travel:Transport', 'Transport', '✈️', '#f0c8a0')
;

INSERT INTO transaction_classifier_rule (rule, account_id)
VALUES ('payee contains "AMAZON"', (SELECT id FROM account WHERE name = 'Shopping:Other')),
       ('payee contains "SNCF VOYAGES"', (SELECT id FROM account WHERE name = 'Travel:Transport')),
       ('payee contains "RATP"', (SELECT id FROM account WHERE name = 'Transport:PublicTransit')),
       ('payee contains "PHARMACIE"', (SELECT id FROM account WHERE name = 'Health:Pharmacy'))
;

INSERT INTO bank_account (name, account_id, external_id)
VALUES ('Main',
        (SELECT id FROM account WHERE type = 'Assets'::account_type AND name = 'Bank:Checking'),
        '123456789');
