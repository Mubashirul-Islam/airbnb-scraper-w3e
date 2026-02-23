#!/bin/bash
set -e

go mod download
docker compose up -d postgres
go run .
