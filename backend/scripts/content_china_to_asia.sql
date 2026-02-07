-- Content 表：在 name、description 中搜索 "China"，并将其替换为 "Asia"
-- 表名见 entity/content.go: TableName() -> "contents"

-- 1) 搜索：查找 name 或 description 中包含 "China" 的记录
SELECT id, name, description
FROM contents
WHERE name LIKE '%China%' OR description LIKE '%China%';

-- 2) 更新：将 name、description 中的 "China" 替换为 "Asia"
--    建议先执行上面的 SELECT 确认影响行数，再执行 UPDATE
UPDATE contents
SET
  name = REPLACE(name, 'China', 'Asia'),
  description = REPLACE(description, 'China', 'Asia')
WHERE name LIKE '%China%' OR description LIKE '%China%';
