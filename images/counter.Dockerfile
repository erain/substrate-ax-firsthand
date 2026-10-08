ARG BASE_STATIC=gcr.io/distroless/static-debian13:latest@sha256:f2ea2709ac8db56323cbd7d014277f32cb572d9ea124b0076f7aafe5980678fe
FROM ${BASE_STATIC}
COPY bin/counter /app/counter
ENTRYPOINT ["/app/counter"]
