# 1. เลือก Base Image
FROM golang:1.22-alpine


# 2. ตั้งค่า Working Directory ภายใน Container
WORKDIR /app

# 3. คัดลอกไฟล์ go.mod และ go.sum เพื่อติดตั้ง dependencies ก่อน
COPY go.mod go.sum ./

# 4. ติดตั้ง dependencies
RUN go mod download

# 5. คัดลอกโค้ดทั้งหมดที่เกี่ยวข้องเข้า Container
COPY . .

# คัดลอกไฟล์ .env เข้าไปใน Container
COPY .env .env

# 6. Build แอปพลิเคชัน Go
RUN go build -o main .

# 7. ระบุพอร์ตที่แอปจะรัน (ถ้าจำเป็น)
EXPOSE 8080

# 8. คำสั่งสำหรับเริ่มแอป
CMD ["./main"]
