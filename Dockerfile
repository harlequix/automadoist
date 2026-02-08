# syntax=docker/dockerfile:1

# Build the application from source
FROM golang:1.25 AS build-stage

WORKDIR /app

COPY automadoist/go.mod automadoist/go.sum ./
RUN go mod edit -dropreplace github.com/harlequix/godoist && go mod tidy && go mod download

COPY automadoist/ ./
COPY godoist/ /godoist/
RUN go mod edit -replace github.com/harlequix/godoist=/godoist

RUN CGO_ENABLED=0 GOOS=linux go build -o /automadoist

# Run the tests in the container
FROM build-stage AS run-test-stage
RUN go test -v ./...

# Deploy the application binary into a lean image
FROM gcr.io/distroless/base-debian11 AS build-release-stage

WORKDIR /

COPY --from=build-stage /automadoist /automadoist

USER nonroot:nonroot

ENTRYPOINT ["/automadoist"]
