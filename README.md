Composer
========

Composer is a simple service manager for dev environments.

How to build/install it?
----------------

To build `composer` under `./bin`, run:

    make build

To build and install `composer` in the system (under `/usr/local/bin`), run:

    sudo make install

How to configure?
-----------------

With a `compose.yml` file:

```yaml
# version of the config file syntax
version: 1

# define global environment variables accessible by all services
# Following variables are always present:
# $HOME - points to user's home directory
# $PATH - contains colon-delimited paths where executables can be found
# $PWD - project's working directory (where the compose.yml is located)
environment:
  - KEY1: value1
  - KEY2: ${OSVAL} # ${OSVAL} allows referencing composer's own environment

services:
  service1:
    # define environment variables to be used by the service 
    environment:
      - KEY1: value1
      - KEY2: ${OSVAL} # ${OSVAL} allows referencing composer's own environment

    # depends_on defines service dependencies.
    # All dependencies will be started and ready before this service's command is executed.
    depends_on:
      - service2

    # command to be executed to run the service (it's possible to use defined environmental variables)
    command: go run main.go ${KEY1}

  service2:
    # ready_on defines a text which is expected on stdout/stderr when the service is ready.
    # When ready_on is not provided, service is considered ready immediately after executing its command.
    ready_on: "I'm ready"

    # workdir defines a working directory (absolute or relative to current working directory)
    # where the command will be executed.
    workdir: src/
    command: echo I'm ready
```

How to run it?
--------------

With a composer binary:

`composer SERVICE`

It's possible to define multiple config files with different names, so to use a non-default (`composer.yml`) file, one
must define an environmental variable `COMPOSER_FILE`, i.e.:

`COMPOSER_FILE=custom-composer.yml composer SERVICE`
