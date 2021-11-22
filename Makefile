prefix=/usr/local

build:
	go build -o bin/composer

install: build
	install bin/composer ${prefix}/bin/composer
