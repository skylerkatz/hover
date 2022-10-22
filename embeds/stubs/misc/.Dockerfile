#
# Base
#
FROM --platform=linux/amd64 php:8.1-fpm-alpine as base

RUN apk --update add build-base libmcrypt-dev autoconf freetype-dev libjpeg-turbo-dev jpeg-dev libpng-dev imagemagick-dev \
                     libzip-dev gettext-dev libxslt-dev icu-dev libgcrypt-dev

RUN pecl channel-update pecl.php.net && pecl install mcrypt redis-5.3.7 && \
    docker-php-ext-install mysqli pdo_mysql pcntl bcmath zip intl gettext soap sockets xsl &&\
    docker-php-ext-enable redis &&\
    cp "/etc/ssl/cert.pem" /opt/cert.pem &&\
    curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer

COPY composer.json /var/task/composer.json

WORKDIR /var/task

RUN composer install --no-interaction --no-plugins --no-scripts --prefer-dist --no-autoloader

COPY . /var/task

RUN composer dump-autoload && \
    cp /var/task/.hover/out/application/hover_runtime/php.ini /usr/local/etc/php/php.ini

ENTRYPOINT []


#
# Assets
#
FROM node:16 as assets
COPY public /app/public
COPY package.json vite.config.js /app/
COPY resources /app/resources
WORKDIR /app
RUN npm install && npm run build


#
# Tests
#
FROM base as tests

CMD vendor/bin/phpunit

#
# Final
#
FROM base as final
RUN composer install --prefer-dist --no-interaction --no-dev --optimize-autoloader

RUN .hover/out/application/hover_runtime/prepare.sh

CMD /opt/bootstrap