FROM golang:1.21-bullseye as builder
WORKDIR /app
COPY . .
RUN go build -o /emass-promotion ./emass-promotion/cmd

FROM gcr.io/distroless/base-debian11
MAINTAINER "GitHub Expert Services"
LABEL org.opencontainers.image.source="https://github.com/department-of-veterans-affairs/codeql-tools-go"
LABEL org.opencontainers.image.description="GitHub Action for validating repositories meet CodeQL governance requirements."
COPY --from=builder /emass-promotion /
ENTRYPOINT ["/emass-promotion"]
