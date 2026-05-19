FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# One-shot stage: copy built assets to the shared volume at /app/dist and exit.
# The nginx reverse-proxy service serves them from the frontend_dist volume.
FROM alpine:3.20
COPY --from=builder /app/dist /dist
CMD ["sh", "-c", "cp -r /dist/. /app/dist/ && echo 'frontend assets ready'"]
