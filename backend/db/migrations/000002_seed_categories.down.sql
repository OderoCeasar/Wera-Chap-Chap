-- Only the seeded names, so a category added later survives a rollback.
--
-- The NOT EXISTS guards matter: tasks.category_id and tasker_skills.category_id
-- both reference this table, so on a database with any real data an unguarded
-- DELETE hits a foreign key violation, which aborts the migration and leaves
-- golang-migrate's schema_migrations marked dirty -- a state that blocks every
-- later `up` until it is forced by hand. Rolling this migration back should
-- remove what it inserted and leave anything in use alone.
DELETE FROM categories
WHERE name IN (
  'Home Repairs',
  'Furniture Assembly',
  'Cleaning',
  'Moving',
  'Delivery & Errands',
  'Yard Work',
  'Personal Assistant',
  'Handyman'
)
AND NOT EXISTS (SELECT 1 FROM tasks WHERE tasks.category_id = categories.id)
AND NOT EXISTS (SELECT 1 FROM tasker_skills WHERE tasker_skills.category_id = categories.id);
