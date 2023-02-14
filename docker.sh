#!/bin/bash
function green_printf () 
{
printf "\e[32m$@\e[m\n"
}

function red_printf () {
  printf "\e[31m$@\e[m\n"
}

function panic () {
  echo
  red_printf "$@"
  echo
  exit 1
}

function panic_fail_op () {
  panic "Operation failed! Exit."
}

cd $(git rev-parse --show-toplevel)

function build() {
  green_printf "Before building, first cleaning... \n"
  clean || panic_fail_op
  green_printf "Building ownediparse image ... \n"
  docker build -t ownediparse-cli-server . || panic_fail_op
  green_printf "Launching ownediparse ... \n"
  docker run -d -p 8080:8080 ownediparse-cli-server || panic_fail_op
  green_printf "Showing ownediparse logs ... \n"
  docker ps --filter="ancestor=ownediparse-cli-server" --format="{{.ID}}" | xargs docker logs -f
}

function clean() {
  green_printf "Stopping/removing all ownediparse containers  ... \n"
  docker ps --filter="ancestor=ownediparse-cli-server" --format="{{.ID}}" | \
      xargs docker rm --force || true
  green_printf "Removing all ownediparse images  ... \n"
  docker images --filter "reference=ownediparse-cli-server:*" --format "{{.ID}}" | \
      xargs docker rmi --force || true
  green_printf "Cleaning complete.\n"
}

if [ "$1" = "build" ]; then
  CMD="build"
elif [ "$1" = "clean" ]; then
  CMD="clean"
elif [ -z "$1" ]; then
  CMD="clean"
else
  panic "Error: unknown arg '$1'" >&2
fi

if [ $CMD = 'build' ]; then
  build
else
  clean
fi
