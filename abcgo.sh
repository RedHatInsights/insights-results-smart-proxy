#!/bin/bash
# Copyright 2020 Red Hat, Inc
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

threshold=64

BLUE=$(tput setaf 4)
RED_BG=$(tput setab 1)
GREEN_BG=$(tput setab 2)
NC=$(tput sgr0) # No Color

VERBOSE=false

if [[ $* == *verbose* ]]; then
    VERBOSE=true
fi

if ! [ -x "$(command -v abcgo)" ]
then
    echo -e "${BLUE}Installing abcgo${NC}"
    if ! go install github.com/droptheplot/abcgo@latest; then
        echo -e "${RED_BG}[FAIL]${NC} Cannot install abcgo!"
        exit 1
    fi
    echo -e "${BLUE}Installed ${NC}"
fi

if [ "$VERBOSE" = true ]; then
    echo -e "${BLUE}All ABC metrics${NC}:"
    abcgo -path .
    echo -e "${BLUE}Functions with ABC metrics greater than ${threshold}${NC}:"
fi

if [[ $(abcgo -path . -sort -format raw | awk "\$4>${threshold}" | tee /dev/tty | wc -l) -ne 0 ]]
then
    echo -e "${RED_BG}[FAIL]${NC} Functions with too high ABC metrics detected!"
    exit 1
else
    echo -e "${GREEN_BG}[OK]${NC} ABC metrics are ok for all functions in all packages"
    exit 0
fi

