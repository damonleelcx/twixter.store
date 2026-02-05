-- Fix content_files: width/height were swapped with height/duration.
-- Before: width=0, height contained width (e.g. 1920), duration contained height (e.g. 1080).
-- After: width = old height, height = old duration (as int), duration = 0 (real duration not stored).
-- Run against PostgreSQL.

-- Option A: Fix only rows that look wrong (width 0 and height/duration look like dimensions)
UPDATE content_files
SET
  width = height,
  height = ROUND(duration)::integer,
  duration = 0
WHERE deleted_at IS NULL
  AND (width = 0 OR width IS NULL)
  AND height IS NOT NULL AND height > 0
  AND duration IS NOT NULL AND duration > 0;

-- Option B: Fix all rows (uncomment to use instead of Option A)
-- UPDATE content_files
-- SET
--   width = height,
--   height = ROUND(duration)::integer,
--   duration = 0
-- WHERE deleted_at IS NULL;
