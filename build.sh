#!/bin/bash

opt=$1

compile_sass () {
    {
        fn=$1 &&
        # Getting basename
        # https://stackoverflow.com/questions/2664740/extract-file-basename-without-path-and-extension-in-bash/36341390
        base=${fn%.scss} &&
        sass --no-source-map ${base}.scss ${base}.css
    } || {
        return 1
    }
}
export -f compile_sass # so that xargs can access

build_static() {
    {
        # Accessing function in xargs
        # https://stackoverflow.com/questions/11003418/calling-shell-functions-with-xargs
        find static/ -name '*.scss' | xargs -I {} bash -c 'compile_sass "$@"' _ {}
    } || {
        return 1
    }
}

build_go () {
    {
        go build -o bin/telescribe ./src/
    } || {
        return 1
    }
}

build_all () {
    {
        build_static &&
        build_go
    } || {
        return 1
    }
}

# Main
{
    case $opt in
    static)
        build_static
        ;;
    go)
        build_go
        ;;
    all|"")
        build_all
        ;;
    *|help)
    # Default
        echo "Available options are all, go, static"
        ;;
    esac
} || {
    echo "Failed to build!"
    exit 1
}