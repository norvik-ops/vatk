# Security headers for nginx-prod.conf.
#
# Included into each location block because nginx does NOT inherit
# server-block `add_header` directives into a location block that
# declares its own `add_header`. Without this include, locations like
# `= /index.html` (which set Cache-Control) would silently drop the
# Content-Security-Policy and the rest of the security headers from
# the parent server block.
#
# Keep this file in sync with caddy/Caddyfile's secdemo block in the
# vakt-server repo — the live demo runs both layers.

add_header X-Frame-Options "DENY" always;
add_header X-Content-Type-Options "nosniff" always;
add_header X-XSS-Protection "0" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
add_header Permissions-Policy "camera=(), microphone=(), geolocation=()" always;
add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self'; connect-src 'self'; frame-ancestors 'none';" always;
