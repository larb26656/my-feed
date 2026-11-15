# Patterns — Go Modular Monolith (Hexagonal)

คู่มือ pattern จากโปรเจกต์นี้ สำหรับนำไป **init โปรเจกต์ใหม่** หรือ scaffold module ใหม่โดยไม่ต้องเดาโครงสร้าง

อ้างอิงเชิงลึกด้านโดเมน: [`docs/adr/0001-architecture.md`](../adr/0001-architecture.md)

## ใช้ยังไงตอน init โปรเจกต์ใหม่

1. คัดลอกโฟลเดอร์ `docs/patterns/` ไปโปรเจกต์เป้าหมาย (หรือ clone แล้วลบโดเมน commerce ออก)
2. ทำตาม [`init-checklist.md`](./init-checklist.md)
3. เวลาเพิ่ม bounded context / module ใหม่ ใช้ [`module-scaffold.md`](./module-scaffold.md) เป็นแม่แบบ
4. เขียนโค้ดตาม layer docs ด้านล่าง — อย่าข้าม dependency rule

## เอกสารในชุดนี้

| ไฟล์ | เนื้อหา |
|------|---------|
| [`architecture.md`](./architecture.md) | ภาพรวม, dependency direction, สิ่งที่ทำ / ไม่ทำ |
| [`module-scaffold.md`](./module-scaffold.md) | โครงโฟลเดอร์ต่อ module + หน้าที่แต่ละแพ็กเกจ |
| [`domain.md`](./domain.md) | Aggregate, invariants, ไม่มี public setter ข้าม domain method |
| [`application.md`](./application.md) | Module service vs cross-module use case |
| [`persistence.md`](./persistence.md) | Repository port, explicit ops, `WithTx`, conditional update |
| [`http.md`](./http.md) | Chi handler, DTO, `apperr` → HTTP |
| [`platform.md`](./platform.md) | Shared kernel: database, apperr, http helpers |
| [`init-checklist.md`](./init-checklist.md) | Checklist เริ่มโปรเจกต์ว่าง |

## Stack อ้างอิง (จาก POC นี้)

- Go + `chi` + `pgx` (ไม่มี ORM)
- PostgreSQL + Goose SQL migrations (`migrations/`)
- Money / จำนวนเงินเป็น `int64` (หน่วยย่อย) — ไม่ใช้ float
- Modular monolith — ไม่แยก microservice, ไม่มี message broker ใน POC
