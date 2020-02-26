#!/bin/bash

# Populate Vars
. ./VARS

VERSION_STRING=${APP_NAME}-${VERSION}
echo Building $VERSION_STRING

opt=$1

build_web() {
    {
        cd ./web &&
        npm run build &> /dev/null &&
        cd ../
    } || {
        return 1
    }
}

build_go () {
    {
        go build -ldflags "-X main.Version=$VERSION_STRING" -o bin/telescribe ./src/
    } || {
        return 1
    }
}

build_all () {
    {
        build_web &&
        build_go &&
        echo "Succesful"
    } || {
        return 1
    }
}

# Main
{
    case $opt in
    web)
        build_web
        ;;
    go)
        build_go
        ;;
    all|"")
        build_all
        ;;
    *|help)
    # Default
        echo "Available options are all, go, web"
        ;;
    esac
} || {
    echo "Failed to build!"
    exit 1
}