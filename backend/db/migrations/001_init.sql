CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  full_name TEXT NOT NULL,
  phone TEXT,
  role VARCHAR(20) NOT NULL CHECK (role IN ('client','tasker')),
  avatar_url TEXT,
  is_verified BOOLEAN NOT NULL DEFAULT FALSE,
  background_check_passed BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tasker_profiles (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  bio TEXT,
  hourly_rate NUMERIC(12,2) DEFAULT 0,
  years_experience INT DEFAULT 0,
  service_radius_km NUMERIC(8,2) DEFAULT 10,
  is_available BOOLEAN NOT NULL DEFAULT TRUE,
  avg_rating NUMERIC(3,2) DEFAULT 0,
  total_reviews INT DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS categories (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  icon_url TEXT,
  description TEXT
);

CREATE TABLE IF NOT EXISTS tasker_skills (
  id BIGSERIAL PRIMARY KEY,
  tasker_id BIGINT NOT NULL REFERENCES tasker_profiles(id) ON DELETE CASCADE,
  category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
  UNIQUE(tasker_id, category_id)
);

CREATE TABLE IF NOT EXISTS tasker_availability (
  id BIGSERIAL PRIMARY KEY,
  tasker_id BIGINT NOT NULL REFERENCES tasker_profiles(id) ON DELETE CASCADE,
  day_of_week INT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
  start_time TIME NOT NULL,
  end_time TIME NOT NULL
);

CREATE TABLE IF NOT EXISTS tasks (
  id BIGSERIAL PRIMARY KEY,
  client_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  category_id BIGINT NOT NULL REFERENCES categories(id),
  title TEXT NOT NULL,
  description TEXT,
  location_address TEXT,
  location_lat NUMERIC(10,7),
  location_lng NUMERIC(10,7),
  budget_type VARCHAR(20) NOT NULL CHECK (budget_type IN ('fixed','hourly')),
  budget_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
  status VARCHAR(30) NOT NULL DEFAULT 'open' CHECK (status IN ('open','matched','in_progress','completed','cancelled')),
  scheduled_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS task_applications (
  id BIGSERIAL PRIMARY KEY,
  task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  tasker_id BIGINT NOT NULL REFERENCES tasker_profiles(id) ON DELETE CASCADE,
  proposed_rate NUMERIC(12,2) NOT NULL DEFAULT 0,
  cover_note TEXT,
  status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','accepted','rejected')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(task_id, tasker_id)
);

CREATE TABLE IF NOT EXISTS bookings (
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

CREATE TABLE IF NOT EXISTS messages (
  id BIGSERIAL PRIMARY KEY,
  booking_id BIGINT NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
  sender_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  is_read BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS reviews (
  id BIGSERIAL PRIMARY KEY,
  booking_id BIGINT NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
  reviewer_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reviewee_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  rating INT NOT NULL CHECK (rating BETWEEN 1 AND 5),
  comment TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(booking_id, reviewer_id)
);

CREATE TABLE IF NOT EXISTS payments (
  id BIGSERIAL PRIMARY KEY,
  booking_id BIGINT NOT NULL UNIQUE REFERENCES bookings(id) ON DELETE CASCADE,
  client_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tasker_id BIGINT NOT NULL REFERENCES tasker_profiles(id) ON DELETE CASCADE,
  amount NUMERIC(12,2) NOT NULL DEFAULT 0,
  tip_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
  stripe_payment_intent_id TEXT,
  status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','completed','refunded')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tasks_status_category ON tasks(status, category_id);
CREATE INDEX IF NOT EXISTS idx_messages_booking_created ON messages(booking_id, created_at);
CREATE INDEX IF NOT EXISTS idx_reviews_reviewee ON reviews(reviewee_id);
