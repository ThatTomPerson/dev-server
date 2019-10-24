test:
	docker run -p 2000:443 -v $$PWD/sites:/var/www/sites -v ~/Library/Application\ Support/mkcert:/root/.local/share/mkcert ttpd/dev-server:v0