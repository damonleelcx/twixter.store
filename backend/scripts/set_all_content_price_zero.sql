-- 将所有内容的 price 设为 0
-- 表名见 entity/content.go: TableName() -> "contents"

-- 可选：先查看当前非零价格行数
-- SELECT COUNT(*) FROM contents WHERE price <> 0 AND deleted_at IS NULL;

-- 更新：仅未软删除的记录（与 GORM 默认查询一致）
UPDATE contents
SET price = 0
WHERE deleted_at IS NULL;

-- 若需连已软删除行一并清零，改用：
-- UPDATE contents SET price = 0;
