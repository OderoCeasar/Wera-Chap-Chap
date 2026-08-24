-- Initial schema for the Wera Chap Chap marketplace.
--
-- Nullability here is deliberate rather than incidental: sqlc reads these files
-- as the source of truth for the generated Go types, so a column left nullable
-- becomes a *string/*float64 in every struct and every JSON response. Columns
-- the application always writes a value for are NOT NULL with a DEFAULT, which
-- keeps the generated types plain and the API shape unchanged. Only the three
-- genuinely-absent moments (scheduled_at, started_at, completed_at) stay
-- nullable, and those are pointers on purpose.

CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  full_name TEXT NOT NULL,
  phone TEXT NOT NULL DEFAULT '',
  role VARCHAR(20) NOT NULL CHECK (role IN ('client','tasker')),
  avatar_url TEXT NOT NULL DEFAULT '',
  is_verified BOOLEAN NOT NULL DEFAULT FALSE,
  background_check_passed BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tasker_profiles (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  bio TEXT NOT NULL DEFAULT '',
  hourly_rate NUMERIC(12,2) NOT NULL DEFAULT 0,
  years_experience INT NOT NULL DEFAULT 0,
  service_radius_km NUMERIC(8,2) NOT NULL DEFAULT 10,
  is_available BOOLEAN NOT NULL DEFAULT TRUE,
  -- avg_rating/total_reviews are denormalised from reviews; RecalculateTaskerRating
  -- is the only writer.
  avg_rating NUMERIC(3,2) NOT NULL DEFAULT 0,
  total_reviews INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE categories (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  icon_url TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE tasker_skills (
  id BIGSERIAL PRIMARY KEY,
  tasker_id BIGINT NOT NULL REFERENCES tasker_profiles(id) ON DELETE CASCADE,
  category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
  UNIQUE(tasker_id, category_id)
);

-- start_time/end_time are TEXT "HH:MM" rather than TIME. The matching service
-- compares them lexically against time.Format("15:04") and the client sends and
-- renders the same shape, so storing them as text keeps one representation end
-- to end instead of converting at three boundaries.
CREATE TABLE tasker_availability (
  id BIGSERIAL PRIMARY KEY,
  tasker_id BIGINT NOT NULL REFERENCES tasker_profiles(id) ON DELETE CASCADE,
  day_of_week INT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
  start_time TEXT NOT NULL CHECK (start_time ~ '^[0-2][0-9]:[0-5][0-9]$'),
  end_time TEXT NOT NULL CHECK (end_time ~ '^[0-2][0-9]:[0-5][0-9]$')
);

CREATE TABLE tasks (
  id BIGSERIAL PRIMARY KEY,
  client_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  category_id BIGINT NOT NULL REFERENCES categories(id),
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  location_address TEXT NOT NULL DEFAULT '',
  location_lat NUMERIC(10,7) NOT NULL DEFAULT 0,
  location_lng NUMERIC(10,7) NOT NULL DEFAULT 0,
  budget_type VARCHAR(20) NOT NULL CHECK (budget_type IN ('fixed','hourly')),
  budget_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
  status VARCHAR(30) NOT NULL DEFAULT 'open' CHECK (status IN ('open','matched','in_progress','completed','cancelled')),
  scheduled_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE task_applications (
  id BIGSERIAL PRIMARY KEY,
  task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  tasker_id BIGINT NOT NULL REFERENCES tasker_profiles(id) ON DELETE CASCADE,
  proposed_rate NUMERIC(12,2) NOT NULL DEFAULT 0,
  cover_note TEXT NOT NULL DEFAULT '',
  status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','accepted','rejected')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(task_id, tasker_id)
);

CREATE TABLE bookings (
  id BIGSERIAL PRIMARY KEY,
  task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  client_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tasker_id BIGINT NOT NULL REFERENCES tasker_profiles(id) ON DELETE CASCADE,
  agreed_rate NUMERIC(12,2) NOT NULL DEFAULT 0,
  status VARCHAR(20) NOT NULL DEFAULT 'confirmed' CHECK (status IN ('confirmed','started','completed','cancelled')),
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE messages (
  id BIGSERIAL PRIMARY KEY,
  booking_id BIGINT NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
  sender_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  is_read BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE reviews (
  id BIGSERIAL PRIMARY KEY,
  booking_id BIGINT NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
  reviewer_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reviewee_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  rating INT NOT NULL CHECK (rating BETWEEN 1 AND 5),
  comment TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(booking_id, reviewer_id)
);

CREATE TABLE payments (
  id BIGSERIAL PRIMARY KEY,
  booking_id BIGINT NOT NULL UNIQUE REFERENCES bookings(id) ON DELETE CASCADE,
  client_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tasker_id BIGINT NOT NULL REFERENCES tasker_profiles(id) ON DELETE CASCADE,
  amount NUMERIC(12,2) NOT NULL DEFAULT 0,
  tip_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
  -- Named for Stripe but carries whatever checkout id the active provider
  -- returns; the M-Pesa callback settles a payment by matching on it.
  stripe_payment_intent_id TEXT NOT NULL DEFAULT '',
  status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','completed','refunded')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tasks_status_category ON tasks(status, category_id);
CREATE INDEX idx_tasks_client ON tasks(client_id);
CREATE INDEX idx_task_applications_task ON task_applications(task_id);
CREATE INDEX idx_task_applications_tasker ON task_applications(tasker_id);
CREATE INDEX idx_bookings_client ON bookings(client_id);
CREATE INDEX idx_bookings_tasker ON bookings(tasker_id);
CREATE INDEX idx_messages_booking_created ON messages(booking_id, created_at);
CREATE INDEX idx_reviews_reviewee ON reviews(reviewee_id);
CREATE INDEX idx_tasker_skills_category ON tasker_skills(category_id);
CREATE INDEX idx_tasker_availability_tasker ON tasker_availability(tasker_id);
-- The M-Pesa callback arrives with only the checkout id to go on.
CREATE INDEX idx_payments_checkout ON payments(stripe_payment_intent_id);
