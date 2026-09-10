-- Demo admin user: email admin@seki.dev / password AdminPass123
INSERT INTO users (email, password_hash, role)
VALUES ('admin@seki.dev', '$2b$12$i4wrCScyv0Xlt8jK0/TW9OJnIanDu5ijfFZOijMQAZWhPTZM.4kw6', 'admin')
ON CONFLICT (email) DO NOTHING;

-- A couple of bookable resources for demoing the API / load test
INSERT INTO resources (name, description, capacity)
VALUES
  ('Meeting Room A', 'Small meeting room, 4th floor, seats 6', 6),
  ('Meeting Room B', 'Large conference room with projector', 20)
ON CONFLICT DO NOTHING;
