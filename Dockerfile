# Use the official Go image as a parent image
FROM golang:1.21-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the application
RUN go build -o /ric ./cmd/ric

# Expose the ports for the A1 and E2 interfaces
EXPOSE 8080 38484 830

# Run the application
CMD [ "/ric" ]
