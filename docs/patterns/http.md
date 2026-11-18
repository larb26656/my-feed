# HTTP Adapter Pattern

## โครง handler

```go
type Handler struct {
    svc *service.Service
}

func NewHandler(svc *service.Service) *Handler { ... }

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
    r.POST("/", h.create)
    r.POST("", h.create) // collection path without trailing slash
    r.GET("/:id", h.get)
    // command-style: POST /:id/publish
}
```

Bootstrap owns the prefix and uses the same registration as HTTP tests:

```go
producthttp.NewHandler(productService).RegisterRoutes(r.Group("/api/v1/products"))
```

Handlers use `*gin.Context` for parameters and pass `c.Request.Context()` to services. Use `platform/http.NewRouter` for request IDs, recovery, untrusted forwarded headers and compatible routing responses.

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
- Import adapter/sqlite
