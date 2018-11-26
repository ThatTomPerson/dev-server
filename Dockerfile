FROM php:fpm-alpine

COPY dev-server /usr/local/bin/

EXPOSE 443
CMD ["/usr/local/bin/dev-server", "-port", "443", "-start-fpm"]