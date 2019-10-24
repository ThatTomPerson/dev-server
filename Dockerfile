FROM php:7.2-fpm-alpine
WORKDIR /var/www

RUN apk add --virtual .phpize-deps $PHPIZE_DEPS \
    && pecl install xdebug-2.6.1 \
    && docker-php-ext-enable xdebug \
    && apk del .phpize-deps \
    && apk add make git libzip-dev \
    && rm -rf /var/cache/apk/*;

RUN docker-php-ext-install pdo pdo_mysql zip opcache

RUN printf "max_execution_time = 0\n\n" > /usr/local/etc/php/conf.d/docker-php-development.ini

COPY dev-server /usr/local/bin/

EXPOSE 443
CMD ["/usr/local/bin/dev-server", "-port", "443", "-supervise", "-init"]

