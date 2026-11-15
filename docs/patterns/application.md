# Application Service Pattern

มีสองแบบ — อย่าปนบทบาท

## 1) Module service

อยู่ที่ `internal/{module}/service`

**หน้าที่:** load aggregate → เรียก domain method → persist

```go
type Service struct {
    repo repository.Repository
}

func (s *Service) Publish(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
    p, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }
    if err := p.Publish(); err != nil {
        return nil, err
    }
    if err := s.repo.Update(ctx, p); err != nil {
        return nil, err
    }
    return p, nil
}
```

**Input types:** ประกาศในแพ็กเกจ service (`CreateInput`, …) — handler map จาก JSON เข้ามา ไม่ให้ domain พึ่ง HTTP DTO

## 2) Cross-module use case

อยู่ที่ `internal/{usecase}/service` (เช่น `checkout`)

**หน้าที่:** orchestrate หลาย port ใน **transaction เดียว**

```go
err := s.db.WithinTransaction(ctx, func(ctx context.Context, tx database.Transaction) error {
    products := s.products.WithTx(tx)
    coupons := s.coupons.WithTx(tx)
    orders := s.orders.WithTx(tx)

    // 1) load / validate ผ่าน domain หรือ repo ops
    // 2) side effects ที่ต้อง atomic (ReserveStock, Redeem, Create order)
    // 3) ถ้า step ใด fail → return error → rollback ทั้งก้อน
    return nil
})
```

### กฎ orchestration

| ทำ | ไม่ทำ |
|----|------|
| Inject **ports** ของ module ที่เกี่ยวข้อง | Import `adapter/sqlite` ของ module อื่น |
| ใช้ `WithTx(tx)` ทุก repo ใน tx | Commit เองใน service ย่อย |
| Idempotency เช็กก่อน/ใน tx ถ้ามี key | กลืน error แล้ว partial success |
| Domain ตรวจ “ทำได้ไหม” ก่อนเขียน | เขียน SQL ใน use case |

## Idempotency (ถ้า use case ต้องการ)

1. ถ้ามี key → อ่าน existing ก่อนทำงานหนัก
2. ใน tx ใส่ UNIQUE ที่ DB
3. ชน unique → ถือว่าแข่งกันได้; ฝ่ายแพ้ rollback แล้วอ่านของฝ่ายชนะกลับ

อ้างอิงพฤติกรรมจริง: checkout ใน POC + ADR 0001

## Cancel / compensation

ถ้า use case สร้าง side effect หลาย aggregate (stock, coupon, order) การย้อนกลับต้องเป็น **use case คนละอัน** ที่ restitution ใน tx เดียวกัน — ไม่พึ่ง “ลบแถวแล้วจบ” โดยไม่คืน invariant
