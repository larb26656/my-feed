# Platform (Shared Kernel)

แพ็กเกจกลางที่ทุก module ใช้ร่วมได้ — **บางและไม่มี business rule ของโดเมนใดโดเมนหนึ่ง**

```
internal/platform/
  apperr/       # typed application errors
  database/     # SQLite database/sql connection + WithinTransaction
  http/         # DecodeJSON, WriteJSON, WriteError
```

## `apperr`

See the [application error pattern](./errors.md) for shared and module-specific error definitions.

## `database`

- `NewPool(ctx, databasePath)` — เปิด SQLite file, ping, ตั้ง busy timeout และ foreign keys
- `WithinTransaction(ctx, fn)` — begin → fn → commit; error หรือ panic path ผ่าน rollback ด้วย `defer`
- Cross-module use case เป็นผู้เรียก `WithinTransaction` แล้วส่ง `tx` เข้า `WithTx`

## `http` (package `httpx`)

- ไม่ผูกกับ module
- `WriteError` ใช้ `errors.As` หา `*apperr.Error`
- ไม่ log รายละเอียดที่นี่ถ้ายังไม่มี requirement — เก็บให้บาง

## Bootstrap vs platform

| `platform` | `bootstrap` |
|------------|-------------|
| primitive ใช้ซ้ำได้ | ประกอบกราฟ dependency ของแอปนี้ |
| ไม่รู้จัก Product/Order | สร้าง repo/service/handler และ mount routes |

`cmd/api/main.go` เหลือแค่ signal context + เรียก `bootstrap.Run`
