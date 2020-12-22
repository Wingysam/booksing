#!/bin/bash

ulimit -S -n 1024

workingdir=$(mktemp -d)
mkdir "${workingdir}/import"

function log {
    echo "> $(date +%T) $*"
}

function cleanup {
    log "Removing old workdir"
    rm -rf "$workingdir"
    log "Done. ✅"
}

trap 'cleanup' EXIT


log "Creating temp workspace in ${workingdir}"
cp -a testdata/import $workingdir/import/

source .env

export BOOKSING_LOGLEVEL=debug
export BOOKSING_ADMINUSER='unknown'
export BOOKSING_DATABASEDIR="${workingdir}/db"
export BOOKSING_IMPORTDIR="${workingdir}/import"
export BOOKSING_FAILDIR="${workingdir}/failed"
export BOOKSING_BOOKDIR="${workingdir}/"
export BOOKSING_SAVEINTERVAL="20s"
export BOOKSING_MQTTENABLED=true
export BOOKSING_MQTTHOST="tcp://sanny.egdk.nl:1883"
export BOOKSING_MQTTTOPIC="events"
export BOOKSING_MQTTCLIENTID="booksing"
export BOOKSING_BINDADDRESS=":7133"


air
