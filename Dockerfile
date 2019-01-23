FROM php:7.2-fpm-alpine
WORKDIR /var/www

RUN docker-php-ext-install pdo pdo_mysql

RUN apk add --no-cache --virtual .phpize-deps $PHPIZE_DEPS \
    && pecl install xdebug-2.6.1 \
    && docker-php-ext-enable xdebug \
    && apk del .phpize-deps;

COPY dev-server /usr/local/bin/

EXPOSE 443
CMD ["/usr/local/bin/dev-server", "start", "--port", "443", "--supervise", "--init"]