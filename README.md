# dev-server


dev-server start --port 443 --supervise --init
ps aux | grep php | awk '{print $1}' | xargs -n1 kill -9
vim /usr/local/etc/php/conf.d/docker-php-ext-xdebug.ini

docker run -v ~/Projects/mill-prod:/var/www -v ~/Library/Application\ Support/mkcert:/root/.local/share/mkcert -it --rm -P test sh