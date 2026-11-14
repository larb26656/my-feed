# Domain Pattern

## หลักการ

1. **Aggregate root** เป็นเจ้าของ state และ transition
2. สร้างผ่าน factory (`New…`) ที่ validate ทันที
3. เปลี่ยน state ผ่าน **method ที่มีชื่อทางธุรกิจ** เท่านั้น — ไม่มี public setter สำหรับ status / stock / usage ที่ข้าม rule
4. Error ใช้ `platform/apperr` (`Validation`, `FailedPrecondition`, …) ไม่ใช่ `fmt.Errorf` เปล่าๆ สำหรับเคสธุรกิจ

## โครง type มาตรฐาน

```go
type Status string

const (
    StatusDraft     Status = "DRAFT"
    StatusAvailable Status = "AVAILABLE"
)

type Product struct {
    ID        uuid.UUID
    // ... fields
    Status    Status
    CreatedAt time.Time
    UpdatedAt time.Time
}

func NewProduct(/* required attrs */) (*Product, error) {
    // validate → return apperr.Validation(...)
    // set initial status, timestamps, id
}

func (p *Product) Publish() error {
    if p.Status != StatusDraft {
        return apperr.FailedPrecondition("only DRAFT products can be published")
    }
    p.Status = StatusAvailable
    p.UpdatedAt = time.Now().UTC()
    return nil
}
```

## Checklist ตอนเขียน domain

- [ ] Invariant อ่านจาก method ได้โดยไม่ต้องเปิด SQL
- [ ] State machine ชัด (ค่าคงที่ status + เงื่อนไขก่อนเปลี่ยน)
- [ ] Money เป็นจำนวนเต็มหน่วยย่อย (`int64`) ถ้ามีราคา/ส่วนลด
- [ ] Snapshot ฟิลด์ที่ต้องคงประวัติ (เช่น order line `unit_price`) อยู่ที่ aggregate ปลายทาง
- [ ] มี unit test ใน `domain/*_test.go` ครอบคลุม happy path + transition ผิด

## สิ่งที่ domain ไม่ทำ

- ไม่เปิด transaction
- ไม่เรียก repository
- ไม่ map JSON tag เพื่อ HTTP (persistence mapping อยู่ adapter)
- ไม่ “เผื่อ” event bus ถ้ายังไม่มี requirement
