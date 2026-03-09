-- Qetero dev seed data
-- Run with:
--   docker exec constructconnect-db psql -U postgres -d qetero -f /path/to/seed.sql
-- Or via psql:
--   psql $DATABASE_URL -f scripts/seed.sql

-- ── Owner user ───────────────────────────────────────────────────────────────
-- Password: password123 (bcrypt, cost 12)
INSERT INTO users (id, name, phone, email, password_hash, role, verified)
VALUES (
  'a1b2c3d4-0000-0000-0000-000000000001',
  'Abebe Girma',
  '+251911000001',
  'abebe@qetero.com',
  '$2a$12$K8GpYbX2vZ3mN4pQ1rS5aOeWjL9cT7uY6dF0hM2bV4nX8kA3oJ1eC',
  'owner',
  true
) ON CONFLICT DO NOTHING;

-- ── Listings ─────────────────────────────────────────────────────────────────
INSERT INTO listings (id, owner_id, title, category, description, location, price_per_day, minimum_days, images, specs, is_available)
VALUES
(
  gen_random_uuid(),
  'a1b2c3d4-0000-0000-0000-000000000001',
  'CAT 320 Excavator',
  'excavator',
  '2019 CAT 320, 20T operating weight, 1.2m³ bucket, diesel. Well maintained, full service history. Operator available on request.',
  'Addis Ababa',
  4500, 2, '{}', '{"weight": "20T", "bucket": "1.2m³", "fuel": "diesel", "year": 2019}',
  true
),
(
  gen_random_uuid(),
  'a1b2c3d4-0000-0000-0000-000000000001',
  '50T Liebherr Mobile Crane',
  'crane',
  '50 tonne capacity mobile crane. Max boom 40m. Ideal for high-rise construction and heavy lifting. Operator included.',
  'Addis Ababa',
  9500, 3, '{}', '{"capacity": "50T", "boom": "40m", "fuel": "diesel"}',
  true
),
(
  gen_random_uuid(),
  'a1b2c3d4-0000-0000-0000-000000000001',
  'Caterpillar D6 Dozer',
  'dozer',
  'CAT D6 bulldozer, 6-way blade, ROPS cab, air conditioning. Good for site clearing, grading, and earthworks.',
  'Hawassa',
  3800, 2, '{}', '{"blade": "6-way", "fuel": "diesel", "year": 2020}',
  true
),
(
  gen_random_uuid(),
  'a1b2c3d4-0000-0000-0000-000000000001',
  '500KVA Perkins Generator',
  'generator',
  '500KVA Perkins silent generator. Ideal for construction sites and events. Includes automatic transfer switch.',
  'Addis Ababa',
  2200, 1, '{}', '{"capacity": "500KVA", "fuel": "diesel", "type": "silent"}',
  true
),
(
  gen_random_uuid(),
  'a1b2c3d4-0000-0000-0000-000000000001',
  'Toyota Dyna Water Truck 10,000L',
  'water_truck',
  '10,000 litre water truck. Used for dust suppression and site water supply. Available 6 days/week.',
  'Adama',
  1800, 1, '{}', '{"capacity": "10000L", "make": "Toyota Dyna"}',
  true
),
(
  gen_random_uuid(),
  'a1b2c3d4-0000-0000-0000-000000000001',
  'Concrete Mixer Truck 6m³',
  'concrete_mixer',
  'Ready-mix concrete truck, 6m³ drum. Can deliver to site within Addis Ababa ring road. Min 4m³ per trip.',
  'Addis Ababa',
  2800, 1, '{}', '{"drum_size": "6m³", "delivery": true}',
  true
),
(
  gen_random_uuid(),
  'a1b2c3d4-0000-0000-0000-000000000001',
  'Komatsu WA380 Wheel Loader',
  'loader',
  '2021 Komatsu WA380 wheel loader, 2.9m³ bucket. Low hours. Great for quarry work, aggregate loading, and site cleanup.',
  'Dire Dawa',
  3200, 2, '{}', '{"bucket": "2.9m³", "year": 2021, "fuel": "diesel"}',
  true
),
(
  gen_random_uuid(),
  'a1b2c3d4-0000-0000-0000-000000000001',
  'Toyota 3T Forklift',
  'forklift',
  '3 tonne diesel forklift. 4.5m max lift height. Suitable for warehouse and construction site material handling.',
  'Addis Ababa',
  1500, 1, '{}', '{"capacity": "3T", "lift_height": "4.5m", "fuel": "diesel"}',
  true
)
ON CONFLICT DO NOTHING;
