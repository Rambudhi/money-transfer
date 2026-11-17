INSERT INTO merchants (id, name, balance)
VALUES (1, 'Demo Merchant', 10000000)
ON CONFLICT (id) DO NOTHING;