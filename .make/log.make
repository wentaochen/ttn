# This makefile has tools for logging useful information
# about build steps

LOG_NAME = $(shell basename $$(pwd))

log = echo -e "\033[1;34m`basename $(LOG_NAME)` \033[0m"

# vim: ft=make
