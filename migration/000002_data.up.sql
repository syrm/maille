INSERT INTO account (type, name)
VALUES ('Assets'::account_type, 'Bank:Checking'),
       ('Assets'::account_type, 'Bank:Savings'),
       ('Assets'::account_type, 'Bank:Investment'),
       ('Expenses'::account_type, 'Other'),
       ('Expenses'::account_type, 'Shopping:Other'),
       ('Expenses'::account_type, 'Health:Pharmacy'),
       ('Expenses'::account_type, 'Transport:PublicTransit'),
       ('Expenses'::account_type, 'Travel:Transport')
;

INSERT INTO transaction_classifier_rule (rule, account_id)
VALUES ('payee contains "AMAZON"', 3),
       ('payee contains "SNCF VOYAGES"', 6),
       ('payee contains "RATP"', 5),
       ('payee contains "PHARMACIE"', 4)
;

INSERT INTO bank_account (name, account_id, external_id)
VALUES ('Main',
        (SELECT id FROM account WHERE type = 'Assets'::account_type AND name = 'Bank:Checking'),
        '123456789');
