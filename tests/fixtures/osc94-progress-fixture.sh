#!/usr/bin/env bash
# Phase 98 PRG-02/PRG-03 manual UAT + e2e fixture.
# Source format: ESC ] 9 ; 4 ; <state> ; <value> BEL  (verified ProgressAddon.ts:50)
set -euo pipefail
printf '\x1b]9;4;1;25\x07'; sleep 1
printf '\x1b]9;4;1;50\x07'; sleep 1
printf '\x1b]9;4;1;75\x07'; sleep 1
printf '\x1b]9;4;1;100\x07'; sleep 1
printf '\x1b]9;4;0\x07'  # clear
