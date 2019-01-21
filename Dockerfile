FROM php:7.2-fpm-alpine
WORKDIR /var/www

RUN docker-php-ext-install pdo pdo_mysql

COPY dev-server /usr/local/bin/

EXPOSE 443
CMD ["/usr/local/bin/dev-server", "start", "--port", "443", "--supervise", "--init"]