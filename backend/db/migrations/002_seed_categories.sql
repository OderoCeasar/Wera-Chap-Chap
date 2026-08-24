INSERT INTO categories (name, icon_url, description) VALUES
('Home Repairs', '🛠️', 'Fixes, maintenance and odd jobs around the home.'),
('Furniture Assembly', '🪑', 'Flat-pack and custom furniture assembly.'),
('Cleaning', '🧽', 'Home, office and post-event cleaning.'),
('Moving', '🚚', 'Packing, lifting and relocation help.'),
('Delivery & Errands', '🛵', 'Fast errands and item delivery.'),
('Yard Work', '🌿', 'Gardening, mowing and outdoor cleanup.'),
('Personal Assistant', '📋', 'Admin, scheduling and personal support.'),
('Handyman', '🔧', 'General repairs and skilled help.')
ON CONFLICT (name) DO NOTHING;
