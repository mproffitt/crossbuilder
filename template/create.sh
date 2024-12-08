#!/usr/bin/env bash

BASE_PATH=""

if [ -z "${BASE_PATH}" ]; then
    echo "Base path is empty - please edit this script to set BASE_PATH"
    echo "to the location of your APIs folder"
    exit 1
fi

RED=$(echo $'\033[31m');
GREEN=$(echo $'\033[32m');
YELLOW=$(echo $'\033[33m');
BLUE=$(echo $'\033[34m');
MAGENTA=$(echo $'\033[35m');
CYAN=$(echo $'\033[36m');
WHITE=$(echo $'\033[37m');

RESET=$(echo $'\033[00m');

BOLD=$(tput bold);
NORMAL=$(tput sgr0)

function inform()
{
    if [ "$1" = '-n' ] ; then
        shift
        echo -n "$WHITE[INFO]$RESET $@" 1>&2;
    else
        echo "$WHITE[INFO]$RESET $@" 1>&2;
    fi
}

function question()
{
    local message="$@"
    shift
    inform -n "$message > "
    read answer
    if [ -z $answer ]; then
        answer="$(question $message)"
    fi
    echo "$answer"
}

moduleroot ()
{
    local wd="$(pwd)";
    while [ ! -d ".git" ] && [ "$(pwd)" != "/" ]; do
        cd ..;
    done;
    local moduleName=$(basename `pwd`);
    if [ "$moduleName" = "/" ]; then
        error "Cannot find root directory for current module. Are you sure it's a git repository?" 1>&2;
        cd "$wd";
        return 1;
    fi;
    return 0
}

moduleroot || exit 1

REPO_NAME="$(git config remote.origin.url | sed 's|git@||;s|:|/|g;s|.git||')"
GROUP_NAME=$(question "Enter the group name" | tr '[:upper:]' '[:lower:]')
COMPOSITION=$(question "Enter the composition name (lowercase, hyphenated)")
GROUP_CLASS=$(question "Enter the group class (camel-cased struct name)")

# Make sure at least the first letter is uppercase so go can export it
GROUP_CLASS="${GROUP_CLASS^}"
group_class_lower=$(tr '[:upper:]' '[:lower:]' <<< $GROUP_CLASS)

inform "creating directories"
mkdir -p {apis,apidocs,hack,${BASE_PATH}/${GROUP_NAME}/{compositions/${COMPOSITION}/templates,v1alpha1,docs,examples}}

inform "templating generate.go"
sed -e "s|<GROUP_NAME>|${GROUP_NAME}|g" \
    -e "s|<BASE_PATH>|${BASE_PATH}|g" \
    template/files/generate.go.tpl > ${BASE_PATH}/${GROUP_NAME}/generate.go

inform "templating main.go"

sed -e "s|<GROUP_NAME>|${GROUP_NAME}|g" \
    -e "s|<GROUP_CLASS>|${GROUP_CLASS}|g" \
    -e "s|<COMPOSITION>|${COMPOSITION}|g" \
    -e "s|<BASE_PATH>|${BASE_PATH}|g" \
    -e "s|<REPO_NAME>|${REPO_NAME}|g" \
    template/files/main.go.tpl > ${BASE_PATH}/${GROUP_NAME}/compositions/${COMPOSITION}/main.go

if [ ! -f ${BASE_PATH}/${GROUP_NAME}/v1alpha1/doc.go ]; then
    inform "templating doc.go"
    sed -e "s|<GROUP_NAME>|${GROUP_NAME}|g" \
        -e "s|<BASE_PATH>|${BASE_PATH}|g" \
        -e "s|<REPO_NAME>|${REPO_NAME}|g" \
        template/files/doc.go.tpl > ${BASE_PATH}/${GROUP_NAME}/v1alpha1/doc.go
fi

if [ ! -f ${BASE_PATH}/${GROUP_NAME}/v1alpha1/groupversion.go ]; then
    inform "templating groupversion.go"
    sed -e "s|<GROUP_NAME>|${GROUP_NAME}|g" \
        -e "s|<GROUP_CLASS>|${GROUP_CLASS}|g" \
        -e "s|<BASE_PATH>|${BASE_PATH}|g" \
        -e "s|<REPO_NAME>|${REPO_NAME}|g" \
        template/files/groupversion.go.tpl > ${BASE_PATH}/${GROUP_NAME}/v1alpha1/groupversion.go
else
    if ! grep -q "${GROUP_CLASS}List" ${BASE_PATH}/${GROUP_NAME}/v1alpha1/groupversion.go; then
        inform "updating groupversion.go"
        schema="SchemeBuilder.Register(\&${GROUP_CLASS}\{\}, \&${GROUP_CLASS}List\{\})"
        sed -i "s|func init() {|func init() {\n\t$schema|" ${BASE_PATH}/${GROUP_NAME}/v1alpha1/groupversion.go
    fi
fi

if [ ! -f "${BASE_PATH}/${GROUP_NAME}/v1alpha1/${group_class_lower}_types.go" ]; then
    inform "templating ${group_class_lower}_types.go"
    SHORTNAME=$(question "Enter a shortname for the XRD type" | tr '[:upper:]' '[:lower:]')
    ENFORCE_COMPOSITION=$(question "Enforce composition? (yes/no)" | tr '[:upper:]' '[:lower:]')
    sed -e "s|<GROUP_NAME>|${GROUP_NAME}|g" \
        -e "s|<GROUP_CLASS_LOWER>|${GROUP_CLASS,,}|g" \
        -e "s|<SHORTNAME>|${SHORTNAME}|g" \
        -e "s|<COMPOSITION>|${COMPOSITION}|g" \
        -e "s|<BASE_PATH>|${BASE_PATH}|g" \
        -e "s|<REPO_NAME>|${REPO_NAME}|g" \
        template/files/xrd.go.tpl > ${BASE_PATH}/${GROUP_NAME}/v1alpha1/${group_class_lower}_types.go

    if [ "$ENFORCE_COMPOSITION" = "no" ]; then
        sed -i '/.*enforcedCompositionRef.*/d' ${BASE_PATH}/${GROUP_NAME}/v1alpha1/${group_class_lower}_types.go
    fi
fi

if [ ! -f hack/boilerplate.go.txt ]; then
    inform "copying boilerplate.go.txt to hack directory for autogen headers"
    cp template/hack/boilerplate.go.txt hack
fi

if [ ! -f go.mod ]; then
    inform "setting up go.mod with ${REPO_NAME}"
    go mod init ${REPO_NAME}
    go mod tidy
fi

if [ ! -f Makefile ]; then
    inform "copying Makefile"
    cp template/Makefile Makefile
fi

inform "Running make"
make

