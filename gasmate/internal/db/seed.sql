-- Only inserted when table is empty

INSERT OR IGNORE INTO countries (name) VALUES ('India');

INSERT OR IGNORE INTO states (name, country_id) VALUES
    ('Tamil Nadu',     (SELECT id FROM countries WHERE name='India')),
    ('Karnataka',      (SELECT id FROM countries WHERE name='India')),
    ('Andhra Pradesh', (SELECT id FROM countries WHERE name='India')),
    ('Kerala',         (SELECT id FROM countries WHERE name='India')),
    ('Maharashtra',    (SELECT id FROM countries WHERE name='India'));

INSERT OR IGNORE INTO cities (name, state_id) VALUES
    ('Chennai',     (SELECT id FROM states WHERE name='Tamil Nadu')),
    ('Coimbatore',  (SELECT id FROM states WHERE name='Tamil Nadu')),
    ('Madurai',     (SELECT id FROM states WHERE name='Tamil Nadu')),
    ('Kumbakonam',  (SELECT id FROM states WHERE name='Tamil Nadu')),
    ('Trichy',      (SELECT id FROM states WHERE name='Tamil Nadu')),
    ('Bangalore',   (SELECT id FROM states WHERE name='Karnataka')),
    ('Mysore',      (SELECT id FROM states WHERE name='Karnataka')),
    ('Hyderabad',   (SELECT id FROM states WHERE name='Andhra Pradesh')),
    ('Vizag',       (SELECT id FROM states WHERE name='Andhra Pradesh')),
    ('Kochi',       (SELECT id FROM states WHERE name='Kerala')),
    ('Thiruvananthapuram', (SELECT id FROM states WHERE name='Kerala')),
    ('Mumbai',      (SELECT id FROM states WHERE name='Maharashtra')),
    ('Pune',        (SELECT id FROM states WHERE name='Maharashtra'));

INSERT OR IGNORE INTO gas_cylinders (unit_kg, type, price, description) VALUES
    (2,    'domestic',   320.00,  '2 kg domestic LPG cylinder — ideal for small households'),
    (5,    'domestic',   520.00,  '5 kg domestic LPG cylinder'),
    (14,   'domestic',  1050.00,  '14 kg domestic LPG cylinder — standard household size'),
    (19,   'commercial', 1750.00, '19 kg commercial LPG cylinder'),
    (35,   'commercial', 3200.00, '35 kg commercial LPG cylinder'),
    (47.5, 'commercial', 4100.00, '47.5 kg commercial LPG cylinder — for large establishments');

INSERT OR IGNORE INTO accessories (name, price, description) VALUES
    ('BURNER PRO',         850.00,  'High-efficiency single-burner LPG stove'),
    ('PORTABLE STOVE',     650.00,  'Compact portable gas stove'),
    ('GAS STOVE',         1200.00,  'Standard 2-burner gas stove'),
    ('REGULATOR',          220.00,  'ISI certified LPG regulator'),
    ('PETROMAX CYLINDER',  450.00,  'Petromax pressure lantern cylinder'),
    ('AUTOMATIC GAS STOVE',1800.00, 'Auto-ignition 3-burner gas stove'),
    ('TRADITIONAL STOVE',  400.00,  'Traditional clay stove'),
    ('BUNSEN BURNER',      350.00,  'Laboratory-grade Bunsen burner'),
    ('LPG HOSE',           180.00,  '1.5 m ISI certified LPG rubber hose');
