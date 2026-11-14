# Persistence Pattern

## Port อยู่ที่ `repository/`

Interface เป็นสัญญาของ module — explicit operations ไม่ใช่ generic `Save(T)`

```go
type Repository interface {
    Create(ctx context.Context, p *domain.Product) error
    GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
    Update(ctx context.Context, p *domain.Product) error
    ReserveStock(ctx context.Context, id uuid.UUID, qty int) error
    RestoreStock(ctx context.Context, id uuid.UUID, qty int) error
    WithTx(tx pgx.Tx) Repository
}
```

ตั้งชื่อ method ตามเจตนาธุรกิจเมื่อเป็น concurrency-sensitive (`ReserveStock`, `Redeem`) ไม่ใช่แค่ `Update` คลุมเครือ

## Postgres adapter

```
internal/{module}/adapter/postgres/repository.go
```

รูปแบบที่ต้องมี:

1. **`Querier` interface** ครอบ `Exec` / `Query` / `QueryRow` — ให้รับได้ทั้ง pool และ tx
2. **`NewRepository(pool)`** สำหรับ request ปกติ
3. **`WithTx(tx)`** คืน instance ใหม่ที่ชี้ tx เดียวกัน (type เป็น port)
4. Map PG errors → `apperr` (unique → `Conflict`, 0 rows ที่ต้องเจอ → `NotFound`)
5. Conditional update สำหรับแข่งกัน:

```sql
UPDATE products
SET stock = stock - $2
WHERE id = $1 AND stock >= $2 AND status = 'AVAILABLE'
```

`RowsAffected() == 0` → fail ด้วย error ธุรกิจที่เหมาะสม แล้วให้ tx rollback

## สองชั้นบังคับ invariant

| ชั้น | ตัวอย่าง |
|------|----------|
| Domain | ปฏิเสธ transition / qty ไม่พอ ก่อนเขียน |
| Database | UNIQUE, CHECK, conditional UPDATE |

อย่าเลือกอย่างใดอย่างหนึ่งอย่างเดียวถ้ามี concurrency จริง

## สิ่งที่ห้าม

- Module เรียก `*pgxpool.Pool` ตรงๆ จาก service โดยข้าม port (ยกเว้น `WithinTransaction` ที่ platform)
- Shared “BaseRepository” ที่ซ่อน SQL ต่างโดเมนไว้ที่เดียวก่อนมีเหตุผลจริง
- ใช้ ORM เป็นค่าเริ่มต้นของ pattern นี้
