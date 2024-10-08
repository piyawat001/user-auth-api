# Build stage
FROM golang:1.22.6 
# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY backend/user-auth-api/go.mod backend/user-auth-api/go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code into the container
COPY backend/user-auth-api .

# Build the Go app
RUN go build -o main .

# Expose port 8080 to the outside world
EXPOSE 3000

# Command to run the executable
CMD ["./main"]