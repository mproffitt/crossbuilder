#!/usr/bin/env bash
BASE_PATH=""

SUBMODULE_URL="https://github.com/mproffitt/crossbuilder.git"

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
CROSSPLANE_VERSION="1.17.0"
VERSION="v0.0.1"

function inform()
{
    if [ "$1" = '-n' ] ; then
        shift
        echo -n "$WHITE[INFO]$RESET $@" 1>&2;
    else
        echo "$WHITE[INFO]$RESET $@" 1>&2;
    fi
}

function error()
{
    echo "$RED[ERROR]$RESET $@" 1>&2;
}

function warn()
{
    echo "$YELLOW[WARN]$RESET $@" 1>&2;
}

function question()
{
    local message="$1"
    shift
    local options="$@"
    if [ -n "${options}" ]; then
        message="$message [${options// /|}]"
        options="${options// /\\|}"
    else
        options=".*"
    fi

    inform -n "$message > "
    # use </dev/tty to ensure read has a tty when
    # this script is run in a pipe
    read -er answer </dev/tty
    if [ -z "$answer" ] || ! grep -qi "${options}" <<< "${answer}"; then
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
        warn "Cannot find root directory for current module." 1>&2;
        cd "$wd";
        return 1;
    fi;
    return 0
}

moduleroot || (
    ans=$(question "Create new git repository?" "yes" "no")
    ans="${ans,,}"
    if [ "${ans:0:1}" = "n" ]; then
        exit 0
    fi

    git init
    git submodule add ${SUBMODULE_URL}
    git submodule init

    url=$(question "Enter the git remote URL (e.g. git@github.com:example/repo.git)")
    git remote add origin $url

    if [ ! -f .gitignore ]; then
        cp crossbuilder/template/files/gitignore .gitignore
    fi

    git add .gitignore

    if [ ! -f README.md ]; then
        echo '# `'$(basename $(pwd))'`' > README.md
    fi
    git add README.md

    git commit -m "Initial commit"
)

sleep 1
crossbuilder_path=$(
    git submodule foreach --quiet 'echo $(git config remote.origin.url) $path' | \
        grep 'crossbuilder.git' | awk '{print $2}'
)
echo "CROSSBUILDER found in ${crossbuilder_path}"

if [ ! -d template ]; then
    echo "Setting up the template directory for the first time"
    cp -r ${crossbuilder_path}/template .
    cp ${crossbuilder_path}/setup.sh template/create.sh
    base_path=$(question "please enter the api extension (e.g. crossplane.example.com)")
    sed -i "s|^BASE_PATH=.*|BASE_PATH='${base_path}'|" template/create.sh
fi

if [ ! -f Makefile ]; then
    inform "copying Makefile"
    cp ${crossbuilder_path}/template/files/Makefile Makefile
fi

if [ ! -f Dockerfile ]; then
    inform "copying Dockerfile"
    cp ${crossbuilder_path}/template/files/Dockerfile Dockerfile
fi

echo Running $0
if grep -q 'setup.sh\|bash' <<< $0 ; then
    make create
    exit $?
fi

if [ -z "${BASE_PATH}" ]; then
    echo "Base path is empty - please edit this script to set BASE_PATH"
    echo "to the location of your APIs folder"
    exit 1
fi

REPO_NAME="$(git config remote.origin.url | sed 's#\(https://\|git@\)##;s#:#/#g;s#.git##')"
OWNER=$(echo $REPO_NAME | cut -d'/' -f2)
GROUP_NAME=$(question "Enter the group name" | tr '[:upper:]' '[:lower:]')
COMPOSITION=$(question "Enter the composition name (lowercase, hyphenated)")
GROUP_CLASS=$(question "Enter the group class (camel-cased struct name)")

# Make sure at least the first letter is uppercase so go can export it
GROUP_CLASS="${GROUP_CLASS^}"
group_class_lower=${GROUP_CLASS,,}

inform "creating directories"
mkdir -p {apis/${GROUP_NAME},apidocs,hack,${BASE_PATH}/${GROUP_NAME}/{compositions/${COMPOSITION}/templates,v1alpha1,docs,examples}}

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
        -e "s|<GROUP_CLASS_LOWER>|${group_class_lower}|g" \
        -e "s|<BASE_PATH>|${BASE_PATH}|g" \
        -e "s|<REPO_NAME>|${REPO_NAME}|g" \
        template/files/groupversion.go.tpl > "${BASE_PATH}/${GROUP_NAME}/v1alpha1/groupversion.go"
else
    if ! grep -q "${GROUP_CLASS}List" "${BASE_PATH}/${GROUP_NAME}/v1alpha1/groupversion.go"; then
        inform "updating groupversion.go"
        schema="SchemeBuilder.Register(\&${GROUP_CLASS}\{\}, \&${GROUP_CLASS}List\{\})"
        sed -i "s|func init() {|func init() {\n\t$schema|" ${BASE_PATH}/${GROUP_NAME}/v1alpha1/groupversion.go
    fi
fi

if [ ! -f "${BASE_PATH}/${GROUP_NAME}/v1alpha1/${group_class_lower}_types.go" ]; then
    inform "templating ${group_class_lower}_types.go"
    SHORTNAME=$(question "Enter a shortname for the XRD type")

    ENFORCE_COMPOSITION=$(question "Enforce composition?" "yes" "no")
    ENFORCE_COMPOSITION="${ENFORCE_COMPOSITION,,}"

    sed -e "s|<GROUP_NAME>|${GROUP_NAME}|g" \
        -e "s|<GROUP_CLASS>|${GROUP_CLASS}|g" \
        -e "s|<GROUP_CLASS_LOWER>|${GROUP_CLASS,,}|g" \
        -e "s|<SHORTNAME>|${SHORTNAME,,}|g" \
        -e "s|<COMPOSITION>|${COMPOSITION}|g" \
        -e "s|<BASE_PATH>|${BASE_PATH}|g" \
        -e "s|<REPO_NAME>|${REPO_NAME}|g" \
        template/files/xrd.go.tpl > ${BASE_PATH}/${GROUP_NAME}/v1alpha1/${group_class_lower}_types.go

    if [ "${ENFORCE_COMPOSITION:0:1}" = "n" ]; then
        sed -i '/.*enforcedCompositionRef.*/d' ${BASE_PATH}/${GROUP_NAME}/v1alpha1/${group_class_lower}_types.go
    fi
fi

if [ ! -f "apis/${GROUP_NAME}/crossplane.yaml"] ; then
    inform "templating crossplane.yaml"
    xp_version=$(question "please enter the minimum Crossplane version")
    if [ "${xp_version:0:1}" = "v" ]; then
        xp_version="${xp_version:1}"
    fi
    CROSSPLANE_VERSION=">=v${xp_version}"

    query="(.metadata.name = \"${GROUP_NAME}\") \
        | (.metadata.labels.\"pkg.crossplane.io/owner\" = \"${OWNER}\") \
        | (.metadata.labels.\"pkg.crossplane.io/version\" = \"${VERSION}\") \
        | (.spec.crossplane.version = \"${CROSSPLANE_VERSION}\")"
    yq "$query" template/files/crossplane.yaml > apis/${GROUP_NAME}/crossplane.yaml
fi

if [ ! -f hack/boilerplate.go.txt ]; then
    inform "copying boilerplate.go.txt to hack directory for autogen headers"
    cp template/files/boilerplate.go.txt hack
fi

if [ ! -f go.mod ]; then
    inform "setting up go.mod with ${REPO_NAME}"
    go mod init ${REPO_NAME}
    go mod tidy
fi

inform "Running make"
make build
