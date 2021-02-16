FROM golang as builder
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /seeder services/firestore/seeder.go

FROM gcr.io/google.com/cloudsdktool/cloud-sdk:latest
COPY --from=builder /seeder /seeder
COPY ./games/metadata-seed.yaml /metadata-seed.yaml
COPY ./services/firestore/entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh", "/metadata-seed.yaml"]
