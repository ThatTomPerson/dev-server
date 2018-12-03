FROM php:fpm-alpine
WORKDIR /var/www
COPY dev-server /usr/local/bin/
EXPOSE 443
CMD ["/usr/local/bin/dev-server", "-port", "443", "-start-fpm"]