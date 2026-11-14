# Init Checklist — โปรเจกต์ใหม่จาก pattern นี้

ใช้เมื่อเริ่ม repo ว่าง หรือตัดโดเมนตัวอย่างออกจาก POC นี้

## 0) ตัดสินใจก่อนลงมือ

- [ ] ชื่อ module แรก (bounded context) และ aggregate root
- [ ] มี cross-module use case ตั้งแต่วันแรกไหม — ถ้ามี ตั้งชื่อแพ็กเกจ use case แยก
- [ ] Module path: `github.com/{you}/{project}` ใน `go.mod`
- [ ] ยืนยันขอบเขตที่ตัดออกตาม [`architecture.md`](./architecture.md) (ไม่มี ORM / broker ถ้ายังไม่ต้อง)

## 1) Skeleton โครงสร้าง

```
cmd/api/main.go
internal/bootstrap/app.go
internal/platform/{apperr,database,http}/
internal/{module}/{domain,repository,service,adapter/http,adapter/postgres}/
migrations/
scripts/migrate.sh
api/openapi.yaml          # optional แต่แนะนำ
tests/integration/        # เมื่อมี flow ข้าม aggregate + DB
docker-compose.yml
Makefile
docs/adr/                 # ตัดสินใจใหญ่คัดลอก/เขียนใหม่
docs/patterns/            # คัดลอกชุดนี้มาทั้งก้อน
```

- [ ] `go mod init …`
- [ ] คัดลอกหรือเขียนใหม่ `platform` ให้ครบ 3 แพ็กเกจ
- [ ] `cmd/api` + `bootstrap` ว่างๆ ที่ขึ้น `:8080` + `/healthz` ได้
- [ ] Docker Postgres + `DATABASE_URL` + migrate script

## 2) Module แรก

ทำตาม [`module-scaffold.md`](./module-scaffold.md):

- [ ] Domain + unit tests
- [ ] Repository port + `WithTx`
- [ ] Postgres adapter + migration (UNIQUE/CHECK ตาม invariant)
- [ ] Module service (load → domain → persist)
- [ ] HTTP handler + mount ใน bootstrap
- [ ] เส้น happy path ทดสอบด้วย `go test ./internal/...` และยิง API มือหรือ integration test

## 3) Cross-module (เมื่อถึงเวลา)

- [ ] แพ็กเกจ use case แยก — inject ports เท่านั้น
- [ ] ทุก side effect ใน `WithinTransaction` + `WithTx`
- [ ] Conditional UPDATE / UNIQUE สำหรับแข่งกัน
- [ ] Idempotency ถ้า client retry ได้
- [ ] Integration test กับ Postgres จริง

## 4) คุณภาพขั้นต่ำก่อนถือว่า “init เสร็จ”

- [ ] Dependency direction ไม่พัง (domain ไม่ import adapter)
- [ ] ไม่มี module import `…/adapter/postgres` ของ module อื่น
- [ ] Error ธุรกิจผ่าน `apperr` และ HTTP map ได้
- [ ] README โปรเจกต์ใหม่ชี้มาที่ `docs/patterns/` + ADR ของตัวเอง
- [ ] `make test-unit` ผ่านโดยไม่ต้องพึ่ง Docker; integration แยกชัด

## สิ่งที่คัดลอกจาก POC นี้ได้ตรงๆ

เหมาะ copy แล้วเปลี่ยนชื่อ:

- `internal/platform/**`
- โครง `bootstrap` / `cmd/api`
- `docker-compose.yml`, `scripts/migrate.sh`, `Makefile`
- ชุด `docs/patterns/**`

ไม่เหมาะ copy ตรงๆ โดยไม่คิด:

- ตาราง / domain ของ Product–Coupon–Order (เป็นตัวอย่างโดเมน ไม่ใช่ template ธุรกิจ)
- ชื่อ error code เฉพาะ commerce (`insufficient_stock`, …) — เก็บเฉพาะที่โปรเจกต์ใหม่ใช้จริง
