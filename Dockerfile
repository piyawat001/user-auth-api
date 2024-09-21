# ใช้ Golang เป็น base image
FROM golang:1.22.6-alpine

# ตั้งค่า GOPATH และโฟลเดอร์สำหรับแอปพลิเคชัน
WORKDIR /app

# คัดลอกไฟล์ go.mod และ go.sum เพื่อดาวน์โหลด dependencies
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# คัดลอกโค้ดทั้งหมด
COPY . .

# สร้างแอปพลิเคชัน
RUN go build -o main .

# เปิดพอร์ตที่ Backend ใช้งาน (3000)
EXPOSE 3000

# คำสั่งสำหรับรันแอปพลิเคชัน
CMD ["./main"]
