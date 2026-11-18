# Module Scaffold

แม่แบบโฟลเดอร์เมื่อเพิ่ม module / bounded context ใหม่ชื่อ `{module}`

## โครงมาตรฐาน

```
internal/{module}/
  domain/
    {entity}.go          # aggregate root + methods
    {entity}_test.go     # unit tests (ไม่ต้องมี DB)
  repository/
    repository.go        # port (interface) เท่านั้น
  service/
    service.go           # application service ของ module
  adapter/
    http/
      handler.go         # Gin routes + DTO
    sqlite/
      repository.go      # implements repository.Repository
```

Cross-module use case (ถ้ามี) แยกแพ็กเกจระดับ `internal/` ไม่ฝังใน module เดียว:

```
internal/{usecase}/
  service/
    service.go           # orchestrate หลาย repository ports + WithinTransaction
```

ตัวอย่างใน POC: `internal/checkout/service`

## แพ็กเกจละทำอะไร

| Path | สร้างเมื่อ | ห้ามใส่ |
|------|-----------|---------|
| `domain/` | มี aggregate / rule | SQL, HTTP, DTO request |
| `repository/` | ต้อง persist | implementation |
| `service/` | มี use case ของ module | raw SQL, `http.ResponseWriter` |
| `adapter/http/` | มี REST API | business rule ซ้ำกับ domain |
| `adapter/sqlite/` | มีตาราง | business rule ที่ควรอยู่ domain |

## Wiring ใน bootstrap

ใน `internal/bootstrap/app.go` ลำดับเดิมเสมอ:

1. สร้าง SQLite repository จาก pool
2. สร้าง module service จาก port
3. สร้าง cross-module service (ถ้ามี) จาก pool + ports
4. สร้าง HTTP handler จาก service
5. `RegisterRoutes` ใต้ `/api/v1/...`

อย่าให้ handler สร้าง repository เอง

## ตั้งชื่อ

- Aggregate / entity: คำโดเมนชัดเจน (`Product`, ไม่ใช่ `ProductModel`)
- Port: `repository.Repository` ในแพ็กเกจของ module นั้น
- SQLite impl: `adapter/sqlite.Repository` — method set ตรงกับ port
- HTTP DTO: `createRequest`, `{entity}Response` ในไฟล์ handler (ไม่แชร์เป็น public API ของ domain)

## Migration

ทุก aggregate ที่ persist ต้องมีตารางใน `migrations/` (Goose: `NNNNNN_name.sql` พร้อม `-- +goose Up` / `-- +goose Down`) พร้อม:

- PK / UNIQUE ที่สะท้อน uniqueness ในโดเมน
- CHECK ที่สะท้อน invariant สำคัญ (เช่น `stock >= 0`)
- ไม่พึ่ง ORM tag — schema คือ source of truth ฝั่ง DB
- ใช้ `make migrate` / `make migrate-status` (ไม่รัน SQL ตรงด้วย `sqlite3`)
