# Use an official Go image as the base image for building
FROM golang:1.23-alpine AS builder 

# Set the working directory
WORKDIR /app

# Copy binary, assets, dan buat folder uploads
COPY --from=builder /app/student-chaincode .
COPY static/ ./static/
RUN mkdir -p ./uploads  # Buat folder uploads

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go application
RUN go build -o main .

# Use a minimal image for the final deployment
FROM gcr.io/distroless/base-debian11

# Set the working directory in the container
WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /app/main ./

# Expose the port on which the application will run
EXPOSE 8000

# Command to run the application
CMD ["./main"]
