# HTTP Adapter Pattern

## โครง handler

```go
type Handler struct {
    svc *service.Service
}

func NewHandler(svc *service.Service) *Handler { ... }

func (h *Handler) Routes() chi.Router {
    r := chi.NewRouter()
    r.Post("/", h.create)
    r.Get("/{id}", h.get)
    // command-style: POST /{id}/publish
    return r
}
```

Bootstrap เป็นคน `Mount` path prefix:

```go
r.Mount("/api/v1/products", producthttp.NewHandler(productService).Routes())
```

## Request / response

- Request DTO ในไฟล์ handler (`createRequest`) พร้อม `json` tags
- Map → `service.CreateInput` (ไม่ส่ง DTO เข้า domain)
- Response DTO แยก (`productResponse`) — แปลงจาก domain ใน `toResponse`
- Decode ด้วย `platform/http.DecodeJSON` (`DisallowUnknownFields`)
- สำเร็จ: `WriteJSON` + status ที่เหมาะสม (`201` สร้างใหม่, `200` อ่าน/อัปเดต)
- ล้มเหลว: `WriteError` — แปลงจาก `apperr.Error`

## Routing style

| แบบ | ใช้เมื่อ |
|-----|---------|
| `POST /` | สร้าง resource |
| `GET /{id}` | อ่าน |
| `PATCH /{id}` | partial update ที่ domain รองรับ |
| `POST /{id}/{action}` | คำสั่งธุรกิจ (`publish`, `archive`, `cancel`) |

อย่าซ่อน state transition ไว้ใน `PUT` ทั้งก้อนถ้าโดเมนมีคำสั่งชัดเจน

## Error → HTTP (ผ่าน apperr)

| Code | HTTP (โดยประมาณ) |
|------|------------------|
| `validation` | 400 |
| `not_found` | 404 |
| `conflict` / `idempotency_conflict` | 409 |
| `failed_precondition` / business exhausted | 422 |
| อื่นๆ ที่ไม่รู้จัก | 500 (ข้อความไม่รั่วรายละเอียดภายใน) |

Body มาตรฐาน:

```json
{ "error": "message", "code": "validation" }
```

## Handler ไม่ทำ

- เปิด DB transaction
- ตรวจ invariant ที่ซ้ำกับ domain (ยกเว้น parse UUID / รูปแบบ request ตื้นๆ)
- Import adapter postgres
