#!/bin/sh
# Copyright (C) 2016  Arista Networks, Inc.
# Use of this source code is governed by the Apache License 2.0
# that can be found in the COPYING file.

DEFAULT_PORT=6042

set -e

if [ "$#" -lt 1 ]
then
   echo "usage: $0 <host> [<port>]"
   exit 1
fi

host=$1
port=$DEFAULT_PORT
if [ "$#" -gt 1 ]
then
   port=$2
fi
iptables="bash sudo iptables -A INPUT -p tcp --dport $port -j ACCEPT"
ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no $host "$iptables"
echo "opened TCP port $port on $host"
