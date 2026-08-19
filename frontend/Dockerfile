# syntax=docker/dockerfile:1

FROM node:20-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

# Vite bakes every VITE_* variable into the built JS at build time, not
# read at container startup — so pointing this build at a non-default API
# needs a build arg, not a runtime env var. Override with
# `docker build --build-arg VITE_API_URL=... .` or docker-compose's
# `build.args` (see docker-compose.yml).
ARG VITE_API_URL=http://localhost:8080
ENV VITE_API_URL=$VITE_API_URL

RUN npm run build

FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
