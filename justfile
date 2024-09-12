#!/usr/bin/env just --justfile

help:
  @just --list

format:
    pnpm run format
    TEMPL_EXPERIMENT=rawgo templ fmt .
    gofmt -w .

test:
    go test ./... -v

clean:
    rm view/**/*_templ.go

run:
    @just format
    TEMPL_EXPERIMENT=rawgo templ generate
    pnpm run build
    go run ./cmd/app --port 4321

dev:
    air
