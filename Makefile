prefix=/usr/local

build:
	go build -o bin/composer

install: build
	install -o0 -g0 bin/composer ${prefix}/bin/composer
