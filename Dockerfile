FROM golang:1.19 AS builder

WORKDIR /home/app
COPY Makefile /home/app/
COPY go.* /home/app/
COPY pkg /home/app/pkg
COPY main.go /home/app/
RUN make build

#  Distroless
FROM gcr.io/distroless/base as runtime
COPY --from=builder /home/app/microvm-action-runner /microvm-action-runner

ENTRYPOINT ["/microvm-action-runner"]
