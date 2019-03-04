FROM php:7.2-fpm-alpine
WORKDIR /var/www

RUN docker-php-ext-install pdo pdo_mysql

RUN apk add --virtual .phpize-deps $PHPIZE_DEPS \
    && pecl install xdebug-2.6.1 \
    && docker-php-ext-enable xdebug \
    && apk del .phpize-deps \
    && apk add make git \
    && rm -rf /var/cache/apk/*;

RUN apk add --no-cache make git

RUN printf "max_execution_time = 0\n" > /usr/local/etc/php/conf.d/docker-php-development.ini

COPY dev-server /usr/local/bin/

EXPOSE 443
CMD ["/usr/local/bin/dev-server", "start", "--port", "443", "--supervise", "--init"]

