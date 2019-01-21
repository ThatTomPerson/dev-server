FROM php:fpm-alpine
WORKDIR /var/www
COPY dev-server /usr/local/bin/

RUN docker-php-ext-install pdo pdo_mysql

EXPOSE 443
CMD ["/usr/local/bin/dev-server", "start", "--port", "443", "--supervise", "--init"]